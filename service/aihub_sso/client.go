package aihubsso

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/QuantumNous/new-api/common"
)

var (
	ErrConfig        = errors.New("ai hub sso config error")
	ErrInvalid       = errors.New("ai hub sso token invalid")
	ErrAppMismatch   = errors.New("ai hub sso app mismatch")
	ErrRequestFailed = errors.New("ai hub sso verification request failed")
)

// VerificationResponse 兼容 appId/appSecret 位于顶层或 data 内的两种返回结构。
type VerificationResponse struct {
	Code      int              `json:"code"`
	Status    string           `json:"status"`
	Data      VerificationData `json:"data"`
	AppID     string           `json:"appId"`
	AppSecret string           `json:"appSecret"`
}

// VerificationData 是 tokenVerification 返回的身份载荷。
type VerificationData struct {
	Valid     bool   `json:"valid"`
	EmployNo  string `json:"employNo"`
	UserName  string `json:"userName"`
	AppID     string `json:"appId"`
	AppSecret string `json:"appSecret"`
}

// VerifyToken 按 AI Hub 文档使用 GET 调用 tokenVerification，并通过 Header 传 token。
func VerifyToken(ctx context.Context, token string, cfg Config) (*VerificationResponse, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("%w: disabled", ErrConfig)
	}
	if cfg.VerificationURL == "" {
		return nil, fmt.Errorf("%w: missing verification url", ErrConfig)
	}

	normalizedToken := NormalizeToken(token)
	if normalizedToken == "" {
		return nil, ErrInvalid
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.VerificationURL, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConfig, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("token", normalizedToken)
	req.Header.Set("appId", cfg.AppID)
	req.Header.Set("appSecret", cfg.AppSecret)

	client := &http.Client{Timeout: cfg.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRequestFailed, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRequestFailed, err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("%w: http status %d", ErrRequestFailed, resp.StatusCode)
	}

	var verification VerificationResponse
	if err := common.Unmarshal(body, &verification); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	if err := ValidateVerificationResponse(&verification, cfg); err != nil {
		return nil, err
	}
	return &verification, nil
}
