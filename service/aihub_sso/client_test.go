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
		_, err := w.Write([]byte(`{"code":200,"status":"success","data":{"valid":true,"employNo":"E001"}}`))
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
	assert.Equal(t, "E001", verification.Data.EmployNo)
}
