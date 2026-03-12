package profile

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"app/helpers"
	"app/models"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
)

const (
	requestTimeout = 8 * time.Second
	dbTimeout      = 8 * time.Second
)

type Handler struct {
	service *Service
	logger  *slog.Logger
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service, logger: slog.Default()}
}

func (h *Handler) RegisterRoutes(router chi.Router) {
	router.Route("/app/profile", func(r chi.Router) {
		r.Use(httprate.LimitByIP(60, time.Minute))

		r.Get("/", helpers.AuthDecorator(h.renderProfile))
		r.Post("/name", helpers.AuthDecorator(h.updateName))
		r.With(httprate.LimitByIP(5, time.Minute)).Post("/delete", helpers.AuthDecorator(h.deleteAccount))
	})
}

func (h *Handler) renderProfile(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	dbCtx, tx, done, err := models.BeginTransaction(ctx, dbTimeout)
	if err != nil {
		h.handlePageError(w, r, "Erro interno", newInternalError(err))
		return
	}
	defer done()

	user, err := helpers.GetAuthUser(dbCtx, tx, r)
	if err != nil {
		helpers.RenderUnauthorized(w, r)
		return
	}

	profile, appErr := h.service.GetProfile(dbCtx, tx, user.ID)
	if appErr != nil {
		h.handlePageError(w, r, "Erro ao carregar perfil", appErr)
		return
	}

	status := strings.TrimSpace(r.URL.Query().Get("status"))
	var alert map[string]any
	if status == "updated" {
		alert = map[string]any{"Type": "success", "Message": "Nome atualizado com sucesso."}
	}
	if status == "deleted" {
		alert = map[string]any{"Type": "success", "Message": "Conta excluída com sucesso."}
	}

	err = helpers.Render(w, filepath.Join("app", "profile.ejs"), map[string]any{
		"User":        user,
		"CurrentPage": "profile",
		"Profile":     profile,
		"Alert":       alert,
		"CSRFToken":   helpers.EnsureCSRFToken(w, r),
	})
	if err != nil {
		h.handlePageError(w, r, "Erro ao renderizar perfil", newInternalError(err))
	}
}

func (h *Handler) updateName(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	dbCtx, tx, done, err := models.BeginTransaction(ctx, dbTimeout)
	if err != nil {
		h.handlePageError(w, r, "Erro interno", newInternalError(err))
		return
	}
	defer done()

	user, err := helpers.GetAuthUser(dbCtx, tx, r)
	if err != nil {
		helpers.RenderUnauthorized(w, r)
		return
	}

	if !helpers.ValidateCSRFToken(r) {
		h.renderProfileWithAlert(w, r, tx, user.ID, map[string]any{"Type": "danger", "Message": "Token CSRF inválido. Atualize a página e tente novamente."})
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	_, appErr := h.service.UpdateName(dbCtx, tx, user.ID, name)
	if appErr != nil {
		if appErr.Status == http.StatusBadRequest {
			h.renderProfileWithAlert(w, r, tx, user.ID, map[string]any{"Type": "danger", "Message": appErr.Message})
			return
		}
		h.handlePageError(w, r, "Erro ao atualizar perfil", appErr)
		return
	}

	http.Redirect(w, r, "/app/profile?status=updated", http.StatusSeeOther)
}

func (h *Handler) deleteAccount(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	dbCtx, tx, done, err := models.BeginTransaction(ctx, dbTimeout)
	if err != nil {
		h.handlePageError(w, r, "Erro interno", newInternalError(err))
		return
	}
	defer done()

	user, err := helpers.GetAuthUser(dbCtx, tx, r)
	if err != nil {
		helpers.RenderUnauthorized(w, r)
		return
	}

	if !helpers.ValidateCSRFToken(r) {
		h.renderProfileWithAlert(w, r, tx, user.ID, map[string]any{"Type": "danger", "Message": "Token CSRF inválido. Atualize a página e tente novamente."})
		return
	}

	confirm := strings.TrimSpace(r.FormValue("confirm"))
	if confirm != "EXCLUIR" {
		h.renderProfileWithAlert(w, r, tx, user.ID, map[string]any{"Type": "danger", "Message": "Confirmação inválida. Digite EXCLUIR para confirmar."})
		return
	}

	if appErr := h.service.DeleteAccount(dbCtx, tx, user.ID); appErr != nil {
		h.handlePageError(w, r, "Erro ao excluir conta", appErr)
		return
	}

	helpers.ClearAuthCookie(w)
	http.Redirect(w, r, "/login?status=account_deleted", http.StatusSeeOther)
}

func (h *Handler) renderProfileWithAlert(w http.ResponseWriter, r *http.Request, tx *sql.Tx, userID string, alert map[string]any) {
	profile, appErr := h.service.GetProfile(r.Context(), tx, userID)
	if appErr != nil {
		h.handlePageError(w, r, "Erro ao carregar perfil", appErr)
		return
	}

	user := &models.User{ID: profile.ID, Name: profile.Name, Email: profile.Email}
	err := helpers.Render(w, filepath.Join("app", "profile.ejs"), map[string]any{
		"User":        user,
		"CurrentPage": "profile",
		"Profile":     profile,
		"Alert":       alert,
		"CSRFToken":   helpers.EnsureCSRFToken(w, r),
	})
	if err != nil {
		h.handlePageError(w, r, "Erro ao renderizar perfil", newInternalError(err))
	}
}

func (h *Handler) handlePageError(w http.ResponseWriter, r *http.Request, title string, appErr *AppError) {
	if appErr == nil {
		appErr = newInternalError(nil)
	}

	h.logger.Error("profile module error",
		"path", r.URL.Path,
		"method", r.Method,
		"code", appErr.Code,
		"status", appErr.Status,
		"error", appErr.Error(),
	)

	helpers.RenderErrorPage(w, helpers.ErrorPageData{
		Title:   title,
		Brand:   "Super Template",
		Message: appErr.Message,
		Path:    r.URL.Path,
	})
}
