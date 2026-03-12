package controllers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"app/helpers"
	"app/models"
)

const (
	RequestTimeout = 10 * time.Second
	DbTimeout      = 10 * time.Second
	LlmTimeout     = 60 * time.Second
	AudioTimeout   = 15 * time.Minute
)

func RenderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	if err := helpers.Render(w, tmpl, data); err != nil {
		http.Error(w, fmt.Sprintf("Erro ao renderizar template (%s): %v", tmpl, err), http.StatusInternalServerError)
	}
}

func RenderWithUser(w http.ResponseWriter, r *http.Request, view string, data map[string]any) {
	ctx, cancel := context.WithTimeout(r.Context(), RequestTimeout)
	defer cancel()

	dbCtx, tx, done, err := models.BeginTransaction(ctx, DbTimeout)
	if err != nil {
		helpers.RenderErrorPage(w, helpers.ErrorPageData{Title: "Erro interno", Brand: "Super Template", Message: err.Error(), Path: r.URL.Path})
		return
	}
	defer done()

	user, err := helpers.GetAuthUser(dbCtx, tx, r)
	if err != nil {
		helpers.RenderUnauthorized(w, r)
		return
	}

	if data == nil {
		data = map[string]any{}
	}

	data["User"] = user

	RenderTemplate(w, view, data)
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	return value
}
