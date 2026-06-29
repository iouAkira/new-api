package aihub_sso

import (
	"net/url"
	"path"
	"strings"
)

func CleanRedirect(rawRedirect string, basePath string) string {
	basePath = normalizeBasePath(basePath)
	rawRedirect = strings.TrimSpace(rawRedirect)
	if rawRedirect == "" {
		rawRedirect = "/"
	}

	parsed, err := url.Parse(rawRedirect)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || strings.HasPrefix(rawRedirect, "//") {
		rawRedirect = "/"
		parsed, _ = url.Parse(rawRedirect)
	}

	cleanPath := parsed.Path
	if cleanPath == "" {
		cleanPath = "/"
	}
	if !strings.HasPrefix(cleanPath, "/") {
		cleanPath = "/" + cleanPath
	}

	query := parsed.Query()
	query.Del("ai-hub-token")
	parsed.RawQuery = query.Encode()
	parsed.Fragment = ""

	if basePath == "/" {
		parsed.Path = cleanPath
	} else {
		trimmedBase := strings.TrimSuffix(basePath, "/")
		if cleanPath == trimmedBase || strings.HasPrefix(cleanPath, trimmedBase+"/") {
			parsed.Path = cleanPath
		} else {
			parsed.Path = path.Join(trimmedBase, cleanPath)
			if strings.HasSuffix(cleanPath, "/") && !strings.HasSuffix(parsed.Path, "/") {
				parsed.Path += "/"
			}
		}
	}

	result := parsed.String()
	if result == "" {
		return basePath + "/"
	}
	return result
}
