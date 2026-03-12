package controllers

import (
	"context"
	"net/http"
	"time"

	"app/helpers"
	"app/models"
)

func trialInfo(user *models.User, now time.Time) (bool, time.Time) {
	if user == nil || user.CreatedAt.IsZero() {
		return false, time.Time{}
	}

	trialEnd := user.CreatedAt.AddDate(0, 0, 7)
	return !now.After(trialEnd), trialEnd
}

func RequireActiveSubscription(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), RequestTimeout)
		defer cancel()

		dbCtx, tx, done, err := models.BeginTransaction(ctx, DbTimeout)
		if err != nil {
			helpers.RenderErrorPage(w, helpers.ErrorPageData{Title: "Erro interno", Brand: "SUPER TEMPLATE", Message: err.Error(), Path: r.URL.Path})
			return
		}
		defer done()

		user, err := helpers.GetAuthUser(dbCtx, tx, r)
		if err != nil {
			helpers.RenderUnauthorized(w, r)
			return
		}

		now := time.Now().UTC()
		inTrial, _ := trialInfo(user, now)
		if inTrial {
			next.ServeHTTP(w, r)
			return
		}

		if allowed, _ := subscriptionAccessReason(user, now); allowed {
			next.ServeHTTP(w, r)
			return
		}

		currentPayment, err := models.IsUserPaymentCurrent(dbCtx, tx, user.ID, now)
		if err != nil {
			helpers.RenderErrorPage(w, helpers.ErrorPageData{Title: "Erro interno", Brand: "SUPER TEMPLATE", Message: err.Error(), Path: r.URL.Path})
			return
		}

		if currentPayment {
			next.ServeHTTP(w, r)
			return
		}

		helpers.Redirect(w, r, "/app/subscription")
		return
	})
}

func subscriptionAccessReason(user *models.User, now time.Time) (bool, string) {
	if user == nil {
		return false, ""
	}

	return false, ""
}
