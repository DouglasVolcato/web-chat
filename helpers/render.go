package helpers

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var viewsBaseDir = resolveTemplatesBaseDir()
var assetsVersion = resolveAssetsVersion()

func resolveTemplatesBaseDir() string {
	candidates := []string{
		filepath.Join("presentation", "views"),
		filepath.Join("app", "presentation", "views"),
	}

	for _, dir := range candidates {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}

	return candidates[0]
}

func resolveAssetsVersion() string {
	if version := strings.TrimSpace(os.Getenv("APP_ASSET_VERSION")); version != "" {
		return version
	}

	publicDir := filepath.Join("presentation", "public")
	info, err := os.Stat(publicDir)
	if err != nil {
		return fmt.Sprint(time.Now().UTC().Unix())
	}

	return fmt.Sprint(info.ModTime().UTC().Unix())
}

func Render(w http.ResponseWriter, view string, data any) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if data == nil {
		data = map[string]any{}
	}

	funcMap := template.FuncMap{
		"url": func(path string) string {
			return PathURL(path)
		},
		"assetVersion": func() string {
			return assetsVersion
		},
		"default": func(value any, defaultValue string) string {
			switch v := value.(type) {
			case string:
				if strings.TrimSpace(v) == "" {
					return defaultValue
				}
				return v
			case fmt.Stringer:
				str := v.String()
				if strings.TrimSpace(str) == "" {
					return defaultValue
				}
				return str
			case nil:
				return defaultValue
			default:
				str := fmt.Sprint(v)
				if strings.TrimSpace(str) == "" {
					return defaultValue
				}
				return str
			}
		},
		"dict": func(values ...any) map[string]any {
			result := map[string]any{}
			for i := 0; i+1 < len(values); i += 2 {
				key := fmt.Sprint(values[i])
				result[key] = values[i+1]
			}
			return result
		},
		"eq": func(a, b any) bool {
			return fmt.Sprint(a) == fmt.Sprint(b)
		},
		"list": func(values ...any) []any {
			return values
		},
		"toLower": strings.ToLower,
	}

	viewPath := filepath.Join(viewsBaseDir, view)
	partials, err := filepath.Glob(filepath.Join(viewsBaseDir, "partials", "*.ejs"))
	if err != nil {
		return err
	}

	files := append([]string{viewPath}, partials...)

	tmpl, err := template.New(filepath.Base(viewPath)).Funcs(funcMap).ParseFiles(files...)
	if err != nil {
		return err
	}

	base := filepath.Base(viewPath)
	defined := strings.TrimSuffix(base, filepath.Ext(base))
	if tmpl.Lookup(defined) != nil {
		return tmpl.ExecuteTemplate(w, defined, data)
	}

	return tmpl.ExecuteTemplate(w, base, data)
}
