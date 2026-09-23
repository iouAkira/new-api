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

func TestFetchUserInfoGetsAccessTokenThenUserProfile(t *testing.T) {
	t.Parallel()

	type accessTokenRequest struct {
		EmployNo  string `json:"employNo"`
		AppID     string `json:"appId"`
		AppSecret string `json:"appSecret"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/access-token":
			assert.Equal(t, http.MethodPost, r.Method)
			var body accessTokenRequest
			require.NoError(t, common.DecodeJson(r.Body, &body))
			assert.Equal(t, accessTokenRequest{
				EmployNo:  "2205090261",
				AppID:     "app-id",
				AppSecret: "app-secret",
			}, body)
			_, err := w.Write([]byte(`{"code":200,"status":"success","message":"操作成功","data":{"expiresIn":"3600","access_token":"user-access-token"}}`))
			require.NoError(t, err)
		case "/user-info":
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "2205090261", r.URL.Query().Get("userNo"))
			assert.Equal(t, "user-access-token", r.Header.Get("token"))
			_, err := w.Write([]byte(`{"code":200,"status":"success","message":"操作成功","data":{"userNo":"2205090261","userName":"范艳梅","orgBizId":"10370002","orgName":"体系运维组","position":"经理","status":"A","email":"GPCM@SUNWODA.COM","sex":"1","orgBizChain":["10000000","10370002"]}}`))
			require.NoError(t, err)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	userInfo, err := FetchUserInfo(context.Background(), "2205090261", Config{
		AccessTokenURL: server.URL + "/access-token",
		UserInfoURL:    server.URL + "/user-info",
		AppID:          "app-id",
		AppSecret:      "app-secret",
		Timeout:        time.Second,
	})

	require.NoError(t, err)
	require.NotNil(t, userInfo)
	assert.Equal(t, "2205090261", userInfo.UserNo)
	assert.Equal(t, "范艳梅", userInfo.UserName)
	assert.Equal(t, "体系运维组", userInfo.OrgName)
}

func TestFetchUserInfoReturnsUpstreamFailureMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		accessTokenResponse string
		userInfoResponse    string
		wantMessage         string
	}{
		{
			name:                "employee does not exist",
			accessTokenResponse: `{"datas":null,"resp_code":0,"resp_msg":"当前用户不存在！"}`,
			wantMessage:         "当前用户不存在！",
		},
		{
			name:                "user info not found",
			accessTokenResponse: `{"code":200,"status":"success","data":{"access_token":"user-access-token"}}`,
			userInfoResponse:    `{"code":200,"status":"success","message":"未查到对应数据","data":null}`,
			wantMessage:         "未查到对应数据",
		},
		{
			name:                "invalid user info token",
			accessTokenResponse: `{"code":200,"status":"success","data":{"access_token":"user-access-token"}}`,
			userInfoResponse:    `{"datas":null,"resp_code":0,"resp_msg":"无效token"}`,
			wantMessage:         "无效token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/access-token" {
					_, err := w.Write([]byte(tt.accessTokenResponse))
					require.NoError(t, err)
					return
				}
				_, err := w.Write([]byte(tt.userInfoResponse))
				require.NoError(t, err)
			}))
			defer server.Close()

			userInfo, err := FetchUserInfo(context.Background(), "2205090261", Config{
				AccessTokenURL: server.URL + "/access-token",
				UserInfoURL:    server.URL + "/user-info",
				Timeout:        time.Second,
			})

			assert.ErrorIs(t, err, ErrInvalid)
			assert.ErrorContains(t, err, tt.wantMessage)
			assert.Nil(t, userInfo)
		})
	}
}
