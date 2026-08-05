package model

import (
	"database/sql"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// DeploymentNode 是一条部署记录：某模型部署在某台机器的某个端口上。
// 数据全部来自 new-api 自身的库（channels + models），不依赖外部 Doris / GPU 监控服务。
//
// 关联逻辑（与运维侧 Doris 脚本一致）：
//  1. 取启用渠道 channels.status = 1
//  2. 渠道的 models 字段是逗号分隔的多个模型 -> 炸裂成单条 (渠道, 模型)
//  3. JOIN models 表（model_name）用 tags 区分内部(私有化部署)/外部(外部API)
//  4. 节点 URL 来自 channels.base_url + channels.remark（remark 存负载均衡多节点，逗号分隔）
//  5. 从每段 URL 用正则抽出 IPv4 与端口
type DeploymentNode struct {
	Model         string `json:"model"`
	IP            string `json:"ip"`
	Port          string `json:"port"`
	ChannelID     int    `json:"channel_id"`
	ChannelName   string `json:"channel_name"`
	Key           string `json:"key"` // 渠道 API Key（私有大模型密钥）
	ChannelStatus int    `json:"channel_status"` // 1=启用 2=手动禁用 3=自动禁用
	Endpoint      string `json:"endpoint"`       // 解析出 ip:port 的原始 URL 片段
}

var (
	deploymentIPRe   = regexp.MustCompile(`https?://((?:[0-9]{1,3}[.]){3}[0-9]{1,3})`)
	deploymentPortRe = regexp.MustCompile(`https?://(?:[0-9]{1,3}[.]){3}[0-9]{1,3}:([0-9]+)`)
)

// GetDeploymentNodes 计算内部（私有化部署）模型的部署节点列表。
//
// tag 为过滤标签（默认"私有化部署"）：只保留 models.tags 命中该标签的模型；
// 传空串则不做标签过滤，返回所有模型的节点。
func GetDeploymentNodes(tag string) ([]DeploymentNode, error) {
	// 1. models 表：model_name -> tags，用于区分内部/外部
	type mRow struct {
		ModelName string
		Tags      sql.NullString
	}
	var mRows []mRow
	if err := DB.Model(&Model{}).
		Select("model_name, tags").
		Scan(&mRows).Error; err != nil {
		return nil, err
	}
	tagOf := make(map[string]string, len(mRows))
	for _, m := range mRows {
		tagOf[m.ModelName] = m.Tags.String
	}

	// 2. 启用渠道 status = 1
	type cRow struct {
		Id      int
		Name    sql.NullString
		BaseURL sql.NullString
		Remark  sql.NullString
		Models  sql.NullString
		Key     sql.NullString
		Status  int
	}
	var cRows []cRow
	if err := DB.Table("channels").
		Select("id, name, base_url, remark, models, `key`, status").
		Where("status = ?", common.ChannelStatusEnabled).
		Scan(&cRows).Error; err != nil {
		return nil, err
	}

	// 3. 炸裂 models + 按 models.tags 区分内外部 + 解析 IP:port
	nodes := make([]DeploymentNode, 0)
	seen := make(map[string]struct{})
	for _, ch := range cRows {
		modelsStr := ch.Models.String
		if modelsStr == "" {
			continue
		}
		for _, model := range strings.Split(modelsStr, ",") {
			model = strings.TrimSpace(model)
			if model == "" {
				continue
			}
			// models 表区分内外部：命中标签才保留
			mTags, ok := tagOf[model]
			if !ok {
				continue
			}
			if tag != "" && !strings.Contains(mTags, tag) {
				continue
			}

			// 节点 URL = base_url + remark（负载均衡多节点）
			combined := ch.BaseURL.String
			if rm := ch.Remark.String; rm != "" {
				if combined != "" {
					combined += ","
				}
				combined += rm
			}
			for _, seg := range strings.Split(combined, ",") {
				seg = strings.TrimSpace(seg)
				if seg == "" {
					continue
				}
				ipMatch := deploymentIPRe.FindStringSubmatch(seg)
				if len(ipMatch) < 2 || ipMatch[1] == "" {
					continue // 非 IPv4 的段（域名等）跳过
				}
				ip := ipMatch[1]
				port := ""
				if portMatch := deploymentPortRe.FindStringSubmatch(seg); len(portMatch) >= 2 {
					port = portMatch[1]
				}
				key := model + "|" + ip + "|" + port
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				nodes = append(nodes, DeploymentNode{
					Model:         model,
					IP:            ip,
					Port:          port,
					ChannelID:     ch.Id,
					ChannelName:   ch.Name.String,
					Key:           ch.Key.String,
					ChannelStatus: ch.Status,
					Endpoint:      seg,
				})
			}
		}
	}

	return nodes, nil
}
