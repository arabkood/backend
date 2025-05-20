package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/arabkood/backend/internal/domains/item/interfaces/exercise"
	exerciseRepo "github.com/arabkood/backend/internal/domains/item/repo/exercise"
	repoItem "github.com/arabkood/backend/internal/domains/item/repo/item"
	"github.com/arabkood/backend/internal/domains/item/runner"
	infraPostgres "github.com/arabkood/backend/internal/postgres"
	appError "github.com/arabkood/backend/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AttemptRequest struct {
	UserFiles map[string]string `form:"files" binding:"required" json:"files"`
}

type AttemptResponse struct {
	SubmissionID string `json:"submissionId"`
}

func (h *ItemHandler) Attempt(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Get user ID from auth context
	userID, exists := c.Get("userID")
	if !exists {
		appError.ErrorUnauthorized().AbortWithErrorJson(c)
		return
	}

	id := c.Param("id")
	if id == "" {
		appError.ErrorInvalidInput().AbortWithErrorJson(c)
		return
	}

	// Bind request body
	var req AttemptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Debug().Err(err).Send()
		appError.ErrorInvalidInput().AbortWithErrorJson(c)
		return
	}

	userFilesBytes := new(bytes.Buffer)
	err := json.NewEncoder(userFilesBytes).Encode(req.UserFiles)
	// 30KB limit
	if userFilesBytes.Len() > 30*1024 || err != nil {
		h.logger.Error().Str("handler", "runner.attempt").
			Int("approximate size in KB", userFilesBytes.Len()/1024).
			Err(err).
			Str("item", id).
			Msg("User submission is too big")
		appError.ErrorInvalidInput().WithMessage("Files submitted are too big, limit is 30KB").AbortWithErrorJson(c)
		return
	}

	// Create repository instance
	itemRepo := repoItem.NewItemRepository(h.db)

	// Get item
	item, aerr := itemRepo.GetItemById(ctx, id)
	if aerr != nil {
		aerr.Log(h.logger.Error().Str("handler", "runner.attempt"), true).
			Str("id", id).
			Msg("Failed to get item by ID")
		aerr.AbortWithErrorJson(c)
		return
	}

	// Create or update attempt row in DB
	newAttempt := &exercise.Attempt{
		ItemID:    item.ID,
		UserID:    userID.(uuid.UUID),
		UserFiles: userFilesBytes.Bytes(),
	}
	attemptID, aerr := upsertAttempt(ctx, h.db, newAttempt)
	if aerr != nil {
		aerr.Log(h.logger.Error().Str("handler", "runner.attempt"), true).
			Str("id", id).
			Msg("Failed to upsert submission")
		aerr.AbortWithErrorJson(c)
		return
	}

	// 3. Save User Files to EFS
	safeJoin := func(basePath, userPath string) (string, error) {
		basePath = filepath.Clean(basePath)
		fullPath := filepath.Join(basePath, userPath)
		fullPath = filepath.Clean(fullPath)

		// Check if the path is within the base directory
		rel, err := filepath.Rel(basePath, fullPath)
		if err != nil || strings.HasPrefix(rel, "..") {
			return "", fmt.Errorf("invalid path")
		}

		return fullPath, nil
	}

	submissionPath, err := safeJoin(h.config.Other.SubmissionsDirectory, attemptID)
	if err != nil {
		h.logger.Error().Str("handler", "runner.attempt").
			Err(err).
			Msg("Wrong path when writing submission to EFS")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// Remove directory if it exists
	if _, err := os.Stat(submissionPath); err == nil {
		if err := os.RemoveAll(submissionPath); err != nil {
			h.logger.Error().Str("handler", "runner.attempt").
				Str("submissionPath", submissionPath).
				Err(err).
				Msg("Failed to remove old submission directory")
			appError.ErrorInternal().AbortWithErrorJson(c)
			return
		}
	}
	// Create a new directory
	if err := os.Mkdir(submissionPath, 0777); err != nil {
		h.logger.Error().Str("handler", "runner.attempt").
			Str("submissionPath", submissionPath).
			Err(err).
			Msg("Failed to create submission directory")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// Get exercise files
	code := &exercise.Code{}
	if item.Type == "exercise" {
		code, err = exerciseRepo.GetCode(c, h.s3Client, h.config.Aws.TopicsBucketName, item.S3Path)
		if err != nil {
			aerr.Log(h.logger.Error().Err(err).Str("handler", "exerciseRepo.GetCode"), true).
				Msg("Failed to get code for exercice")
			appError.ErrorInternal().AbortWithErrorJson(c)
			return
		}
	}

	errors := make(map[string]string)

	for path, content := range code.Files {
		// Validate and sanitize the file path
		fullPath, err := safeJoin(submissionPath, path)
		if err != nil {
			errors[path] = "Invalid path"
			continue
		}

		// Create directory if it doesn't exist
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0766); err != nil {
			errors[path] = "Failed to create directory"
			continue
		}

		// Write file content
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			errors[path] = "Failed to write file"
		}
	}

	for path, content := range req.UserFiles {
		// Validate and sanitize the file path
		fullPath, err := safeJoin(submissionPath, path)
		if err != nil {
			errors[path] = "Invalid path"
			continue
		}

		// Create directory if it doesn't exist
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0766); err != nil {
			errors[path] = "Failed to create directory"
			continue
		}

		// Write/Replace file content
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			errors[path] = "Failed to write file"
		}
	}

	if len(errors) > 0 {
		h.logger.Error().Str("handler", "runner.attempt").
			Any("errors", errors).
			Msg("Failed to create user files")
		appError.ErrorInvalidInput().AbortWithErrorJson(c)
		return
	}

	// 4. Queue the job to SQS
	job := &runner.SubmissionJob{
		ID:     attemptID,
		Type:   "test",
		Runner: code.Config.Image,
		InvocationArgs: []string{
			"/mnt/kood-iteration",
			"/mnt/kood-iteration",
		},
	}

	// Send the job
	err = runner.SendJob(context.Background(), h.sqsClient, h.config.Aws.SubmissionQueueURL, job)
	if err != nil {
		h.logger.Error().Str("handler", "runner.attempt").
			Err(err).
			Msg("Something went wrong with sending submission job to sqs")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	// Return response
	res := AttemptResponse{
		SubmissionID: attemptID,
	}
	c.JSON(http.StatusOK, res)
}

func upsertAttempt(ctx context.Context, querier infraPostgres.Querier, attempt *exercise.Attempt) (string, *appError.Error) {
	query := `
         INSERT INTO users.code_attempt (
              user_id, item_id,
              user_files, status, attempts
         ) VALUES (
             $1, $2,
             $3, 'wait', 0
         )
         ON CONFLICT (user_id, item_id) DO UPDATE
         SET user_files = $3, id = gen_random_uuid(), status = 'wait', results = default
         RETURNING id`
	var id string
	err := querier.QueryRow(ctx, query,
		attempt.UserID,
		attempt.ItemID,
		attempt.UserFiles,
	).Scan(&id)
	if err != nil {
		return "", appError.ErrorInternal().WithError(err)
	}
	return id, nil
}
