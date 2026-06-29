package middleware

import (
	"net/http"
	"net/url"

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

		entry := url.URL{Path: "/api/auth/aihub-sso/entry"}
		entryQuery := entry.Query()
		entryQuery.Set("ai-hub-token", token)
		entryQuery.Set("redirect", cleanRedirect)
		entryQuery.Set("basePath", cfg.FrontendBasePath)
		entry.RawQuery = entryQuery.Encode()

		c.Redirect(http.StatusFound, entry.String())
		c.Abort()
	}
}
