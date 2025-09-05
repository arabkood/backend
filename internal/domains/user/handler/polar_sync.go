package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/arabkood/backend/internal/domains/user/repo"
	appError "github.com/arabkood/backend/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type UpdatePlanRequest struct {
	UserID               string    `json:"userId" binding:"required,uuid"`
	PremiumActive        bool      `json:"premium_active"`
	PolarLastSyncedAt    time.Time `json:"polar_last_synced_at"`
	PolarCustomerID      *string   `json:"polar_customer_id"`
	PolarSubscriptionIDs []string  `json:"polar_subscription_ids"`
}

type UpdatePlanResponse struct {
	ID                   string    `json:"id"`
	PremiumActive        bool      `json:"premium_active"`
	PolarLastSyncedAt    time.Time `json:"polar_last_synced_at"`
	PolarCustomerID      *string   `json:"polar_customer_id"`
	PolarSubscriptionIDs []string  `json:"polar_subscription_ids"`
}

func (h *UserHandler) PolarSync(c *gin.Context) {
	logError := func() *zerolog.Event {
		return h.logger.Error().Str("handler", "user.PolarSync")
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Bind request body
	var req UpdatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appError.ErrorInvalidInput().AbortWithErrorJson(c)
		return
	}

	// Start transaction
	tx, err := h.db.Begin(ctx)
	if err != nil {
		logError().Err(err).Msg("Failed to begin transaction for update plan")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	// Create repository instance
	userRepo := repo.NewUserRepository(tx)

	// Get user
	uid, _ := uuid.Parse(req.UserID)
	user, aerr := userRepo.GetByID(ctx, uid)
	if aerr != nil {
		if aerr.Type == appError.ErrUserNotFound {
			appError.ErrorUnauthorized().AbortWithErrorJson(c)
			return
		}
		aerr.Log(logError(), true).Msg("Failed to get user by ID")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// Update subscription fields
	user.PremiumActive = req.PremiumActive
	user.PolarLastSyncedAt = &req.PolarLastSyncedAt
	if req.PolarCustomerID != nil {
		if cid, err := uuid.Parse(*req.PolarCustomerID); err == nil {
			user.PolarCustomerID = &cid
		}
	}
	if len(req.PolarSubscriptionIDs) > 0 {
		subIDs := make([]uuid.UUID, 0, len(req.PolarSubscriptionIDs))
		for _, s := range req.PolarSubscriptionIDs {
			if sid, err := uuid.Parse(s); err == nil {
				subIDs = append(subIDs, sid)
			}
		}
		user.PolarSubscriptionIDs = subIDs
	}

	// Update user in database
	if aerr := userRepo.Update(ctx, user); aerr != nil {
		aerr.Log(logError(), true).Msg("Failed to update user plan")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		logError().Err(err).Msg("Couldn't commit transaction")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// Return response
	res := UpdatePlanResponse{
		ID:                   user.ID.String(),
		PremiumActive:        user.PremiumActive,
		PolarLastSyncedAt:    *user.PolarLastSyncedAt,
		PolarCustomerID:      req.PolarCustomerID,
		PolarSubscriptionIDs: req.PolarSubscriptionIDs,
	}
	c.JSON(http.StatusOK, res)
}
