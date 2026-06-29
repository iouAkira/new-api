package aihub_sso

import (
	"strings"
)

func NormalizeToken(token string) string {
	token = strings.TrimSpace(token)
	if len(token) >= 7 && strings.EqualFold(token[:7], "Bearer ") {
		return strings.TrimSpace(token[7:])
	}
	return token
}

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

func MaskToken(token string) string {
	token = NormalizeToken(token)
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}
