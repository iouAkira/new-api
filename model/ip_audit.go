package model

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// IP 审计：监控对外分发 API Key 的项目账号（如智能助手项目、数据分析平台），
// 识别黑名单命中 IP 与本月新增 IP，支持处理状态标记与飞书告警。
//
// 三张表：
//   ip_audits          黑名单/白名单条目（支持单 IP、CIDR 网段、闭区间范围三种形式）
//   ip_audit_status    每个 (账号, IP) 的处理状态（新增 IP 确认处理）
//   ip_audit_alert_log 飞书告警静默窗口去重 + 上次告警调用计数快照

const (
	IpAuditTypeBlacklist = 1
	IpAuditTypeWhitelist = 2

	IpAuditStatusPending = 0
	IpAuditStatusHandled = 1

	// Options KV 中的配置键（普通 OptionMap 字符串项，值为 JSON）
	IpAuditAccountsOptionKey = "ip.audit.accounts"
	IpAuditAlertOptionKey    = "ip.audit.alert"
)

type IpAudit struct {
	Id        uint   `json:"id" gorm:"primaryKey"`
	Type      int    `json:"type" gorm:"uniqueIndex:idx_ip_audit_type_ip,priority:1"`
	Ip        string `json:"ip" gorm:"size:64;uniqueIndex:idx_ip_audit_type_ip,priority:2"`
	Remark    string `json:"remark" gorm:"size:255"`
	OperateBy string `json:"operate_by" gorm:"size:64"`
	CreatedAt int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt int64  `json:"updated_at" gorm:"bigint"`
}

func (IpAudit) TableName() string {
	return "ip_audits"
}

type IpAuditStatus struct {
	Id        uint   `json:"id" gorm:"primaryKey"`
	UserId    int    `json:"user_id" gorm:"uniqueIndex:idx_ip_audit_status_user_ip,priority:1"`
	Ip        string `json:"ip" gorm:"size:64;uniqueIndex:idx_ip_audit_status_user_ip,priority:2"`
	Status    int    `json:"status"`
	HandledBy string `json:"handled_by" gorm:"size:64"`
	Note      string `json:"note" gorm:"size:512"`
	HandledAt *int64 `json:"handled_at" gorm:"bigint"`
	CreatedAt int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt int64  `json:"updated_at" gorm:"bigint"`
}

func (IpAuditStatus) TableName() string {
	return "ip_audit_status"
}

type IpAuditAlertLog struct {
	Id         uint   `json:"id" gorm:"primaryKey"`
	AlertKey   string `json:"alert_key" gorm:"size:128;uniqueIndex:uk_ip_audit_alert_key"`
	LastSentAt int64  `json:"last_sent_at" gorm:"bigint"`
	// 上次告警时该行的本月累计调用数快照（计数快照 key 按 token 细化），
	// 下次告警用它计算"较上次新增多少次调用"
	LastCalls int64 `json:"last_calls" gorm:"bigint;default:0"`
}

func (IpAuditAlertLog) TableName() string {
	return "ip_audit_alert_log"
}

// ============ 名单（黑名单/白名单）CRUD ============

// GetIpAuditList 获取某类型的名单条目，按添加时间倒序
func GetIpAuditList(entryType int) ([]IpAudit, error) {
	var entries []IpAudit
	if err := DB.Where("type = ?", entryType).Order("id desc").Find(&entries).Error; err != nil {
		return nil, err
	}
	return entries, nil
}

// AddIpAuditEntry 新增名单条目，校验格式并保证 (type, ip) 唯一
func AddIpAuditEntry(entry *IpAudit) error {
	entry.Ip = strings.TrimSpace(entry.Ip)
	if err := ValidateIpAuditEntry(entry.Ip); err != nil {
		return err
	}
	var count int64
	if err := DB.Model(&IpAudit{}).Where("type = ? AND ip = ?", entry.Type, entry.Ip).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("该 IP 已存在于名单中: %s", entry.Ip)
	}
	now := common.GetTimestamp()
	entry.CreatedAt = now
	entry.UpdatedAt = now
	return DB.Create(entry).Error
}

// DeleteIpAuditEntry 按 ID 删除名单条目
func DeleteIpAuditEntry(id uint) error {
	result := DB.Delete(&IpAudit{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("名单条目不存在")
	}
	return nil
}

// ValidateIpAuditEntry 校验名单条目格式，支持三种形式：
//
//	单个 IPv4:   172.30.1.1
//	CIDR 网段:   172.30.1.0/24
//	闭区间范围:  172.30.1.1-172.30.1.100
func ValidateIpAuditEntry(entry string) error {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return fmt.Errorf("IP 不能为空")
	}
	if strings.Contains(entry, "/") {
		ip, _, err := net.ParseCIDR(entry)
		if err != nil {
			return fmt.Errorf("CIDR 网段格式错误: %s", entry)
		}
		if ip.To4() == nil {
			return fmt.Errorf("仅支持 IPv4 网段: %s", entry)
		}
		return nil
	}
	if idx := strings.Index(entry, "-"); idx > 0 {
		loIp := net.ParseIP(strings.TrimSpace(entry[:idx]))
		hiIp := net.ParseIP(strings.TrimSpace(entry[idx+1:]))
		if loIp == nil || hiIp == nil {
			return fmt.Errorf("IP 范围格式错误: %s", entry)
		}
		lo, loOk := ipAuditIpv4ToUint32(loIp)
		hi, hiOk := ipAuditIpv4ToUint32(hiIp)
		if !loOk || !hiOk {
			return fmt.Errorf("仅支持 IPv4 范围: %s", entry)
		}
		if lo > hi {
			return fmt.Errorf("IP 范围起始不能大于结束: %s", entry)
		}
		return nil
	}
	ip := net.ParseIP(entry)
	if ip == nil || ip.To4() == nil {
		return fmt.Errorf("IP 格式错误: %s", entry)
	}
	return nil
}

// GetIpAuditSet 读取某类型名单的条目集合，key 为条目原始字符串（单 IP/CIDR/范围）
func GetIpAuditSet(entryType int) (map[string]struct{}, error) {
	var entries []IpAudit
	if err := DB.Where("type = ?", entryType).Find(&entries).Error; err != nil {
		return nil, err
	}
	set := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		set[entry.Ip] = struct{}{}
	}
	return set, nil
}

// MatchIpAuditIp 判断 ip 是否命中名单（条目支持单 IP、CIDR、闭区间范围）
func MatchIpAuditIp(ip string, set map[string]struct{}) bool {
	if len(set) == 0 {
		return false
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for entry := range set {
		if ipAuditEntryMatches(parsed, entry) {
			return true
		}
	}
	return false
}

func ipAuditEntryMatches(ip net.IP, entry string) bool {
	if strings.Contains(entry, "/") {
		_, network, err := net.ParseCIDR(entry)
		if err != nil {
			return false
		}
		return network.Contains(ip)
	}
	if idx := strings.Index(entry, "-"); idx > 0 {
		loIp := net.ParseIP(strings.TrimSpace(entry[:idx]))
		hiIp := net.ParseIP(strings.TrimSpace(entry[idx+1:]))
		if loIp == nil || hiIp == nil {
			return false
		}
		lo, loOk := ipAuditIpv4ToUint32(loIp)
		hi, hiOk := ipAuditIpv4ToUint32(hiIp)
		target, targetOk := ipAuditIpv4ToUint32(ip)
		if !loOk || !hiOk || !targetOk {
			return false
		}
		return target >= lo && target <= hi
	}
	target := net.ParseIP(strings.TrimSpace(entry))
	return target != nil && ip.Equal(target)
}

func ipAuditIpv4ToUint32(ip net.IP) (uint32, bool) {
	v4 := ip.To4()
	if v4 == nil {
		return 0, false
	}
	return binary.BigEndian.Uint32(v4), true
}

// ============ (账号, IP) 处理状态 ============

func ipAuditStatusKey(userId int, ip string) string {
	return strconv.Itoa(userId) + "|" + ip
}

// GetIpAuditStatusMap 读取监控账号的处理状态，key 为 "userId|ip"
func GetIpAuditStatusMap(userIds []int) (map[string]IpAuditStatus, error) {
	statusMap := make(map[string]IpAuditStatus)
	if len(userIds) == 0 {
		return statusMap, nil
	}
	var statuses []IpAuditStatus
	if err := DB.Where("user_id IN ?", userIds).Find(&statuses).Error; err != nil {
		return nil, err
	}
	for _, status := range statuses {
		statusMap[ipAuditStatusKey(status.UserId, status.Ip)] = status
	}
	return statusMap, nil
}

// UpsertIpAuditStatus 写入或更新 (账号, IP) 的处理状态，handled=true 表示标记已处理
func UpsertIpAuditStatus(userId int, ip string, handled bool, handledBy string, note string) error {
	now := common.GetTimestamp()
	status := IpAuditStatusPending
	var handledAt *int64
	if handled {
		status = IpAuditStatusHandled
		at := now
		handledAt = &at
	}
	record := IpAuditStatus{
		UserId:    userId,
		Ip:        ip,
		Status:    status,
		HandledBy: handledBy,
		Note:      note,
		HandledAt: handledAt,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "ip"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"status":     status,
			"handled_by": handledBy,
			"note":       note,
			"handled_at": handledAt,
			"updated_at": now,
		}),
	}).Create(&record).Error
}

// ============ 告警静默窗口 ============

// ShouldSendIpAuditAlert 判断告警 key 是否越过了静默窗口（可以发送）
func ShouldSendIpAuditAlert(alertKey string, silenceMinutes int) (bool, error) {
	var record IpAuditAlertLog
	err := DB.Where("alert_key = ?", alertKey).First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return true, nil
		}
		return false, err
	}
	if silenceMinutes <= 0 {
		return true, nil
	}
	return time.Now().Unix()-record.LastSentAt >= int64(silenceMinutes)*60, nil
}

// MarkIpAuditAlertSent 记录告警 key 的最近发送时间
func MarkIpAuditAlertSent(alertKey string) error {
	now := common.GetTimestamp()
	record := IpAuditAlertLog{AlertKey: alertKey, LastSentAt: now}
	return DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "alert_key"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"last_sent_at": now}),
	}).Create(&record).Error
}

// GetIpAuditAlertCalls 读取计数快照 key 上次告警时记录的本月累计调用量
func GetIpAuditAlertCalls(callsKey string) int64 {
	var record IpAuditAlertLog
	if err := DB.Select("last_calls").Where("alert_key = ?", callsKey).First(&record).Error; err != nil {
		return 0
	}
	return record.LastCalls
}

// MarkIpAuditAlertCalls 更新计数快照 key 的本月累计调用量（不影响 last_sent_at）
func MarkIpAuditAlertCalls(callsKey string, calls int64) error {
	record := IpAuditAlertLog{AlertKey: callsKey, LastCalls: calls}
	return DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "alert_key"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"last_calls": calls}),
	}).Create(&record).Error
}

// ============ Options KV 配置 ============

// IpAuditAccount 监控账号（运行时组装：KV 只存 user_id，Username / Name
// 从 users 表实时解析，账号改名后审计展示自动跟随）
type IpAuditAccount struct {
	UserId   int    `json:"user_id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

// IpAuditAlertConfig 飞书告警配置
type IpAuditAlertConfig struct {
	Enabled        bool   `json:"enabled"`
	Webhook        string `json:"webhook"`
	Appkey         string `json:"appkey"`
	TemplateId     string `json:"template_id"`
	EventHit       bool   `json:"event_hit"`
	EventNew       bool   `json:"event_new"`
	SilenceMinutes int    `json:"silence_minutes"`
}

var defaultIpAuditAlertConfig = IpAuditAlertConfig{
	Enabled:        false,
	Webhook:        "",
	Appkey:         "",
	TemplateId:     "",
	EventHit:       true,
	EventNew:       true,
	SilenceMinutes: 1440,
}

func getIpAuditOption(key string) string {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	return common.OptionMap[key]
}

// GetIpAuditAccountIds 读取监控账号 user_id 列表（KV 里存裸 id 数组）
func GetIpAuditAccountIds() []int {
	raw := getIpAuditOption(IpAuditAccountsOptionKey)
	if raw == "" {
		return []int{}
	}
	var ids []int
	if err := common.UnmarshalJsonStr(raw, &ids); err != nil {
		common.SysError("failed to parse option " + IpAuditAccountsOptionKey + ": " + err.Error())
		return []int{}
	}
	return ids
}

// SaveIpAuditAccountIds 保存监控账号 user_id 列表
func SaveIpAuditAccountIds(ids []int) error {
	data, err := common.Marshal(ids)
	if err != nil {
		return err
	}
	return UpdateOption(IpAuditAccountsOptionKey, string(data))
}

// GetIpAuditAccounts 读取监控账号并从 users 表解析展示名
func GetIpAuditAccounts() []IpAuditAccount {
	ids := GetIpAuditAccountIds()
	accounts := make([]IpAuditAccount, 0, len(ids))
	if len(ids) == 0 {
		return accounts
	}
	userNames, err := getIpAuditUserNames(ids)
	if err != nil {
		common.SysError("ip audit: failed to resolve account names: " + err.Error())
	}
	for _, id := range ids {
		username := ""
		name := ""
		if user, ok := userNames[id]; ok {
			username = user.Username
			name = user.DisplayName
			if name == "" {
				name = user.Username
			}
		}
		accounts = append(accounts, IpAuditAccount{UserId: id, Username: username, Name: name})
	}
	return accounts
}

// ResolveIpAuditUserIdsByUsername 按用户账号（username，登录名）批量解析
// user_id；返回与输入同序的 id 列表和未找到的 username 列表
func ResolveIpAuditUserIdsByUsername(usernames []string) ([]int, []string, error) {
	var users []ipAuditUserName
	if err := DB.Table("users").Select("id, username").Where("username IN ?", usernames).Find(&users).Error; err != nil {
		return nil, nil, err
	}
	idByName := make(map[string]int, len(users))
	for _, user := range users {
		idByName[user.Username] = user.Id
	}
	ids := make([]int, 0, len(usernames))
	missing := make([]string, 0)
	for _, name := range usernames {
		if id, ok := idByName[name]; ok {
			ids = append(ids, id)
		} else {
			missing = append(missing, name)
		}
	}
	return ids, missing, nil
}

// GetIpAuditAlertConfig 读取告警配置（未配置时返回默认值）
func GetIpAuditAlertConfig() IpAuditAlertConfig {
	raw := getIpAuditOption(IpAuditAlertOptionKey)
	if raw == "" {
		return defaultIpAuditAlertConfig
	}
	config := defaultIpAuditAlertConfig
	if err := common.UnmarshalJsonStr(raw, &config); err != nil {
		common.SysError("failed to parse option " + IpAuditAlertOptionKey + ": " + err.Error())
		return defaultIpAuditAlertConfig
	}
	return config
}

// SaveIpAuditAlertConfig 保存告警配置
func SaveIpAuditAlertConfig(config IpAuditAlertConfig) error {
	data, err := common.Marshal(config)
	if err != nil {
		return err
	}
	return UpdateOption(IpAuditAlertOptionKey, string(data))
}

// ============ 审计计算（审计视图 + 告警任务共用） ============

// IpAuditRow 审计视图中一行：一个 (账号, Token, IP) 在本月/上月的调用情况
type IpAuditRow struct {
	UserId      int    `json:"user_id"`
	UserName    string `json:"user_name"`
	AccountName string `json:"account_name"`
	TokenName   string `json:"token_name"`
	Ip          string `json:"ip"`
	CurCalls    int64  `json:"cur_calls"`
	CurQuota    int64  `json:"cur_quota"`
	PrevCalls   int64  `json:"prev_calls"`
	PrevQuota   int64  `json:"prev_quota"`
	IsNew       bool   `json:"is_new"`
	White       bool   `json:"white"`
	Hit         bool   `json:"hit"`
	Handled     bool   `json:"handled"`
	HandledBy   string `json:"handled_by"`
	HandledAt   *int64 `json:"handled_at"`
	// 首次/最近出现（本月与上月聚合窗口内最早/最晚的 created_at）
	FirstSeen int64 `json:"first_seen"`
	LastSeen  int64 `json:"last_seen"`
}

// IpAuditStats 审计视图顶部的统计卡片
type IpAuditStats struct {
	Accounts    int   `json:"accounts"`
	Hits        int   `json:"hits"`
	NewPending  int   `json:"new_pending"`
	AbnormalReq int64 `json:"abnormal_req"`
	TotalIp     int   `json:"total_ip"`
}

type ipAuditAggregate struct {
	UserId    int
	TokenName string
	Ip        string
	Username  string
	Calls     int64
	Quota     int64
	FirstSeen int64
	LastSeen  int64
}

type ipAuditUserName struct {
	Id          int
	Username    string
	DisplayName string
}

// parseIpAuditMonth 解析 "YYYY-MM"（空串取当前月），
// 返回 [本月一号零点, 下月一号零点) 的时间范围
func parseIpAuditMonth(month string) (time.Time, time.Time, error) {
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	parsed, err := time.Parse("2006-01", month)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("月份格式错误，应为 YYYY-MM: %s", month)
	}
	start := time.Date(parsed.Year(), parsed.Month(), 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 1, 0)
	return start, end, nil
}

// queryIpAuditAggregates 聚合日志表（type=2 消费日志）中监控账号在
// [start, end) 的调用量与额度，按 (user_id, token_name, ip) 分组
func queryIpAuditAggregates(start int64, end int64, userIds []int) ([]ipAuditAggregate, error) {
	var aggregates []ipAuditAggregate
	err := LOG_DB.Table("logs").
		Select("user_id, token_name, ip, MAX(username) AS username, COUNT(*) AS calls, COALESCE(SUM(quota), 0) AS quota, MIN(created_at) AS first_seen, MAX(created_at) AS last_seen").
		Where("type = ? AND created_at >= ? AND created_at < ?", LogTypeConsume, start, end).
		Where("user_id IN ?", userIds).
		Group("user_id, token_name, ip").
		Find(&aggregates).Error
	if err != nil {
		return nil, err
	}
	return aggregates, nil
}

func getIpAuditUserNames(userIds []int) (map[int]ipAuditUserName, error) {
	var users []ipAuditUserName
	// 用 users 表解析展示名（Table 查询不过滤软删除，账号被删后审计仍能显示名称）
	if err := DB.Table("users").Select("id, username, display_name").Where("id IN ?", userIds).Find(&users).Error; err != nil {
		return nil, err
	}
	userMap := make(map[int]ipAuditUserName, len(users))
	for _, user := range users {
		userMap[user.Id] = user
	}
	return userMap, nil
}

// GetIpAuditRows 计算某月监控账号的审计数据：
// 本月/上月两个聚合查询，Go 内合并后标记 白名单/黑名单命中/新增 IP/处理状态。
// 新增 IP 以 (账号, IP) 为粒度判断：本月出现过而上月没有。
func GetIpAuditRows(month string, accounts []IpAuditAccount) ([]IpAuditRow, error) {
	if len(accounts) == 0 {
		return []IpAuditRow{}, nil
	}
	start, end, err := parseIpAuditMonth(month)
	if err != nil {
		return nil, err
	}
	userIds := make([]int, 0, len(accounts))
	for _, account := range accounts {
		userIds = append(userIds, account.UserId)
	}

	curAggregates, err := queryIpAuditAggregates(start.Unix(), end.Unix(), userIds)
	if err != nil {
		return nil, err
	}
	prevAggregates, err := queryIpAuditAggregates(start.AddDate(0, -1, 0).Unix(), start.Unix(), userIds)
	if err != nil {
		return nil, err
	}
	prevAggs := make(map[string]ipAuditAggregate, len(prevAggregates))
	for _, agg := range prevAggregates {
		prevAggs[ipAuditStatusKey(agg.UserId, agg.Ip)] = agg
	}

	blackSet, err := GetIpAuditSet(IpAuditTypeBlacklist)
	if err != nil {
		return nil, err
	}
	whiteSet, err := GetIpAuditSet(IpAuditTypeWhitelist)
	if err != nil {
		return nil, err
	}
	statusMap, err := GetIpAuditStatusMap(userIds)
	if err != nil {
		return nil, err
	}
	userNames, err := getIpAuditUserNames(userIds)
	if err != nil {
		return nil, err
	}
	// 监控账号名从 users 表实时取（display_name 优先，其次 username）
	accountNames := make(map[int]string, len(userNames))
	for id, user := range userNames {
		if user.DisplayName != "" {
			accountNames[id] = user.DisplayName
		} else {
			accountNames[id] = user.Username
		}
	}

	rows := make([]IpAuditRow, 0, len(curAggregates))
	for _, agg := range curAggregates {
		if agg.Ip == "" {
			continue
		}
		prev, prevSeen := prevAggs[ipAuditStatusKey(agg.UserId, agg.Ip)]
		white := MatchIpAuditIp(agg.Ip, whiteSet)
		hit := MatchIpAuditIp(agg.Ip, blackSet)
		status, hasStatus := statusMap[ipAuditStatusKey(agg.UserId, agg.Ip)]
		handled := hasStatus && status.Status == IpAuditStatusHandled

		firstSeen := agg.FirstSeen
		if prevSeen && prev.FirstSeen > 0 && (firstSeen == 0 || prev.FirstSeen < firstSeen) {
			firstSeen = prev.FirstSeen
		}
		row := IpAuditRow{
			UserId:      agg.UserId,
			UserName:    agg.Username,
			AccountName: accountNames[agg.UserId],
			TokenName:   agg.TokenName,
			Ip:          agg.Ip,
			CurCalls:    agg.Calls,
			CurQuota:    agg.Quota,
			PrevCalls:   prev.Calls,
			PrevQuota:   prev.Quota,
			IsNew:       !prevSeen,
			White:       white,
			Hit:         hit,
			Handled:     handled,
			FirstSeen:   firstSeen,
			LastSeen:    agg.LastSeen,
		}
		// 账号列始终显示用户账号（username，工号/登录名），
		// 不展示 display_name 中文名；日志聚合缺失 username 时从 users 表兜底
		if row.UserName == "" {
			if user, ok := userNames[agg.UserId]; ok {
				row.UserName = user.Username
			}
		}
		if handled {
			row.HandledBy = status.HandledBy
			row.HandledAt = status.HandledAt
		}
		rows = append(rows, row)
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].UserId != rows[j].UserId {
			return rows[i].UserId < rows[j].UserId
		}
		if rows[i].CurCalls != rows[j].CurCalls {
			return rows[i].CurCalls > rows[j].CurCalls
		}
		return rows[i].Ip < rows[j].Ip
	})
	return rows, nil
}

// GetIpAuditStats 基于审计行计算统计卡片：
// 黑名单命中行数 / 新增待处理行数 / 异常行（命中或新增待处理）调用量合计 / 本月去重 (账号,IP) 数
func GetIpAuditStats(accountCount int, rows []IpAuditRow) IpAuditStats {
	stats := IpAuditStats{Accounts: accountCount}
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		seen[ipAuditStatusKey(row.UserId, row.Ip)] = struct{}{}
		if row.Hit {
			stats.Hits++
		}
		newPending := row.IsNew && !row.White && !row.Handled
		if newPending {
			stats.NewPending++
		}
		if row.Hit || newPending {
			stats.AbnormalReq += row.CurCalls
		}
	}
	stats.TotalIp = len(seen)
	return stats
}

// ============ 逐日明细（详情抽屉柱状图） ============

type IpAuditDayPoint struct {
	Day   int   `json:"day"`
	Calls int64 `json:"calls"`
	Quota int64 `json:"quota"`
}

// GetIpAuditDailyDetail 某账号某 IP 在指定月份的逐日调用量，天在 Go 内分桶，
// 避免 strftime/DAY() 等方言专用函数
func GetIpAuditDailyDetail(userId int, ip string, month string) ([]IpAuditDayPoint, error) {
	start, end, err := parseIpAuditMonth(month)
	if err != nil {
		return nil, err
	}
	var points []struct {
		CreatedAt int64
		Quota     int
	}
	err = LOG_DB.Table("logs").
		Select("created_at, quota").
		Where("type = ? AND user_id = ? AND ip = ?", LogTypeConsume, userId, ip).
		Where("created_at >= ? AND created_at < ?", start.Unix(), end.Unix()).
		Find(&points).Error
	if err != nil {
		return nil, err
	}
	days := end.AddDate(0, 0, -1).Day()
	result := make([]IpAuditDayPoint, days)
	for i := range result {
		result[i] = IpAuditDayPoint{Day: i + 1}
	}
	for _, point := range points {
		day := time.Unix(point.CreatedAt, 0).In(time.Local).Day()
		if day >= 1 && day <= days {
			result[day-1].Calls++
			result[day-1].Quota += int64(point.Quota)
		}
	}
	return result, nil
}
