package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	itemInterface "github.com/arabkood/backend/internal/domains/item/interfaces/item"
	repoItem "github.com/arabkood/backend/internal/domains/item/repo/item"
	xpRepo "github.com/arabkood/backend/internal/domains/xp/repo"
	infraPostgres "github.com/arabkood/backend/internal/postgres"
	appError "github.com/arabkood/backend/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TestResult struct {
	Version int     `json:"version"`           // Required: Must be 3
	Status  string  `json:"status"`            // Required: "pass", "fail", or "error"
	Message *string `json:"message,omitempty"` // Required if status is "error" or when status is "fail"
	Tests   []Test  `json:"tests,omitempty"`   // Required if status is "fail" or "pass"
}
type Test struct {
	Name     string  `json:"name"`                // Required: Human-readable test name
	Status   string  `json:"status"`              // Required: "pass", "fail", or "error"
	Message  *string `json:"message,omitempty"`   // Required if status is "fail" or "error"
	Output   *string `json:"output,omitempty"`    // Optional: User output (max 500 chars)
	TestCode *string `json:"test_code,omitempty"` // Required for Concept Exercises, recommended for Practice Exercises
	TaskID   *int    `json:"task_id,omitempty"`   // Optional: Links test to a specific task
}

type PostResultRequest struct {
	Id     string     `json:"id"`
	Result TestResult `json:"result"`
}

func (h *ItemHandler) PostResult(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	var req PostResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Debug().Err(err).Send()
		appError.ErrorInvalidInput().AbortWithErrorJson(c)
		return
	}

	var res map[string]any
	inres, _ := json.Marshal(req.Result)
	if err := json.Unmarshal(inres, &res); err != nil {
		h.logger.Debug().Err(err).Send()
		appError.ErrorInvalidInput().AbortWithErrorJson(c)
		return
	}

	// Parse submission ID
	submissionID, err := uuid.Parse(req.Id)
	if err != nil {
		h.logger.Debug().Err(err).Send()
		appError.ErrorInvalidInput().AbortWithErrorJson(c)
		return
	}

	// Start transaction
	tx, err := h.db.Begin(ctx)
	if err != nil {
		h.logger.Error().
			Err(err).
			Msg("Failed to begin transaction for result update")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	// Update submission with results
	updatedSubmission, aerr := updateSubmissionWithResults(ctx, tx, submissionID, req.Result.Status, res)
	if aerr != nil {
		aerr.Log(h.logger.Error().Str("handler", "runner.result"), true).
			Str("id", submissionID.String()).
			Msg("Failed to update submission with results")
		aerr.AbortWithErrorJson(c)
		return
	}

	// If the submission passed, handle XP reward
	if updatedSubmission.Status == "pass" && updatedSubmission.XPReward > 0 {
		// Get item details for XP event
		itemRepo := repoItem.NewItemRepository(tx)
		item, aerr := itemRepo.GetItemById(ctx, updatedSubmission.ItemID.String())
		if aerr != nil {
			aerr.Log(h.logger.Error().Str("handler", "runner.result"), true).
				Str("itemId", updatedSubmission.ItemID.String()).
				Msg("Failed to get item details for XP event")
			aerr.AbortWithErrorJson(c)
			return
		}

		xpEvent := &xpRepo.XPEvent{
			UserID:     updatedSubmission.UserID,
			XPAmount:   updatedSubmission.XPReward,
			SourceType: fmt.Sprintf("%s/%s", "item/", *item.Type),
			SourceID:   item.ID,
		}
		err := xpRepo.AddXP(ctx, tx, *xpEvent)
		if err != nil {
			h.logger.Error().
				Err(err).
				Str("error", err.Error()).
				Msg("Failed to add XP for successful submission")
			appError.ErrorInternal().AbortWithErrorJson(c)
			return
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		h.logger.Error().
			Err(err).
			Msg("Couldn't commit transaction")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	c.Status(http.StatusOK)
}

func updateSubmissionWithResults(ctx context.Context, querier infraPostgres.Querier, submissionID uuid.UUID, status string, results map[string]any) (*itemInterface.Submission, *appError.Error) {
	query := `
		UPDATE users.submission 
		SET 
			status = $2,
			results = $3,
			attempts = attempts + 1,
			updated_at = NOW()
		WHERE id = $1
		RETURNING 
			id, user_id, item_id,
			status, xp_reward, attempts,
			metadata, data, results,
			created_at, updated_at`

	var submission itemInterface.Submission
	err := querier.QueryRow(ctx, query,
		submissionID,
		status,
		results,
	).Scan(
		&submission.ID,
		&submission.UserID,
		&submission.ItemID,
		&submission.Status,
		&submission.XPReward,
		&submission.Attempts,
		&submission.Metadata,
		&submission.Data,
		&submission.Results,
		&submission.CreatedAt,
		&submission.UpdatedAt,
	)
	if err != nil {
		return nil, appError.ErrorInternal().WithError(err)
	}

	return &submission, nil
}
