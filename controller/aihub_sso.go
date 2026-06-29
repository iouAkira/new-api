package controller

import (
	"errors"
	"fmt"
	"html"
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
		renderAIHubSSOErrorPage(c, basePath, "sso-disabled")
		return
	}

	token := c.Query("ai-hub-token")
	common.SysLog("AI Hub SSO entry received token=" + aihubsso.MaskToken(token))
	verification, err := aihubsso.VerifyToken(c.Request.Context(), token, cfg)
	if err != nil {
		common.SysLog("AI Hub SSO verification failed: " + err.Error())
		renderAIHubSSOErrorPage(c, basePath, aiHubSSOErrorCode(err))
		return
	}

	userCreated := false
	user, err := model.GetUserByAIHubEmployNo(verification.Data.EmployNo, cfg.UserMatchField)
	if err != nil {
		if model.IsAIHubUserNotFound(err) {
			if cfg.UserMatchField != "username" || !validAIHubAutoCreateUsername(verification.Data.EmployNo) {
				renderAIHubSSOErrorPage(c, basePath, "no-permission")
				return
			}
			user, err = model.CreateAIHubSSOUser(verification.Data.EmployNo)
			if err != nil {
				common.SysLog("AI Hub SSO auto create user failed: " + err.Error())
				renderAIHubSSOErrorPage(c, basePath, "no-permission")
				return
			}
			userCreated = true
		} else {
			common.SysLog("AI Hub SSO user lookup failed: " + err.Error())
			renderAIHubSSOErrorPage(c, basePath, "sso-invalid")
			return
		}
	}
	if user.Status != common.UserStatusEnabled {
		renderAIHubSSOErrorPage(c, basePath, "user-disabled")
		return
	}

	if err := setupAIHubSSOSession(c, user); err != nil {
		common.SysLog("AI Hub SSO session save failed: " + err.Error())
		renderAIHubSSOErrorPage(c, basePath, "sso-invalid")
		return
	}

	redirect := aihubsso.CleanRedirect(c.Query("redirect"), basePath)
	renderAIHubSSOBootstrap(c, user, redirect, userCreated)
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

func validAIHubAutoCreateUsername(username string) bool {
	username = strings.TrimSpace(username)
	if len(username) != 10 {
		return false
	}
	digits := 0
	for _, r := range username {
		if r >= '0' && r <= '9' {
			digits++
		}
	}
	return digits >= 8
}

func renderAIHubSSOErrorPage(c *gin.Context, basePath string, errorCode string) {
	title, message := aiHubSSOErrorText(errorCode)
	htmlBody := fmt.Sprintf(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>%s</title>
  <style>
    body{margin:0;background:#f6f7f9;color:#1f2937;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif}
    main{min-height:100vh;display:flex;align-items:center;justify-content:center;padding:24px}
    section{width:min(520px,100%%);background:#fff;border:1px solid #e5e7eb;border-radius:8px;padding:28px;box-shadow:0 12px 30px rgba(15,23,42,.08)}
    h1{margin:0 0 12px;font-size:22px}
    p{margin:0 0 20px;color:#64748b;line-height:1.7}
    a{display:inline-block;color:#0f766e;text-decoration:none;font-weight:600}
    code{background:#eef2f7;border-radius:4px;padding:2px 5px;color:#334155}
  </style>
</head>
<body>
  <main>
    <section>
      <h1>%s</h1>
      <p>%s</p>
      <p><code>%s</code></p>
    </section>
  </main>
</body>
</html>`, html.EscapeString(title), html.EscapeString(title), html.EscapeString(message), html.EscapeString(errorCode))

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(htmlBody))
	c.Abort()
}

func aiHubSSOErrorText(errorCode string) (string, string) {
	switch errorCode {
	case "no-permission":
		return "无法创建或匹配用户", "AI Hub 身份校验已通过，但本地不存在对应用户，且工号不满足自动创建规则。请联系管理员处理。"
	case "user-disabled":
		return "用户已被禁用", "AI Hub 身份校验已通过，但对应的本地用户已被禁用，请联系管理员处理。"
	case "sso-config-error":
		return "SSO 配置错误", "当前系统的 AI Hub SSO 配置不完整或不可用，请检查 tokenVerification 地址和相关环境变量。"
	case "sso-timeout":
		return "AI Hub 校验超时", "系统暂时无法连接 AI Hub tokenVerification 服务，请稍后重试或联系管理员。"
	case "sso-disabled":
		return "SSO 未启用", "当前系统未启用 AI Hub SSO，请检查 APP_AUTH_AIHUB_SSO_ENABLED 配置。"
	default:
		return "SSO 登录失败", "AI Hub token 无效、已过期，或返回内容未通过系统校验，请重新从 AI Hub 发起登录。"
	}
}

func renderAIHubSSOBootstrap(c *gin.Context, user *model.User, redirect string, userCreated bool) {
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
		renderAIHubSSOErrorPage(c, "/", "sso-invalid")
		return
	}

	message := "正在登录..."
	delay := 100
	if userCreated {
		message = "用户不存在，正在创建用户..."
		delay = 900
	}

	htmlBody := fmt.Sprintf(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="referrer" content="no-referrer">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>AI Hub SSO 登录中</title>
  <style>
    body{margin:0;background:#f6f7f9;color:#1f2937;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif}
    main{min-height:100vh;display:flex;align-items:center;justify-content:center;padding:24px}
    section{width:min(520px,100%%);background:#fff;border:1px solid #e5e7eb;border-radius:8px;padding:28px;box-shadow:0 12px 30px rgba(15,23,42,.08)}
    h1{margin:0 0 12px;font-size:22px}
    p{margin:0;color:#64748b;line-height:1.7}
  </style>
</head>
<body>
<main><section><h1>%s</h1><p>完成后将自动进入系统。</p></section></main>
<script>
(function () {
  try {
    localStorage.setItem('uid', %q);
    localStorage.setItem('user', %q);
  } catch (e) {}
  setTimeout(function () {
    location.replace(%q);
  }, %d);
})();
</script>
</body>
</html>`, html.EscapeString(message), fmt.Sprintf("%d", user.Id), string(userJSON), redirect, delay)

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(htmlBody))
	c.Abort()
}
