package mvpchat

import (
	"context"
	"database/sql"
	"encoding/json"
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

const (
	requestTimeout = 10 * time.Second
	dbTimeout      = 10 * time.Second
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

type jsonMessageResponse struct {
	Message     string `json:"message,omitempty"`
	RedirectURL string `json:"redirect_url,omitempty"`
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/app/messages", func(router chi.Router) {
		router.Use(httprate.LimitByIP(60, time.Minute))
		router.Use(helpers.UserRateLimit(120, time.Minute))
		router.Get("/", helpers.AuthDecorator(h.listChatsPage))
		router.Get("/{chatID}", helpers.AuthDecorator(h.chatPage))
		router.Post("/send", helpers.AuthDecorator(h.sendMessage))
		router.Post("/contacts/qr", helpers.AuthDecorator(h.generateQR))
		router.Post("/contacts/qr/redeem", helpers.AuthDecorator(h.redeemQR))
		router.Post("/account/delete", helpers.AuthDecorator(h.deleteAccount))
	})

	r.Route("/app/push", func(router chi.Router) {
		router.Use(httprate.LimitByIP(40, time.Minute))
		router.Use(helpers.UserRateLimit(80, time.Minute))
		router.Post("/subscriptions", helpers.AuthDecorator(h.registerPushSubscription))
		router.Delete("/subscriptions", helpers.AuthDecorator(h.deletePushSubscription))
		router.Post("/test", helpers.AuthDecorator(h.pushTest))
	})

	r.With(httprate.LimitByIP(10, time.Minute)).Post("/internal/tasks/expire-messages", h.expireMessages)
}

func (h *Handler) listChatsPage(w http.ResponseWriter, r *http.Request) {
	ctx, tx, done, user, err := authTx(r)
	if err != nil {
		helpers.RenderUnauthorized(w, r)
		return
	}
	defer done()
	items, err := h.service.ListChats(ctx, tx, user.ID)
	if err != nil {
		helpers.RenderErrorPage(w, helpers.ErrorPageData{Title: "Erro", Message: err.Error(), Path: r.URL.Path})
		return
	}
	_ = helpers.Render(w, filepath.Join("app", "messages.ejs"), map[string]any{"User": user, "Chats": items, "CSRFToken": helpers.EnsureCSRFToken(w, r), "VAPIDPublicKey": strings.TrimSpace(os.Getenv("VAPID_PUBLIC_KEY"))})
}

func (h *Handler) chatPage(w http.ResponseWriter, r *http.Request) {
	ctx, tx, done, user, err := authTx(r)
	if err != nil {
		helpers.RenderUnauthorized(w, r)
		return
	}
	defer done()
	chatID := chi.URLParam(r, "chatID")
	items, err := h.service.ListChats(ctx, tx, user.ID)
	if err != nil {
		helpers.RenderErrorPage(w, helpers.ErrorPageData{Title: "Erro", Message: err.Error(), Path: r.URL.Path})
		return
	}
	msgs, err := h.service.ListMessages(ctx, tx, user.ID, chatID)
	if err != nil {
		helpers.RenderErrorPage(w, helpers.ErrorPageData{Title: "Erro", Message: err.Error(), Path: r.URL.Path})
		return
	}
	_ = helpers.Render(w, filepath.Join("app", "messages.ejs"), map[string]any{"User": user, "Chats": items, "Messages": msgs, "ActiveChatID": chatID, "CSRFToken": helpers.EnsureCSRFToken(w, r), "VAPIDPublicKey": strings.TrimSpace(os.Getenv("VAPID_PUBLIC_KEY"))})
}

func (h *Handler) sendMessage(w http.ResponseWriter, r *http.Request) {
	if !helpers.ValidateCSRFToken(r) {
		http.Error(w, "token CSRF inválido", http.StatusForbidden)
		return
	}
	ctx, tx, done, user, err := authTx(r)
	if err != nil {
		helpers.RenderUnauthorized(w, r)
		return
	}
	defer done()
	chatID, err := h.service.SendMessage(ctx, tx, user.ID, strings.TrimSpace(r.FormValue("target_user_id")), strings.TrimSpace(r.FormValue("content")), r.RemoteAddr)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, ErrNotContact) {
			status = http.StatusForbidden
		}
		http.Error(w, err.Error(), status)
		return
	}
	helpers.Redirect(w, r, "/app/messages/"+chatID)
}

func (h *Handler) registerPushSubscription(w http.ResponseWriter, r *http.Request) {
	if !helpers.ValidateCSRFToken(r) {
		http.Error(w, "token CSRF inválido", http.StatusForbidden)
		return
	}
	ctx, tx, done, user, err := authTx(r)
	if err != nil {
		helpers.RenderUnauthorized(w, r)
		return
	}
	defer done()
	var payload struct {
		Endpoint    string                        `json:"endpoint"`
		Keys        struct{ P256DH, Auth string } `json:"keys"`
		DeviceLabel string                        `json:"device_label"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "payload inválido", http.StatusBadRequest)
		return
	}
	err = h.service.RegisterPushSubscription(ctx, tx, user.ID, r.RemoteAddr, PushSubscriptionInput{Endpoint: payload.Endpoint, P256DH: payload.Keys.P256DH, Auth: payload.Keys.Auth, Device: payload.DeviceLabel, UserAgent: r.UserAgent()})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deletePushSubscription(w http.ResponseWriter, r *http.Request) {
	if !helpers.ValidateCSRFToken(r) {
		http.Error(w, "token CSRF inválido", http.StatusForbidden)
		return
	}
	ctx, tx, done, user, err := authTx(r)
	if err != nil {
		helpers.RenderUnauthorized(w, r)
		return
	}
	defer done()
	endpoint := strings.TrimSpace(r.URL.Query().Get("endpoint"))
	if endpoint == "" {
		endpoint = strings.TrimSpace(r.FormValue("endpoint"))
	}
	if err := h.service.RevokePushSubscription(ctx, tx, user.ID, endpoint, r.RemoteAddr); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) pushTest(w http.ResponseWriter, r *http.Request) {
	if !helpers.ValidateCSRFToken(r) {
		http.Error(w, "token CSRF inválido", http.StatusForbidden)
		return
	}
	ctx, tx, done, user, err := authTx(r)
	if err != nil {
		helpers.RenderUnauthorized(w, r)
		return
	}
	defer done()
	subs, err := h.service.repo.ListActivePushSubscriptions(ctx, tx, user.ID)
	if err != nil || len(subs) == 0 {
		http.Error(w, "sem subscriptions", http.StatusBadRequest)
		return
	}
	status, sendErr := h.service.notifier.NotifyMessage(ctx, subs[0], PushPayload{Title: "Teste", Body: "Push de teste", ChatID: "", URL: "/app/messages", Timestamp: time.Now().UnixMilli()})
	if sendErr != nil {
		http.Error(w, sendErr.Error(), http.StatusBadGateway)
		return
	}
	w.WriteHeader(status)
}

func (h *Handler) generateQR(w http.ResponseWriter, r *http.Request) {
	if !helpers.ValidateCSRFToken(r) {
		http.Error(w, "token CSRF inválido", http.StatusForbidden)
		return
	}
	ctx, tx, done, user, err := authTx(r)
	if err != nil {
		helpers.RenderUnauthorized(w, r)
		return
	}
	defer done()
	qr, err := h.service.GenerateContactQR(ctx, tx, user.ID, r.RemoteAddr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(qr)
}

func (h *Handler) redeemQR(w http.ResponseWriter, r *http.Request) {
	if !helpers.ValidateCSRFToken(r) {
		if wantsJSONResponse(r) {
			writeJSONResponse(w, http.StatusForbidden, jsonMessageResponse{Message: "token CSRF inválido"})
			return
		}
		http.Error(w, "token CSRF inválido", http.StatusForbidden)
		return
	}
	ctx, tx, done, user, err := authTx(r)
	if err != nil {
		helpers.RenderUnauthorized(w, r)
		return
	}
	defer done()
	chatID, err := h.service.RedeemContactQR(ctx, tx, user.ID, strings.TrimSpace(r.FormValue("token")), r.RemoteAddr)
	if err != nil {
		if wantsJSONResponse(r) {
			writeJSONResponse(w, http.StatusBadRequest, jsonMessageResponse{Message: err.Error()})
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if wantsJSONResponse(r) {
		writeJSONResponse(w, http.StatusOK, jsonMessageResponse{RedirectURL: helpers.PathURL("/app/messages/" + chatID)})
		return
	}
	helpers.Redirect(w, r, "/app/messages/"+chatID)
}

func (h *Handler) deleteAccount(w http.ResponseWriter, r *http.Request) {
	if !helpers.ValidateCSRFToken(r) {
		http.Error(w, "token CSRF inválido", http.StatusForbidden)
		return
	}
	ctx, tx, done, user, err := authTx(r)
	if err != nil {
		helpers.RenderUnauthorized(w, r)
		return
	}
	defer done()
	if strings.TrimSpace(r.FormValue("confirm")) != "EXCLUIR" {
		http.Error(w, "confirmação inválida", http.StatusBadRequest)
		return
	}
	if err := h.service.DeleteAccount(ctx, tx, user.ID, r.RemoteAddr); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	helpers.ClearAuthCookie(w)
	helpers.Redirect(w, r, "/login")
}

func (h *Handler) expireMessages(w http.ResponseWriter, r *http.Request) {
	secret := strings.TrimSpace(r.Header.Get("X-Internal-Task-Secret"))
	if secret == "" || secret != strings.TrimSpace(os.Getenv("INTERNAL_TASK_SECRET")) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()
	dbCtx, tx, done, err := models.BeginTransaction(ctx, dbTimeout)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer done()
	if err := h.service.PurgeExpiredMessages(dbCtx, tx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func authTx(r *http.Request) (context.Context, *sql.Tx, func(), *models.User, error) {
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	dbCtx, tx, done, err := models.BeginTransaction(ctx, dbTimeout)
	if err != nil {
		cancel()
		return nil, nil, nil, nil, err
	}
	user, err := helpers.GetAuthUser(dbCtx, tx, r)
	if err != nil {
		done()
		cancel()
		return nil, nil, nil, nil, err
	}
	return dbCtx, tx, func() { done(); cancel() }, user, nil
}

func wantsJSONResponse(r *http.Request) bool {
	accept := strings.ToLower(strings.TrimSpace(r.Header.Get("Accept")))
	if strings.Contains(accept, "application/json") {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Requested-With")), "fetch")
}

func writeJSONResponse(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
