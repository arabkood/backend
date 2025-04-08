package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/arabkood/backend/internal/domains/module/interfaces/exercise"
	moduleRepo "github.com/arabkood/backend/internal/domains/module/repo/module"
	"github.com/arabkood/backend/internal/postgres"
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

func (h *ModuleHandler) PostResult(c *gin.Context) {
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

	// Update attempt row in DB
	reqId, err := uuid.Parse(req.Id)
	if err != nil {
		h.logger.Debug().Err(err).Send()
		appError.ErrorInvalidInput().AbortWithErrorJson(c)
		return
	}
	newAttempt := &exercise.Attempt{
		ID:      reqId,
		Status:  req.Result.Status,
		Results: res,
	}
	updatedAttempt, aerr := updateAttempt(ctx, h.db, newAttempt)
	if aerr != nil {
		aerr.Log(h.logger.Error().Str("handler", "runner.attempt"), true).
			Str("id", newAttempt.ID.String()).
			Msg("Failed to update attempt")
		aerr.AbortWithErrorJson(c)
		return
	}
	if updatedAttempt.Status == "pass" {
		aerr := handleSuccess(ctx, h.db, updatedAttempt)
		if aerr != nil {
			aerr.Log(h.logger.Error().Str("handler", "runner.attempt"), true).
				Str("id", updatedAttempt.ID.String()).
				Msg("Failed to handle success attempt")
			aerr.AbortWithErrorJson(c)
			return
		}
	}

	c.Status(http.StatusOK)
}

func handleSuccess(ctx context.Context, querier postgres.Querier, attempt *exercise.Attempt) *appError.Error {
	// Get XP amount to reward user
	mr := moduleRepo.NewModuleRepository(querier)
	moduleID := attempt.ModuleID.String()
	if moduleID == "" || moduleID == "00000000-0000-0000-0000-000000000000" {
		return appError.ErrorInternal().WithMessage("Couldn't parse moduleID")
	}
	module, err := mr.GetModuleById(ctx, moduleID)
	if err != nil {
		return err
	}
	// Try to insert now submission
	submission := &exercise.Submission{
		ID:       attempt.ID,
		UserID:   attempt.UserID,
		ModuleID: attempt.ModuleID,

		XpReward: int64(module.XPReward),
		Attempts: attempt.Attempts,

		UserFiles: attempt.UserFiles,
		Args:      attempt.Args,
		Results:   attempt.Results,
	}
	_, err = insertSubmission(ctx, querier, submission)
	if err != nil {
		// update submission instead
		err = updateSubmission(ctx, querier, submission)
		if err != nil {
			return err
		}
		return nil
	}
	_, err = upsertDailyStats(ctx, querier, attempt.UserID.String(), module.XPReward)
	if err != nil {
		return err
	}
	err = upsertStats(ctx, querier, attempt.UserID.String(), module.XPReward)
	if err != nil {
		return err
	}

	return nil
}

func upsertStats(ctx context.Context, querier postgres.Querier, userId string, xp_earned int) *appError.Error {
	query := `
    INSERT INTO users.stats (
        user_id, total_xp, completed_items, last_active_at
    ) VALUES (
        $1, $2, 1, NOW()
    )
    ON CONFLICT (user_id) DO UPDATE
    SET total_xp = users.stats.total_xp + $2,
        completed_items = users.stats.completed_items + 1,
        last_active_at = NOW();
    `
	_, err := querier.Exec(ctx, query,
		userId,
		xp_earned,
	)
	if err != nil {
		return appError.ErrorInternal().WithError(err)
	}
	return nil
}

func upsertDailyStats(ctx context.Context, querier postgres.Querier, userId string, xp_earned int) (*time.Time, *appError.Error) {
	query := `
    INSERT INTO users.daily_stats (
        user_id, date, xp_earned, items_completed
    ) VALUES (
        $1, CURRENT_DATE, $2, 1
    )
    ON CONFLICT (user_id, date) DO UPDATE
    SET xp_earned = users.daily_stats.xp_earned + $2, 
        items_completed = users.daily_stats.items_completed + 1
    RETURNING date`
	var date *time.Time
	err := querier.QueryRow(ctx, query,
		userId,
		xp_earned,
	).Scan(&date)
	if err != nil {
		return nil, appError.ErrorInternal().WithError(err)
	}
	return date, nil
}

func updateAttempt(ctx context.Context, querier postgres.Querier, attempt *exercise.Attempt) (*exercise.Attempt, *appError.Error) {
	query := `
        UPDATE users.modules_attempt 
        SET 
            status = $2,
            results = $3,
            attempts = attempts + 1,
            updated_at = NOW()
        WHERE id = $1
        RETURNING id, user_id, module_id, status, attempts, created_at, updated_at, user_files, args, results`

	updatedAttempt := &exercise.Attempt{}
	err := querier.QueryRow(ctx, query,
		attempt.ID,
		attempt.Status,
		attempt.Results,
	).Scan(
		&updatedAttempt.ID,
		&updatedAttempt.UserID,
		&updatedAttempt.ModuleID,
		&updatedAttempt.Status,
		&updatedAttempt.Attempts,
		&updatedAttempt.CreatedAt,
		&updatedAttempt.UpdatedAt,
		&updatedAttempt.UserFiles,
		&updatedAttempt.Args,
		&updatedAttempt.Results,
	)
	if err != nil {
		return nil, appError.ErrorInternal().WithError(err)
	}

	return updatedAttempt, nil
}

func insertSubmission(ctx context.Context, querier postgres.Querier, sub *exercise.Submission) (string, *appError.Error) {
	query := `
         INSERT INTO users.modules_submission (
              id, user_id, module_id,
              user_files, args, results,
							xp_reward, attempts
         ) VALUES (
             $1, $2, $3,
             $4, $5, $6,
						 $7, $8
         )
         RETURNING id`
	var id string
	err := querier.QueryRow(ctx, query,
		sub.ID,
		sub.UserID,
		sub.ModuleID,
		sub.UserFiles,
		sub.Args,
		sub.Results,
		sub.XpReward,
		sub.Attempts,
	).Scan(&id)
	if err != nil {
		return "", appError.ErrorInternal().WithError(err)
	}
	return id, nil
}

func updateSubmission(ctx context.Context, querier postgres.Querier, sub *exercise.Submission) *appError.Error {
	query := `
         UPDATE users.modules_submission SET
              user_files = $1,
              args = $2,
              results = $3,
              attempts = $4,
							id = $5
         WHERE user_id = $6 AND module_id = $7`

	_, err := querier.Exec(ctx, query,
		sub.UserFiles,
		sub.Args,
		sub.Results,
		sub.Attempts,
		sub.ID,
		sub.UserID,
		sub.ModuleID,
	)
	if err != nil {
		return appError.ErrorInternal().WithError(err)
	}
	return nil
}
