package controllers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"app/helpers"
	"app/models"
	"app/payments"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
)

var errNoActiveSubscription = errors.New("nenhuma assinatura ativa encontrada")

type AppController struct {
	paymentService *payments.Service
}

func NewAppController(paymentService *payments.Service) *AppController {
	return &AppController{paymentService: paymentService}
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

			chats, _ := models.GetUserChats(dbCtx, tx, user.ID)

			RenderTemplate(w, filepath.Join("app", "dashboard.ejs"), map[string]any{
				"User":  user,
				"Chats": chats,
			})
		})

		r.Post("/app/subscription/cancel", c.handleCancelSubscription)

		r.Get("/app/subscription", func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), RequestTimeout)
			defer cancel()

			dbCtx, tx, done, err := models.BeginTransaction(ctx, DbTimeout)
			if err != nil {
				helpers.RenderErrorPage(w, helpers.ErrorPageData{Title: "Erro ao carregar assinatura", Brand: "SUPER TEMPLATE", Message: err.Error(), Path: r.URL.Path})
				return
			}
			defer done()

			user, err := helpers.GetAuthUser(dbCtx, tx, r)
			if err != nil {
				helpers.RenderUnauthorized(w, r)
				return
			}

			now := time.Now().UTC()
			inTrial, trialEnd := trialInfo(user, now)

			subscriptions, err := models.GetUserSubscriptionOverview(dbCtx, tx, user.ID)
			if err != nil {
				helpers.RenderErrorPage(w, helpers.ErrorPageData{Title: "Erro ao carregar assinatura", Brand: "SUPER TEMPLATE", Message: err.Error(), Path: r.URL.Path})
				return
			}

			subscriptionPaid, err := models.IsUserPaymentCurrent(dbCtx, tx, user.ID, now)
			if err != nil {
				helpers.RenderErrorPage(w, helpers.ErrorPageData{Title: "Erro ao validar assinatura", Brand: "SUPER TEMPLATE", Message: err.Error(), Path: r.URL.Path})
				return
			}

			_, accessReason := subscriptionAccessReason(user, now)

			paymentHistory, err := models.GetUserPaymentHistory(dbCtx, tx, user.ID)
			if err != nil {
				helpers.RenderErrorPage(w, helpers.ErrorPageData{Title: "Erro ao carregar histórico de pagamentos", Brand: "SUPER TEMPLATE", Message: err.Error(), Path: r.URL.Path})
				return
			}

			hasSubscriptionHistory := len(subscriptions) > 0
			showCompactSubscription := !inTrial && !subscriptionPaid && !hasSubscriptionHistory

			data := map[string]any{
				"User":                    user,
				"Subscriptions":           subscriptions,
				"ActiveSubscription":      latestSubscription(subscriptions),
				"PaymentHistory":          paymentHistory,
				"TrialActive":             inTrial,
				"TrialEndsAt":             trialEnd,
				"SubscriptionPaid":        subscriptionPaid,
				"ShowCompactSubscription": showCompactSubscription,
				"HasSubscriptionHistory":  hasSubscriptionHistory,
				"AccessReason":            accessReason,
			}

			RenderTemplate(w, filepath.Join("app", "subscription.ejs"), data)
		})
	})
}

func (c *AppController) handleCancelSubscription(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), RequestTimeout)
	defer cancel()

	dbCtx, tx, done, err := models.BeginTransaction(ctx, DbTimeout)
	if err != nil {
		helpers.RenderErrorPage(w, helpers.ErrorPageData{Title: "Erro ao cancelar assinatura", Brand: "SUPER TEMPLATE", Message: err.Error(), Path: r.URL.Path})
		return
	}
	defer done()

	user, err := helpers.GetAuthUser(dbCtx, tx, r)
	if err != nil {
		helpers.RenderUnauthorized(w, r)
		return
	}

	freeTo, err := c.cancelUserSubscription(dbCtx, tx, user, nil)
	if err != nil {
		if errors.Is(err, errNoActiveSubscription) {
			RenderTemplate(w, filepath.Join("partials", "alert.ejs"), map[string]any{
				"Type":    "info",
				"Message": "Nenhuma assinatura ativa foi encontrada.",
			})
			return
		}
		RenderTemplate(w, filepath.Join("partials", "alert.ejs"), map[string]any{
			"Type":    "danger",
			"Message": err.Error(),
		})
		return
	}

	message := "Assinatura cancelada com sucesso."
	if freeTo.Valid {
		message = fmt.Sprintf("Assinatura cancelada com sucesso. Seu acesso fica liberado até %s.", freeTo.Time.Format("02/01/2006"))
	}

	helpers.Redirect(w, r, "/app/subscription")
	RenderTemplate(w, filepath.Join("partials", "alert.ejs"), map[string]any{
		"Type":    "success",
		"Message": message,
	})
}

func (c *AppController) cancelUserSubscription(ctx context.Context, tx *sql.Tx, user *models.User, existingLinks []models.UserPayment) (sql.NullTime, error) {
	if c.paymentService == nil {
		return sql.NullTime{}, errors.New("serviço de pagamentos indisponível no momento")
	}

	links := existingLinks
	if len(links) == 0 {
		var err error
		links, err = models.GetUserPayments(ctx, tx, user.ID)
		if err != nil {
			return sql.NullTime{}, fmt.Errorf("falha ao buscar assinatura: %w", err)
		}
	}

	if len(links) == 0 {
		return sql.NullTime{}, errNoActiveSubscription
	}

	subscriptionID := ""
	for _, link := range links {
		if link.PaymentSubscriptionID != "" {
			subscriptionID = link.PaymentSubscriptionID
			break
		}
	}
	if subscriptionID == "" {
		return sql.NullTime{}, errors.New("assinatura inválida")
	}

	if _, err := c.paymentService.CancelSubscription(ctx, subscriptionID); err != nil {
		return sql.NullTime{}, fmt.Errorf("falha ao cancelar assinatura: %w", err)
	}

	for _, link := range links {
		if err := link.Delete(ctx, tx); err != nil {
			return sql.NullTime{}, fmt.Errorf("falha ao desvincular assinatura: %w", err)
		}
	}

	lastPaid, err := models.GetLastPaidSubscriptionDueDate(ctx, tx, user.ID)
	if err != nil {
		return sql.NullTime{}, fmt.Errorf("falha ao recuperar último pagamento: %w", err)
	}

	freeTo := lastPaid
	if freeTo.Valid {
		cycle, cycleErr := models.GetPaymentSubscriptionCycle(ctx, tx, subscriptionID)
		if cycleErr == nil && strings.EqualFold(cycle, "YEARLY") {
			freeTo.Time = lastDayOfYear(freeTo.Time)
		} else {
			freeTo.Time = endOfMonth(freeTo.Time)
		}
	}

	return lastPaid, nil
}

func latestSubscription(subscriptions []models.SubscriptionOverview) *models.SubscriptionOverview {
	if len(subscriptions) == 0 {
		return nil
	}

	latest := subscriptions[0]
	for _, subscription := range subscriptions[1:] {
		if subscription.NextDueDate.Valid && (!latest.NextDueDate.Valid || subscription.NextDueDate.Time.After(latest.NextDueDate.Time)) {
			latest = subscription
			continue
		}

		if subscription.CreatedAt.After(latest.CreatedAt) {
			latest = subscription
		}
	}

	return &latest
}

func endOfMonth(t time.Time) time.Time {
	loc := t.Location()
	y, m, _ := t.Date()
	firstNext := time.Date(y, m+1, 1, 0, 0, 0, 0, loc)
	return firstNext.Add(-time.Nanosecond)
}

func lastDayOfYear(t time.Time) time.Time {
	loc := t.Location()
	firstJanNext := time.Date(t.Year()+1, time.January, 1, 0, 0, 0, 0, loc)
	return firstJanNext.Add(-time.Nanosecond)
}
