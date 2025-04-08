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

func (h *ModuleHandler) GetAttempt(c *gin.Context) {
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

	// Get attempt ID from path
	attemptID := c.Param("id")
	if attemptID == "" {
		appError.ErrorInvalidInput().AbortWithErrorJson(c)
		return
	}

	attempt, xp, aerr := getAttempt(ctx, h.db, attemptID)
	if aerr != nil {
		aerr.Log(h.logger.Error().Str("handler", "module.getAttempt"), true).
			Str("userId", userId).
			Str("attemptID", attemptID).
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

func getAttempt(ctx context.Context, querier postgres.Querier, attemptID string) (*exercise.Attempt, uint, *appError.Error) {
	query := `
        SELECT 
            ma.id, ma.user_id, ma.module_id, ma.status, ma.attempts, ma.created_at, ma.updated_at, ma.user_files, ma.args, ma.results,
            COALESCE(
                CASE 
                    WHEN ma.status = 'pass' THEN ms.xp_reward 
                    ELSE NULL 
                END, 
                0
            ) as xp_reward
        FROM users.modules_attempt ma
        LEFT JOIN users.modules_submission ms ON ma.id = ms.id
        WHERE ma.id = $1
	`
	attempt := &exercise.Attempt{}
	var xp uint = 0
	err := querier.QueryRow(ctx, query,
		attemptID,
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
		&xp,
	)
	if err != nil {
		return nil, xp, appError.ErrorInternal().WithError(err)
	}
	return attempt, xp, nil
}
