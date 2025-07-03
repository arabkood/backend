package handler

import (
	"context"
	"encoding/json"
	"errors"
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
	"github.com/jackc/pgx/v5"
)

// TestResult represents the detailed test outcomes from the runner.
type TestResult struct {
	Version int     `json:"version"`
	Status  string  `json:"status"`
	Message *string `json:"message,omitempty"`
	Tests   []Test  `json:"tests,omitempty"`
}

// Test represents an individual test case.
type Test struct {
	Name     string  `json:"name"`
	Status   string  `json:"status"`
	Message  *string `json:"message,omitempty"`
	Output   *string `json:"output,omitempty"`
	TestCode *string `json:"test_code,omitempty"`
	TaskID   *int    `json:"task_id,omitempty"`
}

type JobResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	Error    string `json:"error"`
	ExitCode int    `json:"exit_code"`
}

type PostResultRequest struct {
	Id           string      `json:"id"`
	RunnerResult *TestResult `json:"result,omitempty"`
	JobResult    *JobResult  `json:"job,omitempty"`
}

// PostResult receives the result from the invoker, which can contain
// a test result, a job result, or both.
func (h *ItemHandler) PostResult(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	var req PostResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.Logger.Debug().Err(err).Send()
		appError.ErrorInvalidInput().AbortWithErrorJson(c)
		return
	}

	if req.RunnerResult == nil && req.JobResult == nil {
		h.Logger.Debug().Msg("PostResult request received with no 'result' or 'job' field")
		appError.ErrorInvalidInput().WithMessage("Request must contain either a 'result' or a 'job' field.").AbortWithErrorJson(c)
		return
	}

	// Parse submission ID
	submissionID, err := uuid.Parse(req.Id)
	if err != nil {
		h.Logger.Debug().Err(err).Send()
		appError.ErrorInvalidInput().AbortWithErrorJson(c)
		return
	}

	var submissionStatus string
	var resultsPayload interface{}

	if req.RunnerResult != nil {
		// The runner result dictates the primary status and populates the 'results' field.
		submissionStatus = req.RunnerResult.Status
		resultsPayload = req.RunnerResult
	} else {
		// If no runner result, the job result dictates the status is 'error'.
		// The 'results' field will be null. The job data goes into metadata.
		submissionStatus = "error"
		resultsPayload = nil
	}

	// Convert the runner result payload to a map[string]any for the database function.
	var resultsMap map[string]any
	if resultsPayload != nil {
		resultsBytes, err := json.Marshal(resultsPayload)
		if err != nil {
			h.Logger.Error().Err(err).Msg("Failed to marshal results payload")
			appError.ErrorInternal().AbortWithErrorJson(c)
			return
		}
		if err := json.Unmarshal(resultsBytes, &resultsMap); err != nil {
			h.Logger.Error().Err(err).Msg("Failed to unmarshal results payload into map")
			appError.ErrorInternal().AbortWithErrorJson(c)
			return
		}
	}

	// Start transaction
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		h.Logger.Error().Err(err).Msg("Failed to begin transaction for result update")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	// Update submission with runner result in 'results' and job result in 'metadata.job'.
	updatedSubmission, aerr := updateSubmissionWithResults(ctx, tx, submissionID, submissionStatus, resultsMap, req.JobResult)
	if aerr != nil {
		aerr.Log(h.Logger.Error().Str("handler", "runner.result"), true).
			Str("id", submissionID.String()).
			Msg("Failed to update submission with results")
		aerr.AbortWithErrorJson(c)
		return
	}

	// *** Only award XP if it was a runner result that passed ***
	if req.RunnerResult != nil && updatedSubmission.Status == "pass" {
		// Get item details for XP event
		itemRepo := repoItem.NewItemRepository(tx)
		item, aerr := itemRepo.GetItemById(ctx, updatedSubmission.ItemID.String())
		if aerr != nil {
			aerr.Log(h.Logger.Error().Str("handler", "runner.result"), true).
				Str("itemId", updatedSubmission.ItemID.String()).
				Msg("Failed to get item details for XP event")
			aerr.AbortWithErrorJson(c)
			return
		}

		// Atomically update xp_reward ONLY if it's the first pass
		var xpAwardedNow int
		err := tx.QueryRow(ctx,
			`UPDATE users.submission
			 SET xp_reward = $1
			 WHERE id = $2 AND xp_reward = 0
			 RETURNING xp_reward`,
			item.BaseXP,
			updatedSubmission.ID,
		).Scan(&xpAwardedNow)

		// If no rows were returned, it means xp_reward was already set (not a first pass). This is not an error.
		// Any other error is a problem.
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			h.Logger.Error().Err(err).Msg("Failed to update submission with XP reward")
			appError.ErrorInternal().AbortWithErrorJson(c)
			return
		}

		// Only add to the XP ledger if we just awarded XP in this transaction.
		if xpAwardedNow > 0 {
			xpEvent := &xpRepo.XPEvent{
				UserID:     updatedSubmission.UserID,
				XPAmount:   xpAwardedNow,
				SourceType: fmt.Sprintf("item/%s", *item.Type),
				SourceID:   item.ID,
			}
			if err := xpRepo.AddXP(ctx, tx, *xpEvent); err != nil {
				h.Logger.Error().Err(err).Msg("Failed to add XP for successful submission")
				appError.ErrorInternal().AbortWithErrorJson(c)
				return
			}
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		h.Logger.Error().Err(err).Msg("Couldn't commit transaction")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	c.Status(http.StatusOK)
}

func updateSubmissionWithResults(ctx context.Context, querier infraPostgres.Querier, submissionID uuid.UUID, status string, results map[string]any, jobResult *JobResult) (*itemInterface.Submission, *appError.Error) {
	query := `
		UPDATE users.submission
		SET
			status = $2,
			results = $3,
			metadata = CASE
				WHEN $4::jsonb IS NOT NULL
				THEN jsonb_set(COALESCE(metadata, '{}'::jsonb), '{job}', $4::jsonb, true)
				ELSE metadata
			END,
			attempts = attempts + 1,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, user_id, item_id, status, xp_reward, attempts, metadata, data, results, created_at, updated_at
	`

	var submission itemInterface.Submission
	err := querier.QueryRow(ctx, query,
		submissionID,
		status,
		results,
		jobResult,
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
		// Use a more specific error check for "no rows" if needed,
		// which could indicate a bad submission ID.
		return nil, appError.ErrorInternal().WithError(err)
	}

	return &submission, nil
}
