package aihubsso

import (
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultTimeoutSeconds = 5
	defaultBasePath       = "/"
	defaultMatchField     = "username"
	defaultInitialBalance = 300
)

// Config 只承载环境变量配置，避免为了集团 SSO 定制改动上游 Option 结构和管理后台。
type Config struct {
	Enabled          bool
	VerificationURL  string
	FrontendBasePath string
	AppID            string
	AppSecret        string
	UserMatchField   string
	RequireAppCheck  bool
	InitialBalance   int
	Timeout          time.Duration
}

// LoadConfig 从环境变量读取 AI Hub SSO 配置。
func LoadConfig() Config {
	return Config{
		Enabled:          parseBoolEnv("APP_AUTH_AIHUB_SSO_ENABLED", false),
		VerificationURL:  strings.TrimSpace(os.Getenv("APP_AUTH_AIHUB_SSO_VERIFICATION_URL")),
		FrontendBasePath: normalizeBasePath(os.Getenv("APP_AUTH_AIHUB_SSO_FRONTEND_BASE_PATH")),
		AppID:            strings.TrimSpace(os.Getenv("APP_AUTH_AIHUB_SSO_APP_ID")),
		AppSecret:        strings.TrimSpace(os.Getenv("APP_AUTH_AIHUB_SSO_APP_SECRET")),
		UserMatchField:   normalizeMatchField(os.Getenv("APP_AUTH_AIHUB_SSO_USER_MATCH_FIELD")),
		RequireAppCheck:  parseBoolEnv("APP_AUTH_AIHUB_SSO_REQUIRE_APP_CHECK", true),
		InitialBalance:   parseIntEnv("APP_AUTH_AIHUB_SSO_INITIAL_BALANCE", defaultInitialBalance),
		Timeout:          time.Duration(parseIntEnv("APP_AUTH_AIHUB_SSO_TIMEOUT_SECONDS", defaultTimeoutSeconds)) * time.Second,
	}
}

func parseBoolEnv(name string, defaultValue bool) bool {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func parseIntEnv(name string, defaultValue int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return defaultValue
	}
	return parsed
}

func normalizeMatchField(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "username":
		return defaultMatchField
	case "oidc_id":
		return "oidc_id"
	default:
		return defaultMatchField
	}
}

func normalizeBasePath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultBasePath
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	for len(value) > 1 && strings.HasSuffix(value, "/") {
		value = strings.TrimSuffix(value, "/")
	}
	return value
}
