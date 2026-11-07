package aihubsso

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyTokenPostsCredentialsInJSONBody(t *testing.T) {
	t.Parallel()

	type verificationRequest struct {
		Token     string `json:"token"`
		AppID     string `json:"appId"`
		AppSecret string `json:"appSecret"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Empty(t, r.Header.Get("token"))
		assert.Empty(t, r.Header.Get("appId"))
		assert.Empty(t, r.Header.Get("appSecret"))

		var body verificationRequest
		require.NoError(t, common.DecodeJson(r.Body, &body))
		assert.Equal(t, verificationRequest{
			Token:     "token+value",
			AppID:     "app-id",
			AppSecret: "app-secret",
		}, body)

		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"code":200,"status":"success","message":"操作成功","data":{"expiresIn":null,"employNo":"2205090261","userName":null,"userId":null,"appId":null,"appSecret":null,"agentId":null,"valid":null,"access_token":null}}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	verification, err := VerifyToken(context.Background(), "Bearer token value", Config{
		Enabled:         true,
		VerificationURL: server.URL,
		AppID:           "app-id",
		AppSecret:       "app-secret",
		Timeout:         time.Second,
	})

	require.NoError(t, err)
	require.NotNil(t, verification)
	assert.Equal(t, "2205090261", verification.Data.EmployNo)
}

func TestVerifyTokenRejectsFailureResponses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		body    string
		respMsg string
	}{
		{
			name:    "token does not exist",
			body:    `{"datas":null,"resp_code":0,"resp_msg":"当前token不存在"}`,
			respMsg: "当前token不存在",
		},
		{
			name:    "token is incorrect",
			body:    `{"datas":null,"resp_code":0,"resp_msg":"当前token不正确"}`,
			respMsg: "当前token不正确",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, err := w.Write([]byte(tt.body))
				require.NoError(t, err)
			}))
			defer server.Close()

			verification, err := VerifyToken(context.Background(), "token", Config{
				Enabled:         true,
				VerificationURL: server.URL,
				Timeout:         time.Second,
			})

			assert.ErrorIs(t, err, ErrInvalid)
			assert.ErrorContains(t, err, tt.respMsg)
			assert.Nil(t, verification)
		})
	}
}
