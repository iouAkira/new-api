package aihubsso

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfigReadsUserInfoEndpointURLs(t *testing.T) {
	t.Setenv("APP_AUTH_AIHUB_SSO_ACCESS_TOKEN_URL", " https://aihub.example.com/access-token ")
	t.Setenv("APP_AUTH_AIHUB_SSO_USER_INFO_URL", " https://aihub.example.com/user-info ")

	cfg := LoadConfig()

	assert.Equal(t, "https://aihub.example.com/access-token", cfg.AccessTokenURL)
	assert.Equal(t, "https://aihub.example.com/user-info", cfg.UserInfoURL)
}
