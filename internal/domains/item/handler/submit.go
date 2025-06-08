package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
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
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Get user ID from auth context
	userID, exists := c.Get("userID")
	if !exists {
		appError.ErrorUnauthorized().AbortWithErrorJson(c)
		return
	}

	itemId := c.Param("itemId")
	if itemId == "" {
		appError.ErrorInvalidInput().AbortWithErrorJson(c)
		return
	}

	// Bind request body
	var req SubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Debug().Err(err).Send()
		appError.ErrorInvalidInput().AbortWithErrorJson(c)
		return
	}

	dataBytes := new(bytes.Buffer)
	err := json.NewEncoder(dataBytes).Encode(req.Data)
	// 30KB limit
	if dataBytes.Len() > 30*1024 || err != nil {
		h.logger.Error().Str("handler", "runner.submit").
			Int("approximate size in KB", dataBytes.Len()/1024).
			Err(err).
			Str("item", itemId).
			Msg("User submission is too big")
		appError.ErrorInvalidInput().WithMessage("Files submitted are too big, limit is 30KB").AbortWithErrorJson(c)
		return
	}

	// Create repository instance
	itemRepo := repoItem.NewItemRepository(h.db)

	// 1. Get previous submittion if exists
	oldsubmission, aerr := getSubmission(ctx, h.db, userID.(uuid.UUID), itemId)
	if aerr != nil {
		aerr.Log(h.logger.Error().Str("handler", "runner.submit"), true).
			Str("id", itemId).
			Msg("Failed to get submission by user ID & item ID")
		aerr.AbortWithErrorJson(c)
		return
	}

	// 2. Get item
	item, aerr := itemRepo.GetItemById(ctx, itemId)
	if aerr != nil {
		aerr.Log(h.logger.Error().Str("handler", "runner.submit"), true).
			Str("id", itemId).
			Msg("Failed to get item by ID")
		aerr.AbortWithErrorJson(c)
		return
	}

	// Create or update attempt row in DB
	newSubmissionID, err := uuid.NewRandom()
	if err != nil {
		h.logger.Error().Str("handler", "runner.submit").
			Err(err).
			Msg("Couldn't generate random number")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	newSubmission := &itemInterface.Submission{
		ID:       newSubmissionID,
		ItemID:   item.ID,
		UserID:   userID.(uuid.UUID),
		Data:     dataBytes.Bytes(),
		Status:   "wait",
		XPReward: item.BaseXP,
		Attempts: 1,
	}

	if oldsubmission != nil {
		// TODO: need to be careful with given XP for resubmitting
		// disallow resubmitting if previous submition is successful
		// disallow resubmitting if item type is lesson
		cantResubmit := oldsubmission.Status == "pass"
		if cantResubmit {
			res := SubmitResponse{
				Submission: oldsubmission,
			}
			c.JSON(http.StatusConflict, res)
			return
		}
	}

	if *item.Type == "code" {
		// TODO: handle code runner queuing
		HandleCode(req.Data)
	} else if *item.Type == "lesson" {
		percent, err := HandleLesson(req.Data)
		if err != nil {
			h.logger.Error().Str("handler", "runner.submit").
				Err(err).
				Msg("Couldn't handle the lesson data")
			appError.ErrorInvalidInput().AbortWithErrorJson(c)
			return
		}
		xpMultiplier := float64(percent) / 100.0
		if percent > 50 {
			newSubmission.Status = "pass"
			newSubmission.XPReward = int(float64(item.BaseXP) * xpMultiplier)
		} else {
			newSubmission.Status = "fail"
			newSubmission.XPReward = 0
		}
	} else {
		h.logger.Error().
			Str("id", itemId).
			Msg("This should not happen: An item with an invalid type. WARN Content Developers to Fix it")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// Start transaction
	tx, err := h.db.Begin(ctx)
	if err != nil {
		h.logger.Error().
			Err(err).
			Msg("Failed to begin transaction for user submission")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}
	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	_, aerr = upsertSubmission(ctx, tx, newSubmission)
	if aerr != nil {
		aerr.Log(h.logger.Error().Str("handler", "runner.submit"), true).
			Any("submission", newSubmission).
			Msg("Failed to upsert submission")
		aerr.AbortWithErrorJson(c)
		return
	}

	if newSubmission.Status == "pass" && newSubmission.XPReward > 0 {
		xpEvent := &xpRepo.XPEvent{
			UserID:     userID.(uuid.UUID),
			XPAmount:   newSubmission.XPReward,
			SourceType: fmt.Sprintf("%s/%s", "item/", *item.Type),
			SourceID:   item.ID,
		}
		err := xpRepo.AddXP(ctx, tx, *xpEvent)
		if err != nil {
			h.logger.Error().
				Err(err).
				Str("error", err.Error()).
				Msg("Failed to add xp for user")
			appError.ErrorInternal().AbortWithErrorJson(c)
			return
		}
	}

	// 5. Commit transaction
	if err := tx.Commit(ctx); err != nil {
		h.logger.Error().
			Err(err).
			Msg("Couldn't commit transaction")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// Return response
	res := SubmitResponse{
		Submission: newSubmission,
	}
	c.JSON(http.StatusOK, res)
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

// FIX:
func HandleCode(data DataType) {
}

func HandleLesson(data DataType) (int, error) {
	encoded, ok := data["_$"].(string)
	if !ok {
		return 0, appError.ErrorInvalidInput()
	}
	/*
	 * Lore Time:
	 *
	 * This is a coding teaching website and since hacking can be considered coding too
	 * I wanna make it possible to cheat in the lesson quizes and get full score without answering,
	 * but we should not make it that easy, therefore the obfuscation
	 */
	salt := 69
	decodedBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return 0, err
	}

	raw, err := strconv.Atoi(string(decodedBytes))
	if err != nil {
		return 0, err
	}

	percent := raw / salt

	// Clamp to 0–100 range: above 100 or below 0 is dangerous and not allowed :)
	if percent > 100 {
		percent = 100
	} else if percent < 0 {
		percent = 0
	}

	return percent, nil
}
