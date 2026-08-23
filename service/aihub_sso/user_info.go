package aihubsso

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// UserInfo 是 AI Hub 用户信息接口返回的用户资料。
type UserInfo struct {
	UserNo      string   `json:"userNo"`
	UserName    string   `json:"userName"`
	OrgBizID    string   `json:"orgBizId"`
	OrgName     string   `json:"orgName"`
	Position    string   `json:"position"`
	Status      string   `json:"status"`
	Email       string   `json:"email"`
	Sex         string   `json:"sex"`
	OrgBizChain []string `json:"orgBizChain"`
}

// FetchUserInfo 先按工号获取访问令牌，再查询该工号对应的用户资料。
func FetchUserInfo(ctx context.Context, employNo string, cfg Config) (*UserInfo, error) {
	employNo = strings.TrimSpace(employNo)
	if employNo == "" {
		return nil, ErrInvalid
	}
	if cfg.AccessTokenURL == "" {
		return nil, fmt.Errorf("%w: missing access token url", ErrConfig)
	}
	if cfg.UserInfoURL == "" {
		return nil, fmt.Errorf("%w: missing user info url", ErrConfig)
	}

	requestBody, err := common.Marshal(struct {
		EmployNo  string `json:"employNo"`
		AppID     string `json:"appId"`
		AppSecret string `json:"appSecret"`
	}{
		EmployNo:  employNo,
		AppID:     cfg.AppID,
		AppSecret: cfg.AppSecret,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConfig, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.AccessTokenURL, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConfig, err)
	}
	req.Header.Set("Content-Type", "application/json")

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
		return nil, fmt.Errorf("%w: access token http status %d", ErrRequestFailed, resp.StatusCode)
	}

	var tokenResponse struct {
		Code    int    `json:"code"`
		Status  string `json:"status"`
		Message string `json:"message"`
		Data    *struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
		RespMsg string `json:"resp_msg"`
	}
	if err := common.Unmarshal(body, &tokenResponse); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	if tokenResponse.Code != http.StatusOK || !strings.EqualFold(tokenResponse.Status, "success") || tokenResponse.Data == nil || strings.TrimSpace(tokenResponse.Data.AccessToken) == "" {
		message := strings.TrimSpace(tokenResponse.RespMsg)
		if message == "" {
			message = strings.TrimSpace(tokenResponse.Message)
		}
		if message != "" {
			return nil, fmt.Errorf("%w: %s", ErrInvalid, message)
		}
		return nil, ErrInvalid
	}

	userInfoURL, err := url.Parse(cfg.UserInfoURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConfig, err)
	}
	query := userInfoURL.Query()
	query.Set("userNo", employNo)
	userInfoURL.RawQuery = query.Encode()

	req, err = http.NewRequestWithContext(ctx, http.MethodGet, userInfoURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConfig, err)
	}
	req.Header.Set("token", tokenResponse.Data.AccessToken)

	resp, err = client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRequestFailed, err)
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRequestFailed, err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("%w: user info http status %d", ErrRequestFailed, resp.StatusCode)
	}

	var userInfoResponse struct {
		Code    int       `json:"code"`
		Status  string    `json:"status"`
		Message string    `json:"message"`
		Data    *UserInfo `json:"data"`
		RespMsg string    `json:"resp_msg"`
	}
	if err := common.Unmarshal(body, &userInfoResponse); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	if userInfoResponse.Code != http.StatusOK || !strings.EqualFold(userInfoResponse.Status, "success") || userInfoResponse.Data == nil || strings.TrimSpace(userInfoResponse.Data.UserName) == "" {
		message := strings.TrimSpace(userInfoResponse.RespMsg)
		if message == "" {
			message = strings.TrimSpace(userInfoResponse.Message)
		}
		if message != "" {
			return nil, fmt.Errorf("%w: %s", ErrInvalid, message)
		}
		return nil, ErrInvalid
	}

	return userInfoResponse.Data, nil
}
