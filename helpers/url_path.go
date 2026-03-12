package helpers

import (
	"os"
	"strings"
)

func URLPath() string {
	raw := strings.TrimSpace(os.Getenv("URL_PATH"))
	if raw == "" || raw == "/" {
		return ""
	}

	raw = "/" + strings.Trim(raw, "/")
	return strings.TrimRight(raw, "/")
}

func PathURL(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		base := URLPath()
		if base == "" {
			return "/"
		}
		return base
	}

	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") || strings.HasPrefix(path, "//") {
		return path
	}

	base := URLPath()
	if path == "/" {
		if base == "" {
			return "/"
		}
		return base + "/"
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	if base == "" {
		return path
	}

	if path == base || strings.HasPrefix(path, base+"/") {
		return path
	}

	return base + path
}
