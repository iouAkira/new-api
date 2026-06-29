package aihubsso

import (
	"strings"
)

// NormalizeToken 兼容文档推荐的裸 token 和历史 Bearer 写法。
func NormalizeToken(token string) string {
	token = strings.TrimSpace(token)
	if len(token) >= 7 && strings.EqualFold(token[:7], "Bearer ") {
		return strings.TrimSpace(token[7:])
	}
	return token
}

// ValidateVerificationResponse 在建立本地会话前执行必要的身份与应用范围校验。
func ValidateVerificationResponse(resp *VerificationResponse, cfg Config) error {
	if resp == nil {
		return ErrInvalid
	}
	if resp.Code != 200 || !strings.EqualFold(resp.Status, "success") || !resp.Data.Valid || strings.TrimSpace(resp.Data.EmployNo) == "" {
		return ErrInvalid
	}

	if !cfg.RequireAppCheck {
		return nil
	}
	appID := resp.Data.AppID
	if appID == "" {
		appID = resp.AppID
	}
	appSecret := resp.Data.AppSecret
	if appSecret == "" {
		appSecret = resp.AppSecret
	}

	if cfg.AppID != "" && appID != "" && appID != cfg.AppID {
		return ErrAppMismatch
	}
	if cfg.AppSecret != "" && appSecret != "" && appSecret != cfg.AppSecret {
		return ErrAppMismatch
	}
	return nil
}

// MaskToken 用于联调日志脱敏，避免暴露完整 SSO token。
func MaskToken(token string) string {
	token = NormalizeToken(token)
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}
