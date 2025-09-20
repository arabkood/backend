package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	appError "github.com/arabkood/backend/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// RunStatusResponse represents the response structure for run status
type RunStatusResponse struct {
	TaskID      string                 `json:"task_id"`
	Status      string                 `json:"status"`
	Results     map[string]interface{} `json:"results,omitempty"`
	Error       string                 `json:"error,omitempty"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	TTL         int64                  `json:"ttl,omitempty"`
}

// RunResult represents the structure stored in Valkey
type RunResult struct {
	TaskID      string                 `json:"task_id"`
	UserID      string                 `json:"user_id"`
	ItemID      string                 `json:"item_id"`
	Status      string                 `json:"status"` // pending, running, success, error, timeout
	Results     map[string]interface{} `json:"results,omitempty"`
	Error       string                 `json:"error,omitempty"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
}

const (
	RunResultKeyPrefix = "run_result:"
)

// GetRunStatus retrieves the status of a run-only task from Valkey
func (h *ItemHandler) GetRunStatus(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	taskID := c.Param("taskId")
	if taskID == "" {
		appError.ErrorInvalidInput().WithMessage("Task ID is required").AbortWithErrorJson(c)
		return
	}

	// Authentication check
	userIDAny, exists := c.Get("userID")
	if !exists {
		appError.ErrorUnauthorized().AbortWithErrorJson(c)
		return
	}
	userID := userIDAny.(uuid.UUID)
	userIDStr := userID.String()

	// Get result from Valkey
	key := RunResultKeyPrefix + taskID
	resultJSON, err := h.ValkeyClient.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			response := RunStatusResponse{
				TaskID: taskID,
				Status: "pending",
			}
			c.JSON(http.StatusOK, response)
			return
		}
		h.Logger.Error().
			Err(err).
			Str("task_id", taskID).
			Str("user_id", userIDStr).
			Msg("Error retrieving run result from Valkey")
		appError.ErrorInternal().WithMessage("Failed to retrieve run status").AbortWithErrorJson(c)
		return
	}

	// Parse the result
	var result RunResult
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		h.Logger.Error().
			Err(err).
			Str("task_id", taskID).
			Str("valkey_data", resultJSON).
			Msg("Failed to unmarshal run result from Valkey")
		appError.ErrorInternal().WithMessage("Failed to parse run status").AbortWithErrorJson(c)
		return
	}

	// Verify user owns this run result
	if result.UserID != userIDStr {
		h.Logger.Warn().
			Str("task_id", taskID).
			Str("requesting_user", userIDStr).
			Str("result_owner", result.UserID).
			Msg("User attempted to access run result they don't own")
		appError.ErrorUnauthorized().WithMessage("Access denied").AbortWithErrorJson(c)
		return
	}

	// Get TTL information
	ttl, err := h.ValkeyClient.TTL(ctx, key).Result()
	if err != nil {
		h.Logger.Warn().Err(err).Str("task_id", taskID).Msg("Failed to get TTL for run result")
	}

	// Build response
	response := RunStatusResponse{
		TaskID:      result.TaskID,
		Status:      result.Status,
		Results:     result.Results,
		Error:       result.Error,
		StartedAt:   &result.StartedAt,
		CompletedAt: result.CompletedAt,
	}

	if ttl > 0 {
		response.TTL = int64(ttl.Seconds())
	}

	c.JSON(http.StatusOK, response)
}
