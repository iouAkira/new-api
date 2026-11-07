package controller

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// GetIpAuditRecords 审计视图主查询：合并名单、处理状态后的 (账号, Token, IP) 月度聚合行，
// 支持账号/状态/关键词筛选与 Go 内分页，统计卡片不受筛选影响
// loadFilteredIpAuditRows 按 month/account/status/keyword 查询并过滤审计行。
// 统计卡基于 account/month 维度的全量行（不受 status/keyword 影响），
// 与列表页/CSV 导出共用。
func loadFilteredIpAuditRows(c *gin.Context) ([]model.IpAuditRow, model.IpAuditStats, error) {
	statusFilter := c.Query("status")
	if statusFilter == "" {
		statusFilter = "all"
	}
	switch statusFilter {
	case "all", "hit", "new", "handled", "white", "exist":
	default:
		return nil, model.IpAuditStats{}, fmt.Errorf("无效的 status 筛选值")
	}

	allAccounts := model.GetIpAuditAccounts()
	accountFilter := c.Query("account")
	selected := make([]model.IpAuditAccount, 0, len(allAccounts))
	for _, account := range allAccounts {
		if accountFilter == "" || accountFilter == "all" || accountFilter == strconv.Itoa(account.UserId) {
			selected = append(selected, account)
		}
	}

	rows, err := model.GetIpAuditRows(c.Query("month"), selected)
	if err != nil {
		return nil, model.IpAuditStats{}, err
	}
	stats := model.GetIpAuditStats(len(allAccounts), rows)

	keyword := strings.ToLower(strings.TrimSpace(c.Query("keyword")))
	filtered := make([]model.IpAuditRow, 0, len(rows))
	for _, row := range rows {
		// 状态标签优先级：黑名单命中 > 白名单 > 新增已处理 > 新增待处理 > 存量
		status := "exist"
		switch {
		case row.Hit:
			status = "hit"
		case row.White:
			status = "white"
		case row.Handled:
			status = "handled"
		case row.IsNew:
			status = "new"
		}
		if statusFilter != "all" && status != statusFilter {
			continue
		}
		if keyword != "" &&
			!strings.Contains(strings.ToLower(row.Ip), keyword) &&
			!strings.Contains(strings.ToLower(row.TokenName), keyword) &&
			!strings.Contains(strings.ToLower(row.UserName), keyword) &&
			!strings.Contains(strings.ToLower(row.AccountName), keyword) {
			continue
		}
		filtered = append(filtered, row)
	}
	return filtered, stats, nil
}

func GetIpAuditRecords(c *gin.Context) {
	filtered, stats, err := loadFilteredIpAuditRows(c)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	pageInfo := common.GetPageQuery(c)
	// 前端按 ?page=N 传页码，仓库通用分页助手只识别 ?p=N，这里兼容两者
	if pageParam := c.Query("page"); pageParam != "" {
		if page, err := strconv.Atoi(pageParam); err == nil && page > 0 {
			pageInfo.Page = page
		}
	}
	pageRows := filtered
	if pageInfo.GetStartIdx() >= len(filtered) {
		pageRows = make([]model.IpAuditRow, 0)
	} else {
		endIdx := pageInfo.GetEndIdx()
		if endIdx > len(filtered) {
			endIdx = len(filtered)
		}
		pageRows = filtered[pageInfo.GetStartIdx():endIdx]
	}

	common.ApiSuccess(c, gin.H{
		"rows":      pageRows,
		"total":     len(filtered),
		"page":      pageInfo.GetPage(),
		"page_size": pageInfo.GetPageSize(),
		"stats":     stats,
	})
}

// ExportIpAuditAuditCsv 按当前筛选导出审计行 CSV（UTF-8 BOM，Excel 可直接打开）
func ExportIpAuditAuditCsv(c *gin.Context) {
	rows, _, err := loadFilteredIpAuditRows(c)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	var buf bytes.Buffer
	buf.WriteString("\ufeff") // UTF-8 BOM, so Excel opens it without garbled Chinese
	writer := csv.NewWriter(&buf)
	_ = writer.Write([]string{"账号", "用户ID", "令牌", "IP", "状态", "本月调用", "消费额度($)", "上月调用", "首次出现", "最新调用", "处理人", "处理时间"})
	for _, row := range rows {
		status := "存量"
		switch {
		case row.Hit:
			status = "黑名单命中"
		case row.White:
			status = "白名单"
		case row.Handled:
			status = "新增·已处理"
		case row.IsNew:
			status = "新增待处理"
		}
		handledAt := ""
		if row.HandledAt != nil && *row.HandledAt > 0 {
			handledAt = time.Unix(*row.HandledAt, 0).Format("2006-01-02 15:04")
		}
		firstSeen := ""
		if row.FirstSeen > 0 {
			firstSeen = time.Unix(row.FirstSeen, 0).Format("2006-01-02")
		}
		lastSeen := ""
		if row.LastSeen > 0 {
			lastSeen = time.Unix(row.LastSeen, 0).Format("2006-01-02 15:04")
		}
		_ = writer.Write([]string{
			row.AccountName,
			strconv.Itoa(row.UserId),
			row.TokenName,
			row.Ip,
			status,
			strconv.FormatInt(row.CurCalls, 10),
			strconv.FormatFloat(float64(row.CurQuota)/common.QuotaPerUnit, 'f', -1, 64),
			strconv.FormatInt(row.PrevCalls, 10),
			firstSeen,
			lastSeen,
			row.HandledBy,
			handledAt,
		})
	}
	writer.Flush()

	month := c.Query("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	recordManageAudit(c, "ip_audit.export", map[string]interface{}{"month": month, "rows": len(rows)})
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"ip_audit_%s.csv\"", month))
	c.String(http.StatusOK, "%s", buf.String())
}

// GetIpAuditRecordsDetail 详情页：按天聚合调用量/额度，附首次出现时间与处理状态
func GetIpAuditRecordsDetail(c *gin.Context) {
	userId, err := strconv.Atoi(c.Query("user_id"))
	if err != nil || userId <= 0 {
		common.ApiErrorMsg(c, "无效的 user_id")
		return
	}
	ip := strings.TrimSpace(c.Query("ip"))
	if ip == "" {
		common.ApiErrorMsg(c, "无效的 ip")
		return
	}

	days, err := model.GetIpAuditDailyDetail(userId, ip, c.Query("month"))
	if err != nil {
		common.ApiError(c, err)
		return
	}

	var firstSeen struct {
		FirstSeen int64
	}
	err = model.LOG_DB.Table("logs").
		Select("COALESCE(MIN(created_at), 0) as first_seen").
		Where("type = ? AND user_id = ? AND ip = ?", model.LogTypeConsume, userId, ip).
		Scan(&firstSeen).Error
	if err != nil {
		common.ApiError(c, err)
		return
	}

	data := gin.H{
		"days":       days,
		"first_seen": firstSeen.FirstSeen,
	}
	statusMap, err := model.GetIpAuditStatusMap([]int{userId})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if status, ok := statusMap[strconv.Itoa(userId)+"|"+ip]; ok {
		data["status"] = status
	}
	common.ApiSuccess(c, data)
}

// UpdateIpAuditStatus 标记/取消标记某 (账号, IP) 的处理状态
func UpdateIpAuditStatus(c *gin.Context) {
	var req struct {
		UserId int    `json:"user_id"`
		Ip     string `json:"ip"`
		Action string `json:"action"`
		Note   string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.UserId <= 0 {
		common.ApiErrorMsg(c, "user_id 必须大于 0")
		return
	}
	req.Ip = strings.TrimSpace(req.Ip)
	if req.Ip == "" {
		common.ApiErrorMsg(c, "ip 不能为空")
		return
	}
	if req.Action != "handle" && req.Action != "reopen" {
		common.ApiErrorMsg(c, "action 必须为 handle 或 reopen")
		return
	}
	handled := req.Action == "handle"
	if err := model.UpsertIpAuditStatus(req.UserId, req.Ip, handled, c.GetString("username"), req.Note); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "ip_audit.status", map[string]interface{}{
		"user_id": req.UserId,
		"ip":      req.Ip,
		"action":  req.Action,
		"note":    req.Note,
	})
	common.ApiSuccess(c, nil)
}

// GetIpAuditLists 名单查询，type=1 黑名单 / type=2 白名单
func GetIpAuditLists(c *gin.Context) {
	listType, err := strconv.Atoi(c.Query("type"))
	if err != nil || (listType != model.IpAuditTypeBlacklist && listType != model.IpAuditTypeWhitelist) {
		common.ApiErrorMsg(c, "type 必须为 1（黑名单）或 2（白名单）")
		return
	}
	entries, err := model.GetIpAuditList(listType)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, entries)
}

// CreateIpAuditList 新增黑名单/白名单条目，支持单个 IPv4、CIDR、闭区间三种格式
// （格式校验与 (type, ip) 唯一性由 model 层保证）
func CreateIpAuditList(c *gin.Context) {
	var req struct {
		Type   int    `json:"type"`
		Ip     string `json:"ip"`
		Remark string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.Type != model.IpAuditTypeBlacklist && req.Type != model.IpAuditTypeWhitelist {
		common.ApiErrorMsg(c, "type 必须为 1（黑名单）或 2（白名单）")
		return
	}
	entry := &model.IpAudit{
		Type:      req.Type,
		Ip:        req.Ip,
		Remark:    req.Remark,
		OperateBy: c.GetString("username"),
	}
	if err := model.AddIpAuditEntry(entry); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "ip_audit.list.create", map[string]interface{}{
		"type": req.Type,
		"ip":   entry.Ip,
	})
	common.ApiSuccess(c, entry)
}

// DeleteIpAuditList 删除名单条目
func DeleteIpAuditList(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		common.ApiErrorMsg(c, "无效的 id")
		return
	}
	if err := model.DeleteIpAuditEntry(uint(id)); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "ip_audit.list.delete", map[string]interface{}{"id": id})
	common.ApiSuccess(c, nil)
}

// GetIpAuditConfig 读取监控账号与告警配置
func GetIpAuditConfig(c *gin.Context) {
	common.ApiSuccess(c, gin.H{
		"accounts": model.GetIpAuditAccounts(),
		"alert":    model.GetIpAuditAlertConfig(),
	})
}

// UpdateIpAuditConfig 校验并保存监控账号与告警配置。
// usernames 为 nil 表示本次不修改账号列表（告警设置页只存 alert）；
// 页面按用户账号（username 登录名）添加，后端解析成 user_id 入库，
// 展示名从 users 表实时解析 display_name。
func UpdateIpAuditConfig(c *gin.Context) {
	var req struct {
		Usernames []string                  `json:"usernames"`
		Alert     *model.IpAuditAlertConfig `json:"alert"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	if req.Usernames != nil {
		seen := make(map[string]bool, len(req.Usernames))
		usernames := make([]string, 0, len(req.Usernames))
		for _, name := range req.Usernames {
			name = strings.TrimSpace(name)
			if name == "" {
				common.ApiErrorMsg(c, "用户账号不能为空")
				return
			}
			if seen[name] {
				common.ApiErrorMsg(c, fmt.Sprintf("用户账号 %s 重复", name))
				return
			}
			seen[name] = true
			usernames = append(usernames, name)
		}
		ids, missing, err := model.ResolveIpAuditUserIdsByUsername(usernames)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if len(missing) > 0 {
			common.ApiErrorMsg(c, fmt.Sprintf("用户账号 %s 不存在", missing[0]))
			return
		}
		if err := model.SaveIpAuditAccountIds(ids); err != nil {
			common.ApiError(c, err)
			return
		}
	}

	alert := model.GetIpAuditAlertConfig()
	if req.Alert != nil {
		alert = *req.Alert
	}
	alert.Webhook = strings.TrimSpace(alert.Webhook)
	alert.Appkey = strings.TrimSpace(alert.Appkey)
	alert.TemplateId = strings.TrimSpace(alert.TemplateId)
	if alert.Webhook != "" && !strings.HasPrefix(alert.Webhook, "http://") && !strings.HasPrefix(alert.Webhook, "https://") {
		common.ApiErrorMsg(c, "Webhook 仅支持 http/https 地址")
		return
	}
	if alert.Enabled && alert.Webhook == "" {
		common.ApiErrorMsg(c, "启用告警时必须填写 Webhook 地址")
		return
	}
	if alert.SilenceMinutes <= 0 {
		// 与 model 默认告警配置的静默期保持一致（24 小时）
		alert.SilenceMinutes = 1440
	}

	if err := model.SaveIpAuditAlertConfig(alert); err != nil {
		common.ApiError(c, err)
		return
	}
	accounts := model.GetIpAuditAccounts()
	recordManageAudit(c, "ip_audit.config.update", map[string]interface{}{
		"accounts":      len(accounts),
		"alert_enabled": alert.Enabled,
	})
	common.ApiSuccess(c, gin.H{
		"accounts": accounts,
		"alert":    alert,
	})
}

// TestIpAuditAlert 用当前（或请求体覆盖的）webhook 配置发送一条飞书测试消息
func TestIpAuditAlert(c *gin.Context) {
	var req struct {
		Webhook    string `json:"webhook"`
		Appkey     string `json:"appkey"`
		TemplateId string `json:"template_id"`
	}
	// 请求体可省略，省略时使用已保存的配置
	c.ShouldBindJSON(&req)

	alert := model.GetIpAuditAlertConfig()
	if webhook := strings.TrimSpace(req.Webhook); webhook != "" {
		alert.Webhook = webhook
	}
	if appkey := strings.TrimSpace(req.Appkey); appkey != "" {
		alert.Appkey = appkey
	}
	if templateId := strings.TrimSpace(req.TemplateId); templateId != "" {
		alert.TemplateId = templateId
	}
	alert.Webhook = strings.TrimSpace(alert.Webhook)
	if alert.Webhook == "" {
		common.ApiErrorMsg(c, "尚未配置 Webhook 地址")
		return
	}
	if !strings.HasPrefix(alert.Webhook, "http://") && !strings.HasPrefix(alert.Webhook, "https://") {
		common.ApiErrorMsg(c, "Webhook 仅支持 http/https 地址")
		return
	}
	body, err := service.SendIpAuditAlertTest(alert)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, body)
}
