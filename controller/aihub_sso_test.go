package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSetupAIHubSSOSessionCreatesCurrentAuthSession(t *testing.T) {
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousRedis := common.RedisEnabled
	previousSecret := common.SessionSecret
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.UserSession{}, &model.Log{}))
	model.DB, model.LOG_DB = db, db
	common.RedisEnabled = false
	common.SessionSecret = "aihub-sso-session-test-secret"
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.RedisEnabled = previousRedis
		common.SessionSecret = previousSecret
	})

	user := &model.User{
		Username:    "aihub-sso-user",
		Password:    "unused",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		AuthVersion: 1,
	}
	require.NoError(t, db.Create(user).Error)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/user/auth/aihub-sso/entry", nil)
	c.Request.Header.Set("User-Agent", "aihub-sso-test-agent")

	require.NoError(t, setupAIHubSSOSession(c, user))

	var refreshCookie *http.Cookie
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == service.RefreshCookieName {
			refreshCookie = cookie
			break
		}
	}
	require.NotNil(t, refreshCookie)
	assert.Equal(t, "/api/user/auth", refreshCookie.Path)
	assert.True(t, refreshCookie.HttpOnly)

	sid, ok := service.RefreshTokenSID(refreshCookie.Value)
	require.True(t, ok)
	createdSession, err := model.GetUserSessionBySID(sid)
	require.NoError(t, err)
	assert.Equal(t, user.Id, createdSession.UserID)
	assert.Equal(t, aiHubSSOLoginMethod, createdSession.LoginMethod)
	assert.Equal(t, model.UserSessionStatusActive, createdSession.Status)

	secondRecorder := httptest.NewRecorder()
	secondContext, _ := gin.CreateTestContext(secondRecorder)
	secondContext.Request = httptest.NewRequest(http.MethodGet, "/api/user/auth/aihub-sso/entry", nil)
	secondContext.Request.Header.Set("User-Agent", "aihub-sso-test-agent")
	secondContext.Request.AddCookie(refreshCookie)

	require.NoError(t, setupAIHubSSOSession(secondContext, user))
	var refreshedCookie *http.Cookie
	for _, cookie := range secondRecorder.Result().Cookies() {
		if cookie.Name == service.RefreshCookieName {
			refreshedCookie = cookie
			break
		}
	}
	require.NotNil(t, refreshedCookie)
	refreshedSID, ok := service.RefreshTokenSID(refreshedCookie.Value)
	require.True(t, ok)
	assert.Equal(t, sid, refreshedSID, "repeated SSO in the same browser should reuse the current session")

	activeCount, err := model.CountActiveUserSessions(user.Id, time.Now().Unix())
	require.NoError(t, err)
	assert.Equal(t, int64(1), activeCount)
}

func TestAIHubSSOErrorCodeReportsSessionLimits(t *testing.T) {
	assert.Equal(t, "sso-session-limit", aiHubSSOErrorCode(model.ErrUserSessionLimit))
	assert.Equal(t, "sso-session-rate-limit", aiHubSSOErrorCode(model.ErrUserSessionIssuanceLimit))
}
