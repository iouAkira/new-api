package service

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// IP 审计飞书告警：
// 移植自 gpu-monitor-service 的 FeishuNotifyService（Java），
// payload 统一用 common.Marshal 构建（替代 Java 手工拼 JSON 字符串）。

// 5 分钟一轮：黑名单命中/新增 IP 最迟几分钟内推送，兼顾聚合查询开销
const ipAuditAlertInterval = 5 * time.Minute

// 告警卡片中的异常类型文案
const (
	IpAuditAlertKindHit = "黑名单命中"
	IpAuditAlertKindNew = "新增IP"
)

// IpAuditAnomaly 一条待告警的异常记录
type IpAuditAnomaly struct {
	Account string
	Ip      string
	Kind    string
	Calls   int64
}

// StartIpAuditAlertTask 定时（ipAuditAlertInterval）扫描监控账号的本月审计数据，
// 发现异常推送飞书告警。仅 master 节点执行。
func StartIpAuditAlertTask() {
	if !common.IsMasterNode {
		return
	}
	go func() {
		RunIpAuditAlertJob()
		ticker := time.NewTicker(ipAuditAlertInterval)
		defer ticker.Stop()
		for range ticker.C {
			RunIpAuditAlertJob()
		}
	}()
}

// RunIpAuditAlertJob 执行一次告警扫描：
// 计算本月异常 -> 按事件配置过滤 -> 静默窗口过滤 -> 合并成一张飞书卡片发送 -> 记录发送时间
func RunIpAuditAlertJob() {
	config := model.GetIpAuditAlertConfig()
	if !config.Enabled || strings.TrimSpace(config.Webhook) == "" {
		return
	}
	accounts := model.GetIpAuditAccounts()
	if len(accounts) == 0 {
		return
	}
	rows, err := model.GetIpAuditRows(time.Now().Format("2006-01"), accounts)
	if err != nil {
		common.SysError("ip audit alert: failed to compute audit rows: " + err.Error())
		return
	}

	type pendingAlert struct {
		key      string
		callsKey string
		curCalls int64
		anomaly  IpAuditAnomaly
	}
	var alerts []pendingAlert
	for _, row := range rows {
		// 已处理（确认过）的行不再告警
		if row.Handled {
			continue
		}
		var key string
		var kind string
		if row.Hit && config.EventHit {
			key = "hit|" + row.Ip
			kind = IpAuditAlertKindHit
		} else if row.IsNew && !row.White && config.EventNew {
			key = fmt.Sprintf("new|%d|%s", row.UserId, row.Ip)
			kind = IpAuditAlertKindNew
		} else {
			continue
		}
		shouldSend, err := model.ShouldSendIpAuditAlert(key, config.SilenceMinutes)
		if err != nil {
			common.SysError("ip audit alert: failed to check silence window for " + key + ": " + err.Error())
			continue
		}
		if !shouldSend {
			continue
		}
		// 调用量按"较上次告警新增"计算：计数快照 key 细化到令牌
		// （一次告警可能含同 IP 多令牌多行，静默 key 只按 IP 去重）
		callsKey := key + "|t|" + row.TokenName
		delta := row.CurCalls - model.GetIpAuditAlertCalls(callsKey)
		if delta < 0 {
			// 跨月累计归零或快照异常：全部计为本期新增
			delta = row.CurCalls
		}
		if delta == 0 {
			// 静默期过后没有新增调用（如黑名单 IP 已停止调用）：无事态变化，不再打扰
			continue
		}
		// 告警卡片同样显示用户账号（username，工号），而非 display_name
		account := row.UserName
		if account == "" {
			account = row.AccountName
		}
		if account == "" {
			account = fmt.Sprintf("%d", row.UserId)
		}
		alerts = append(alerts, pendingAlert{
			key:      key,
			callsKey: callsKey,
			curCalls: row.CurCalls,
			anomaly:  IpAuditAnomaly{Account: account, Ip: row.Ip, Kind: kind, Calls: delta},
		})
	}
	if len(alerts) == 0 {
		return
	}

	anomalies := make([]IpAuditAnomaly, 0, len(alerts))
	for _, alert := range alerts {
		anomalies = append(anomalies, alert.anomaly)
	}
	title := "IP 审计异常告警 " + time.Now().Format("2006-01-02 15:04")
	payload, err := BuildFeishuPayload(title, BuildIpAuditMarkdown(anomalies), config.TemplateId)
	if err != nil {
		common.SysError("ip audit alert: failed to build payload: " + err.Error())
		return
	}
	respBody, err := SendFeishu(config.Webhook, config.Appkey, payload)
	if err != nil {
		common.SysError("ip audit alert: failed to send feishu message: " + err.Error())
		return
	}
	for _, alert := range alerts {
		if err := model.MarkIpAuditAlertSent(alert.key); err != nil {
			common.SysError("ip audit alert: failed to mark sent for " + alert.key + ": " + err.Error())
		}
		if err := model.MarkIpAuditAlertCalls(alert.callsKey, alert.curCalls); err != nil {
			common.SysError("ip audit alert: failed to mark calls for " + alert.callsKey + ": " + err.Error())
		}
	}
	common.SysLog(fmt.Sprintf("ip audit alert sent: %d anomalies, response: %s", len(alerts), respBody))
}

// BuildIpAuditMarkdown 构建告警卡片的 markdown 内容（标题 + 时间 + 异常明细表）
func BuildIpAuditMarkdown(anomalies []IpAuditAnomaly) string {
	now := time.Now().Format("2006-01-02 15:04:05")
	var md strings.Builder
	md.WriteString("## 🚨 IP 审计异常告警\n\n")
	md.WriteString("**告警时间**: " + now + "\n")
	md.WriteString(fmt.Sprintf("**异常数量**: %d\n\n", len(anomalies)))
	md.WriteString("| 账号 | IP | 类型 | 新增调用 |\n")
	md.WriteString("|---|---|---|---|\n")
	for _, anomaly := range anomalies {
		md.WriteString(fmt.Sprintf("| %s | %s | %s | %d |\n", anomaly.Account, anomaly.Ip, anomaly.Kind, anomaly.Calls))
	}
	return md.String()
}

// BuildFeishuPayload 构建飞书卡片消息体：
// 配置了模板 ID 时使用模板卡片，否则使用普通 lark_md 卡片（红色标题栏）
func BuildFeishuPayload(title string, md string, templateId string) ([]byte, error) {
	return buildIpAuditFeishuCard(title, md, templateId, "red")
}

func buildIpAuditFeishuCard(title string, md string, templateId string, headerTemplate string) ([]byte, error) {
	templateId = strings.TrimSpace(templateId)
	if templateId != "" {
		return common.Marshal(map[string]interface{}{
			"msg_type": "interactive",
			"card": map[string]interface{}{
				"type": "template",
				"data": map[string]interface{}{
					"template_id": templateId,
					"template_variable": map[string]interface{}{
						"title":    title,
						"contents": md,
					},
				},
			},
		})
	}
	return common.Marshal(map[string]interface{}{
		"msg_type": "interactive",
		"card": map[string]interface{}{
			"elements": []interface{}{
				map[string]interface{}{
					"tag": "div",
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": md,
					},
				},
			},
			"header": map[string]interface{}{
				"title": map[string]interface{}{
					"tag":     "plain_text",
					"content": title,
				},
				"template": headerTemplate,
			},
		},
	})
}

// SendFeishu POST 消息体到飞书 webhook，返回响应正文（含非 2xx 时的错误正文）。
// 连接超时 10s、读超时 30s；配置了 appkey 时附带小写 appkey 请求头（iPaaS 网关代理）。
func SendFeishu(webhook string, appkey string, payload []byte) (string, error) {
	webhook = strings.TrimSpace(webhook)
	if webhook == "" {
		return "", fmt.Errorf("飞书 webhook 未配置")
	}
	req, err := http.NewRequest(http.MethodPost, webhook, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if strings.TrimSpace(appkey) != "" {
		req.Header.Set("appkey", strings.TrimSpace(appkey))
	}
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		},
		Timeout: 40 * time.Second, // 10s 连接 + 30s 读取
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return string(body), fmt.Errorf("飞书接口返回状态码 %d", resp.StatusCode)
	}
	return string(body), nil
}

// SendIpAuditAlertTest 发送一条测试消息（蓝色标题栏，不走静默窗口），返回飞书响应正文
func SendIpAuditAlertTest(config model.IpAuditAlertConfig) (string, error) {
	if strings.TrimSpace(config.Webhook) == "" {
		return "", fmt.Errorf("飞书 webhook 未配置")
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	md := "## ✅ IP 审计测试消息\n\n" +
		"这是一条来自 IP 审计功能的测试消息，收到本消息说明 Webhook 配置正确。\n\n" +
		"**发送时间**: " + now
	title := "IP 审计测试消息 " + now
	payload, err := buildIpAuditFeishuCard(title, md, config.TemplateId, "blue")
	if err != nil {
		return "", err
	}
	return SendFeishu(config.Webhook, config.Appkey, payload)
}
