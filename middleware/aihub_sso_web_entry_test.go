package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAIHubSSOWebEntryUsesSameOriginNavigationToCookieScopedEndpoint(t *testing.T) {
	t.Setenv("APP_AUTH_AIHUB_SSO_ENABLED", "true")
	t.Setenv("APP_AUTH_AIHUB_SSO_FRONTEND_BASE_PATH", "/console")

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/console?ai-hub-token=token-value&tab=home", nil)

	AIHubSSOWebEntry()(c)

	result := recorder.Result()
	require.Equal(t, http.StatusOK, result.StatusCode)
	assert.Equal(t, "no-store", result.Header.Get("Cache-Control"))
	assert.Empty(t, result.Header.Get("Location"), "a server redirect can suppress SameSite=Strict cookies across the redirect chain")
	body := recorder.Body.String()
	assert.True(t, strings.Contains(body, `location.replace("/api/user/auth/aihub-sso/entry?`))
	assert.Contains(t, body, "ai-hub-token=token-value")
	assert.Contains(t, body, "redirect=%2Fconsole%3Ftab%3Dhome")
}
