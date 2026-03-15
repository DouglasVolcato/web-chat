package controllers

import (
	"context"
	"net/http"
	"path/filepath"
	"time"

	"app/helpers"
	"app/models"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
)

type AppController struct{}

func NewAppController() *AppController {
	return &AppController{}
}

func (c *AppController) RegisterRoutes(router chi.Router) {
	router.Group(func(r chi.Router) {
		r.Use(httprate.LimitByIP(30, time.Minute))
		r.Use(func(next http.Handler) http.Handler {
			return helpers.AuthDecorator(next.ServeHTTP)
		})

		r.Get("/app", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, helpers.PathURL("/app/dashboard"), http.StatusSeeOther)
		})

		r.Get("/app/dashboard", func(w http.ResponseWriter, r *http.Request) {
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

			RenderTemplate(w, filepath.Join("app", "dashboard.ejs"), map[string]any{
				"User": user,
			})
		})
	})
}
