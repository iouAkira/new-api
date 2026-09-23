package middleware

import (
	"net/http"
	"net/url"

	"github.com/QuantumNous/new-api/common"
	aihubsso "github.com/QuantumNous/new-api/service/aihub_sso"
	"github.com/gin-gonic/gin"
)

// AIHubSSOWebEntry 将页面 URL 上的 ai-hub-token 转交给后端 SSO 入口，
// 从而避免改动 default/classic 两套前端。
func AIHubSSOWebEntry() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet {
			c.Next()
			return
		}

		token := c.Query("ai-hub-token")
		if token == "" {
			c.Next()
			return
		}

		cfg := aihubsso.LoadConfig()
		if !cfg.Enabled {
			c.Next()
			return
		}

		cleanURL := *c.Request.URL
		query := cleanURL.Query()
		query.Del("ai-hub-token")
		cleanURL.RawQuery = query.Encode()
		cleanRedirect := cleanURL.RequestURI()

		// 放在 refresh Cookie 的 Path 下，使同一浏览器重复 SSO 时可以复用当前会话。
		entry := url.URL{Path: "/api/user/auth/aihub-sso/entry"}
		entryQuery := entry.Query()
		entryQuery.Set("ai-hub-token", token)
		entryQuery.Set("redirect", cleanRedirect)
		entryQuery.Set("basePath", cfg.FrontendBasePath)
		entry.RawQuery = entryQuery.Encode()

		// 先落地为同源页面，再由浏览器发起导航。这样从 AI Hub 跨站进入时，
		// SameSite=Strict 的 refresh Cookie 也会在下一跳带到 /api/user/auth 路径。
		entryJSON, err := common.Marshal(entry.String())
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		body := []byte(`<!doctype html><html><head><meta charset="utf-8"><meta name="referrer" content="no-referrer"><meta name="robots" content="noindex"><title>AI Hub SSO</title></head><body><script>location.replace(` + string(entryJSON) + `);</script></body></html>`)
		c.Header("Cache-Control", "no-store")
		c.Data(http.StatusOK, "text/html; charset=utf-8", body)
		c.Abort()
	}
}
