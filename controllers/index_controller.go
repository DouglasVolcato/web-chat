package controllers

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"app/helpers"
	"app/models"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
)

type IndexController struct{}

func (c *IndexController) RegisterRoutes(router chi.Router) {
	router.Group(func(r chi.Router) {
		r.Use(httprate.LimitByIP(30, time.Minute))

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			helpers.Redirect(w, r, "/login")
		})

		r.Get("/termos", func(w http.ResponseWriter, r *http.Request) {
			RenderTemplate(w, filepath.Join("landing", "terms.ejs"), nil)
		})

		r.Get("/privacidade", func(w http.ResponseWriter, r *http.Request) {
			RenderTemplate(w, filepath.Join("landing", "privacy.ejs"), nil)
		})

		r.Get("/login", func(w http.ResponseWriter, r *http.Request) {
			if userID, err := helpers.ResolveUserIDFromRequest(r); err == nil && userID != "" {
				helpers.Redirect(w, r, "/app/messages")
				return
			}

			RenderTemplate(w, filepath.Join("landing", "login.ejs"), map[string]any{
				"GoogleClientID": strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")),
			})
		})

		r.Get("/register", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, helpers.PathURL("/login"), http.StatusTemporaryRedirect)
		})

		r.Post("/logout", func(w http.ResponseWriter, r *http.Request) {
			helpers.ClearAuthCookie(w)
			helpers.Redirect(w, r, "/login")
		})
	})

	router.Group(func(r chi.Router) {
		r.Use(httprate.LimitByIP(7, time.Minute))

		r.Post("/auth/google", func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), RequestTimeout)
			defer cancel()

			credential := strings.TrimSpace(r.FormValue("credential"))
			if credential == "" {
				RenderTemplate(w, filepath.Join("partials", "alert.ejs"), map[string]any{
					"Type":    "warning",
					"Message": "Não recebemos o token do Google. Tente novamente.",
				})
				return
			}

			profile, err := helpers.VerifyGoogleIDToken(ctx, credential)
			if err != nil {
				RenderTemplate(w, filepath.Join("partials", "alert.ejs"), map[string]any{
					"Type":    "danger",
					"Message": "Não foi possível validar seu login com o Google. Tente novamente.",
				})
				return
			}

			dbCtx, tx, done, err := models.BeginTransaction(ctx, DbTimeout)
			if err != nil {
				RenderTemplate(w, filepath.Join("partials", "alert.ejs"), map[string]any{
					"Type":    "danger",
					"Message": "Erro ao iniciar a autenticação. Tente novamente.",
				})
				return
			}
			defer done()

			user, err := models.GetUserByEmail(dbCtx, tx, profile.Email)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					displayName := profile.Name
					if displayName == "" {
						parts := strings.Split(profile.Email, "@")
						displayName = parts[0]
					}

					user = &models.User{
						Name:     displayName,
						Email:    profile.Email,
						Password: "",
					}

					if err := user.Create(dbCtx, tx); err != nil {
						RenderTemplate(w, filepath.Join("partials", "alert.ejs"), map[string]any{
							"Type":    "danger",
							"Message": "Erro ao salvar usuário. Atualize e tente novamente.",
						})
						return
					}
				} else {
					RenderTemplate(w, filepath.Join("partials", "alert.ejs"), map[string]any{
						"Type":    "danger",
						"Message": "Erro ao validar sua conta. Tente novamente.",
					})
					return
				}
			}

			if err := helpers.SetAuthCookie(w, user.ID, 24*time.Hour); err != nil {
				RenderTemplate(w, filepath.Join("partials", "alert.ejs"), map[string]any{
					"Type":    "danger",
					"Message": "Erro ao salvar sua sessão. Atualize a página e tente novamente.",
				})
				return
			}

			helpers.Redirect(w, r, "/app/messages")

			RenderTemplate(w, filepath.Join("partials", "alert.ejs"), map[string]any{
				"Type":    "success",
				"Message": "Login com Google realizado! Redirecionando...",
			})
		})
	})
}
