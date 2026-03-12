package helpers

import (
	"os"
	"testing"
)

func TestPathURL(t *testing.T) {
	t.Setenv("URL_PATH", "/chat")

	tests := map[string]string{
		"":                    "/chat",
		"/":                   "/chat/",
		"/app/dashboard":      "/chat/app/dashboard",
		"app/dashboard":       "/chat/app/dashboard",
		"/chat":               "/chat",
		"/chat/app/dashboard": "/chat/app/dashboard",
		"https://example.com": "https://example.com",
	}

	for input, expected := range tests {
		if got := PathURL(input); got != expected {
			t.Fatalf("PathURL(%q) = %q; want %q", input, got, expected)
		}
	}
}

func TestURLPathRootOrEmpty(t *testing.T) {
	for _, value := range []string{"", "/", "  /  "} {
		t.Setenv("URL_PATH", value)
		if got := URLPath(); got != "" {
			t.Fatalf("URLPath() with %q = %q; want empty", value, got)
		}
	}

	os.Unsetenv("URL_PATH")
}
