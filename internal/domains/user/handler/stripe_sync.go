package handler

import (
	"context"
	"net/http"
	"time"

	domainUser "github.com/arabkood/backend/internal/domains/user/interfaces/user"
	"github.com/arabkood/backend/internal/domains/user/repo"
	appError "github.com/arabkood/backend/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type StripeSyncRequest struct {
	UserID               string     `json:"userId" binding:"required,uuid"`
	StripeCustomerID     string     `json:"stripe_customer_id" binding:"required"`
	StripeSubscriptionID *string    `json:"stripe_subscription_id"`
	Plan                 string     `json:"plan" binding:"required,oneof=free pro past_due"`
	PlanInterval         *string    `json:"plan_interval" binding:"omitempty,oneof=monthly yearly"`
	ProUntil             *time.Time `json:"pro_until"`
	CancelAtPeriodEnd    bool       `json:"cancel_at_period_end"`
}

type StripeSyncResponse struct {
	UserID               string     `json:"user_id"`
	StripeCustomerID     string     `json:"stripe_customer_id"`
	StripeSubscriptionID *string    `json:"stripe_subscription_id,omitempty"`
	Plan                 string     `json:"plan"`
	PlanInterval         *string    `json:"plan_interval,omitempty"`
	ProUntil             *time.Time `json:"pro_until,omitempty"`
	CancelAtPeriodEnd    bool       `json:"cancel_at_period_end"`
	IsPro                bool       `json:"is_pro"`
}

// StripeSync upserts a user's subscription state from Stripe into
// auth.user_subscriptions. Called by the internal-only billing endpoint.
func (h *UserHandler) StripeSync(c *gin.Context) {
	logError := func() *zerolog.Event {
		return h.logger.Error().Str("handler", "user.StripeSync")
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Bind request body
	var req StripeSyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appError.ErrorInvalidInput().AbortWithErrorJson(c)
		return
	}

	uid, err := uuid.Parse(req.UserID)
	if err != nil {
		appError.ErrorInvalidInput().AbortWithErrorJson(c)
		return
	}

	// Start transaction
	tx, err := h.db.Begin(ctx)
	if err != nil {
		logError().Err(err).Msg("Failed to begin transaction for stripe sync")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	userRepo := repo.NewUserRepository(tx)
	subRepo := repo.NewSubscriptionRepository(tx)

	// Ensure the user exists before writing a subscription for them.
	if _, aerr := userRepo.GetByID(ctx, uid); aerr != nil {
		if aerr.Type == appError.ErrUserNotFound {
			appError.ErrorUserNotFound().AbortWithErrorJson(c)
			return
		}
		aerr.Log(logError(), true).Msg("Failed to get user by ID")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	sub := &domainUser.Subscription{
		UserID:               uid,
		StripeCustomerID:     req.StripeCustomerID,
		StripeSubscriptionID: req.StripeSubscriptionID,
		Plan:                 domainUser.PlanType(req.Plan),
		ProUntil:             req.ProUntil,
		CancelAtPeriodEnd:    req.CancelAtPeriodEnd,
	}
	if req.PlanInterval != nil {
		interval := domainUser.PlanInterval(*req.PlanInterval)
		sub.PlanInterval = &interval
	}

	if aerr := subRepo.Upsert(ctx, sub); aerr != nil {
		aerr.Log(logError(), true).Msg("Failed to upsert user subscription")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		logError().Err(err).Msg("Couldn't commit transaction")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	res := StripeSyncResponse{
		UserID:               sub.UserID.String(),
		StripeCustomerID:     sub.StripeCustomerID,
		StripeSubscriptionID: sub.StripeSubscriptionID,
		Plan:                 string(sub.Plan),
		PlanInterval:         req.PlanInterval,
		ProUntil:             sub.ProUntil,
		CancelAtPeriodEnd:    sub.CancelAtPeriodEnd,
		IsPro:                sub.IsPro(),
	}
	c.JSON(http.StatusOK, res)
}
