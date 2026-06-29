package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	aihubsso "github.com/QuantumNous/new-api/service/aihub_sso"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

const aiHubSSOLoginMethod = "AI_HUB_SSO"

// AIHubSSOEntry 校验 AI Hub token，恢复 new-api 既有 session 结构，
// 并返回极薄的 bootstrap 页面以同步 SPA 需要的本地登录态。
func AIHubSSOEntry(c *gin.Context) {
	cfg := aihubsso.LoadConfig()
	basePath := cfg.FrontendBasePath
	if queryBasePath := strings.TrimSpace(c.Query("basePath")); queryBasePath != "" {
		basePath = queryBasePath
	}

	if !cfg.Enabled {
		redirectAIHubSSOError(c, basePath, "sso-disabled")
		return
	}

	token := c.Query("ai-hub-token")
	common.SysLog("AI Hub SSO entry received token=" + aihubsso.MaskToken(token))
	verification, err := aihubsso.VerifyToken(c.Request.Context(), token, cfg)
	if err != nil {
		common.SysLog("AI Hub SSO verification failed: " + err.Error())
		redirectAIHubSSOError(c, basePath, aiHubSSOErrorCode(err))
		return
	}

	user, err := model.GetUserByAIHubEmployNo(verification.Data.EmployNo, cfg.UserMatchField)
	if err != nil {
		if model.IsAIHubUserNotFound(err) {
			redirectAIHubSSOError(c, basePath, "no-permission")
			return
		}
		common.SysLog("AI Hub SSO user lookup failed: " + err.Error())
		redirectAIHubSSOError(c, basePath, "sso-invalid")
		return
	}
	if user.Status != common.UserStatusEnabled {
		redirectAIHubSSOError(c, basePath, "user-disabled")
		return
	}

	if err := setupAIHubSSOSession(c, user); err != nil {
		common.SysLog("AI Hub SSO session save failed: " + err.Error())
		redirectAIHubSSOError(c, basePath, "sso-invalid")
		return
	}

	redirect := aihubsso.CleanRedirect(c.Query("redirect"), basePath)
	renderAIHubSSOBootstrap(c, user, redirect)
}

func aiHubSSOErrorCode(err error) string {
	switch {
	case errors.Is(err, aihubsso.ErrAppMismatch):
		return "sso-invalid"
	case errors.Is(err, aihubsso.ErrConfig):
		return "sso-config-error"
	case errors.Is(err, aihubsso.ErrRequestFailed):
		return "sso-timeout"
	default:
		return "sso-invalid"
	}
}

func setupAIHubSSOSession(c *gin.Context, user *model.User) error {
	model.UpdateUserLastLoginAt(user.Id)
	session := sessions.Default(c)
	session.Set("id", user.Id)
	session.Set("username", user.Username)
	session.Set("role", user.Role)
	session.Set("status", user.Status)
	session.Set("group", user.Group)
	session.Set("login_type", aiHubSSOLoginMethod)
	if err := session.Save(); err != nil {
		return err
	}

	model.RecordLoginLog(user.Id, user.Username, "Logged in successfully via AI Hub SSO", c.ClientIP(), "login", map[string]interface{}{
		"method": aiHubSSOLoginMethod,
	}, map[string]interface{}{
		"login_method": aiHubSSOLoginMethod,
		"user_agent":   c.Request.UserAgent(),
	})
	return nil
}

func redirectAIHubSSOError(c *gin.Context, basePath string, errorCode string) {
	redirect := aihubsso.CleanRedirect("/sign-in?ssoError="+errorCode, basePath)
	c.Redirect(http.StatusFound, redirect)
	c.Abort()
}

func renderAIHubSSOBootstrap(c *gin.Context, user *model.User, redirect string) {
	userData := map[string]interface{}{
		"id":           user.Id,
		"username":     user.Username,
		"display_name": user.DisplayName,
		"role":         user.Role,
		"status":       user.Status,
		"group":        user.Group,
	}
	userJSON, err := common.Marshal(userData)
	if err != nil {
		common.SysLog("AI Hub SSO bootstrap marshal failed: " + err.Error())
		redirectAIHubSSOError(c, "/", "sso-invalid")
		return
	}

	htmlBody := fmt.Sprintf(`<!doctype html>
<html>
<head><meta charset="utf-8"><meta name="referrer" content="no-referrer"><title>Signing in</title></head>
<body>
<script>
(function () {
  try {
    localStorage.setItem('uid', %q);
    localStorage.setItem('user', %q);
  } catch (e) {}
  location.replace(%q);
})();
</script>
</body>
</html>`, fmt.Sprintf("%d", user.Id), string(userJSON), redirect)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(htmlBody))
	c.Abort()
}
