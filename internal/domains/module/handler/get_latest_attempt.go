package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/arabkood/backend/internal/domains/module/interfaces/exercise"
	"github.com/arabkood/backend/internal/postgres"
	appError "github.com/arabkood/backend/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *ModuleHandler) GetLatestAttempt(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	id := c.Param("id")
	if id == "" {
		appError.ErrorInvalidInput().AbortWithErrorJson(c)
		return
	}

	// Get user ID from auth context
	userID, exists := c.Get("userID")
	if !exists {
		appError.ErrorUnauthorized().AbortWithErrorJson(c)
		return
	}
	userId := userID.(uuid.UUID).String()
	if userId == "" {
		appError.ErrorUnauthorized().AbortWithErrorJson(c)
		return
	}

	// Get module ID from path
	moduleID := c.Param("id")
	if moduleID == "" {
		appError.ErrorInvalidInput().AbortWithErrorJson(c)
		return
	}

	attempt, xp, aerr := getLatestAttempt(ctx, h.db, userId, moduleID)
	if aerr != nil {
		aerr.Log(h.logger.Error().Str("handler", "module.getAttempt"), true).
			Str("userId", userId).
			Str("attemptID", moduleID).
			Msg("Failed to get attempt")
		aerr.AbortWithErrorJson(c)
		return
	}

	type Response struct {
		exercise.Attempt
		XpReward uint `json:"xp_reward" db:"xp_reward"`
	}

	res := &Response{
		Attempt:  *attempt,
		XpReward: xp,
	}

	c.JSON(http.StatusOK, res)
}

func getLatestAttempt(ctx context.Context, querier postgres.Querier, userID, moduleID string) (*exercise.Attempt, uint, *appError.Error) {
	query := `
        SELECT 
            ma.id, ma.user_id, ma.module_id, ma.status, ma.attempts, ma.created_at, ma.updated_at, ma.user_files, ma.args, ma.results
        FROM users.modules_attempt ma
        WHERE ma.user_id = $1 AND ma.module_id = $2
	`
	attempt := &exercise.Attempt{}
	err := querier.QueryRow(ctx, query,
		userID,
		moduleID,
	).Scan(
		&attempt.ID,
		&attempt.UserID,
		&attempt.ModuleID,
		&attempt.Status,
		&attempt.Attempts,
		&attempt.CreatedAt,
		&attempt.UpdatedAt,
		&attempt.UserFiles,
		&attempt.Args,
		&attempt.Results,
	)
	if err != nil {
		return nil, 0, appError.ErrorInternal().WithError(err)
	}
	return attempt, 0, nil
}
