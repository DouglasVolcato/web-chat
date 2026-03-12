package helpers

import (
	"html/template"
	"net/http"
)

type ErrorPageData struct {
	Title   string
	Brand   string
	Message string
	Path    string
	DashboardPath string
}

func RenderErrorPage(w http.ResponseWriter, data ErrorPageData) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)

	if data.DashboardPath == "" {
		data.DashboardPath = PathURL("/app/dashboard")
	}

	tmpl, err := template.New("error-page").Parse(errorPageTemplate)
	if err != nil {
		return err
	}

	return tmpl.Execute(w, data)
}

const errorPageTemplate = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{ .Title }}</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css">
</head>
<body>
    <div class="container py-5">
        <div class="row justify-content-center">
            <div class="col-12 col-lg-8">
                <div class="card shadow-sm border-0">
                    <div class="card-body p-4">
                        <div class="d-flex align-items-center mb-3">
                            <span class="badge rounded-pill bg-danger-subtle text-danger me-2">500</span>
                            <span class="fw-semibold text-uppercase text-muted small">{{ .Brand }}</span>
                        </div>
                        <h1 class="h4 mb-3">{{ .Title }}</h1>
                        <p class="text-secondary mb-4">{{ .Message }}</p>
                        <div class="rounded-3 p-3 border d-flex align-items-center">
                            <span class="text-muted small">Caminho:</span>
                            <span class="ms-2 fw-semibold text-body">{{ .Path }}</span>
                        </div>
                        <a href="{{ .DashboardPath }}" class="btn btn-primary mt-4">Voltar para o painel</a>
                    </div>
                </div>
            </div>
        </div>
    </div>
</body>
</html>`
