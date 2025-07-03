package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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

type DataType map[string]any

type SubmitRequest struct {
	Data DataType `form:"data" binding:"required" json:"data"`
}

type SubmitResponse struct {
	Submission *itemInterface.Submission `json:"submission"`
}

func (h *ItemHandler) Submit(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// 1. Authentication & Authorization
	// Get user ID from auth context
	userIDAny, exists := c.Get("userID")
	if !exists {
		appError.ErrorUnauthorized().AbortWithErrorJson(c)
		return
	}
	userID := userIDAny.(uuid.UUID)

	itemId := c.Param("itemId")
	if itemId == "" {
		appError.ErrorInvalidInput().WithMessage("Item ID is required").AbortWithErrorJson(c)
		return
	}

	// 2. Input Binding & Validation
	var req SubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appError.ErrorInvalidInput().WithError(err).AbortWithErrorJson(c)
		return
	}

	// 3. Fetch Core Data (Item and Previous Submission)
	itemRepo := repoItem.NewItemRepository(h.DB)
	item, aerr := itemRepo.GetItemById(ctx, itemId)
	if aerr != nil {
		aerr.Log(h.Logger.Error().Str("handler", "Submit"), true).Msg("Failed to get item by ID")
		aerr.AbortWithErrorJson(c)
		return
	}

	oldSubmission, aerr := getSubmission(ctx, h.DB, userID, itemId)
	if aerr != nil {
		aerr.Log(h.Logger.Error().Str("handler", "Submit"), true).Msg("Failed to get previous submission")
		aerr.AbortWithErrorJson(c)
		return
	}

	// 4. Business Logic: Check if user can resubmit
	if oldSubmission != nil && oldSubmission.Status == "pass" {
		c.JSON(http.StatusConflict, SubmitResponse{Submission: oldSubmission})
		return
	}

	// 5. Start Track (ensures user is enrolled)
	if _, _, aerr := startTrackFromModule(ctx, h.DB, userID, item.ModuleID); aerr != nil {
		aerr.Log(h.Logger.Error().Str("handler", "Submit"), true).Msg("Failed to start track")
		aerr.AbortWithErrorJson(c)
		return
	}

	// 6. Process Submission Based on Item Type
	var newSubmission *itemInterface.Submission
	var processingErr *appError.Error

	switch *item.Type {
	case "code":
		newSubmission, processingErr = h.processCodeSubmission(ctx, item, userID, oldSubmission, req.Data)
	case "lesson":
		newSubmission, processingErr = h.processLessonSubmission(ctx, item, userID, oldSubmission, req.Data)
	default:
		log.Printf("CRITICAL: Item with invalid type submitted. ItemID: %s, Type: %s", item.ID, *item.Type)
		processingErr = appError.ErrorInternal().WithMessage("Invalid item type encountered")
	}

	if processingErr != nil {
		processingErr.Log(h.Logger.Error().Str("handler", "Submit"), true).Msg("Failed to process submission")
		processingErr.AbortWithErrorJson(c)
		return
	}

	// 7. Return Response to User
	c.JSON(http.StatusOK, SubmitResponse{Submission: newSubmission})
}

func (h *ItemHandler) processCodeSubmission(ctx context.Context, item *itemInterface.Item, userID uuid.UUID, oldSubmission *itemInterface.Submission, data DataType) (*itemInterface.Submission, *appError.Error) {
	// 1. Create a new Submission record in the database with "pending" status.
	newSubmissionID, err := uuid.NewRandom()
	if err != nil {
		return nil, appError.ErrorInternal().WithError(err).WithMessage("Failed to generate submission ID")
	}

	attempts := 1
	if oldSubmission != nil {
		attempts = oldSubmission.Attempts + 1
	}

	// The `data` field now serves as a historical record of what the user submitted.
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, appError.ErrorInternal().WithError(err).WithMessage("Failed to marshal submission data")
	}

	submission := &itemInterface.Submission{
		ID:       newSubmissionID,
		ItemID:   item.ID,
		UserID:   userID,
		Data:     dataBytes,
		Status:   "pending",
		XPReward: 0,
		Attempts: attempts,
	}

	// We use an upsert here to create/update the submission record in one go.
	if _, aerr := upsertSubmission(ctx, h.DB, submission); aerr != nil {
		return nil, aerr
	}

	// 2. Enqueue the task for the invoker.
	_, aerr := handleCodeSubmission(ctx, h, data, item, userID, newSubmissionID)
	if aerr != nil {
		h.Logger.Error().Str("error", aerr.Err.Error()).Str("msg", *aerr.Message).Send()
		// If queueing fails, we update the submission status to "internal".
		submission.Status = "internal"
		if _, aerr2 := upsertSubmission(ctx, h.DB, submission); aerr != nil {
			return submission, aerr2
		}
		return nil, aerr
	}

	return submission, nil
}

func (h *ItemHandler) processLessonSubmission(ctx context.Context, item *itemInterface.Item, userID uuid.UUID, oldSubmission *itemInterface.Submission, data DataType) (*itemInterface.Submission, *appError.Error) {
	percent, err := handleLesson(data)
	if err != nil {
		return nil, appError.ErrorInvalidInput().WithError(err).WithMessage("Invalid lesson data")
	}

	newSubmissionID, err := uuid.NewRandom()
	if err != nil {
		return nil, appError.ErrorInternal().WithError(err).WithMessage("Failed to generate submission ID")
	}

	attempts := 1
	if oldSubmission != nil {
		attempts = oldSubmission.Attempts + 1
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, appError.ErrorInternal().WithError(err).WithMessage("Failed to marshal submission data")
	}

	submission := &itemInterface.Submission{
		ID:       newSubmissionID,
		ItemID:   item.ID,
		UserID:   userID,
		Data:     dataBytes,
		Attempts: attempts,
	}

	// Determine status and XP
	if percent > 50 {
		submission.Status = "pass"
		xpMultiplier := float64(percent) / 100.0
		submission.XPReward = int(float64(item.BaseXP) * xpMultiplier)
	} else {
		submission.Status = "fail"
		submission.XPReward = 0
	}

	// Use a single transaction for submission and XP to ensure atomicity.
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		return nil, appError.ErrorInternal().WithError(err).WithMessage("Failed to begin transaction")
	}
	defer tx.Rollback(context.Background()) // Defer rollback in case of panic or early return

	if _, aerr := upsertSubmission(ctx, tx, submission); aerr != nil {
		return nil, aerr
	}

	if submission.Status == "pass" && submission.XPReward > 0 {
		xpEvent := &xpRepo.XPEvent{
			UserID:     userID,
			XPAmount:   submission.XPReward,
			SourceType: fmt.Sprintf("item/%s", *item.Type),
			SourceID:   item.ID,
		}
		if err := xpRepo.AddXP(ctx, tx, *xpEvent); err != nil {
			return nil, appError.ErrorInternal().WithError(err).WithMessage("Failed to add XP")
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, appError.ErrorInternal().WithError(err).WithMessage("Failed to commit transaction")
	}

	return submission, nil
}

func getSubmission(ctx context.Context, querier infraPostgres.Querier, userID uuid.UUID, itemID string) (*itemInterface.Submission, *appError.Error) {
	query := `
		SELECT 
			id, user_id, item_id,
			status, xp_reward, attempts,
			metadata, data, results
		FROM users.submission
		WHERE user_id = $1 AND item_id = $2
		LIMIT 1`

	row := querier.QueryRow(ctx, query, userID, itemID)

	var submission itemInterface.Submission
	err := row.Scan(
		&submission.ID,
		&submission.UserID,
		&submission.ItemID,
		&submission.Status,
		&submission.XPReward,
		&submission.Attempts,
		&submission.Metadata,
		&submission.Data,
		&submission.Results,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, appError.ErrorInternal().WithError(err)
	}

	return &submission, nil
}

func startTrackFromModule(ctx context.Context, querier infraPostgres.Querier, userID, moduleID uuid.UUID) (string, string, *appError.Error) {
	query := `
		WITH track_for_module AS (
			SELECT track_id FROM class.modules WHERE id = $1
		), insert_user_track AS (
			INSERT INTO users.track (user_id, track_id)
			SELECT $2, track_id FROM track_for_module
			ON CONFLICT (user_id, track_id) DO NOTHING
			RETURNING user_id, track_id
		)
		SELECT user_id, track_id FROM insert_user_track
		UNION
		SELECT $2, track_id FROM track_for_module`

	var returnedUserID, returnedTrackID string
	err := querier.QueryRow(ctx, query, moduleID, userID).Scan(&returnedUserID, &returnedTrackID)
	if err != nil {
		return "", "", appError.ErrorInternal().WithError(err)
	}
	return returnedUserID, returnedTrackID, nil
}

func upsertSubmission(ctx context.Context, querier infraPostgres.Querier, submission *itemInterface.Submission) (string, *appError.Error) {
	query := `
    INSERT INTO users.submission (
        user_id, item_id,
        status, xp_reward, attempts,
        metadata, data, results,
				id
    ) VALUES (
        $1, $2,
        $3, $4, $5,
        $6, $7, $8,
				$9
    )
    ON CONFLICT (user_id, item_id) DO UPDATE
    SET 
        status = EXCLUDED.status,
        xp_reward = EXCLUDED.xp_reward,
        attempts = EXCLUDED.attempts,
        metadata = EXCLUDED.metadata,
        data = EXCLUDED.data,
        results = EXCLUDED.results,
			  id = EXCLUDED.id
    RETURNING id`
	var id string
	err := querier.QueryRow(ctx, query,
		submission.UserID,
		submission.ItemID,
		submission.Status,
		submission.XPReward,
		submission.Attempts,
		submission.Metadata,
		submission.Data,
		submission.Results,
		submission.ID,
	).Scan(&id)
	if err != nil {
		return "", appError.ErrorInternal().WithError(err)
	}
	return id, nil
}
