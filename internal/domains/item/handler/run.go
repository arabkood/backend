package handler

import (
	"context"
	"maps"
	"net/http"
	"time"

	itemInterface "github.com/arabkood/backend/internal/domains/item/interfaces/item"
	exerciseRepo "github.com/arabkood/backend/internal/domains/item/repo/exercise"
	repoItem "github.com/arabkood/backend/internal/domains/item/repo/item"
	appError "github.com/arabkood/backend/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

type RunRequest struct {
	Data   DataType `form:"data" binding:"required" json:"data"`
	Inputs string   `json:"inputs,omitempty"`
}

type RunResponse struct {
	TaskID  string `json:"task_id"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

const (
	TypeCodeRun = "code:run"
)

// Run handles code execution without creating a submission record
func (h *ItemHandler) Run(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// 1. Authentication & Authorization
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
	var req RunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		appError.ErrorInvalidInput().WithError(err).AbortWithErrorJson(c)
		return
	}

	// 3. Fetch Item and Validate Type
	itemRepo := repoItem.NewItemRepository(h.DB)
	item, aerr := itemRepo.GetItemById(ctx, itemId)
	if aerr != nil {
		aerr.Log(h.Logger.Error().Str("handler", "Run"), true).Msg("Failed to get item by ID")
		aerr.AbortWithErrorJson(c)
		return
	}

	// 4. Ensure item is a code exercise
	if item.Type == nil || *item.Type != "code" {
		appError.ErrorInvalidInput().WithMessage("Run endpoint only supports code exercises").AbortWithErrorJson(c)
		return
	}

	// 5. Process Code Execution (without submission)
	taskInfo, processingErr := h.processCodeRun(ctx, item, userID, req.Data, req.Inputs)
	if processingErr != nil {
		processingErr.Log(h.Logger.Error().Str("handler", "Run"), true).Msg("Failed to process code run")
		processingErr.AbortWithErrorJson(c)
		return
	}

	// 6. Return Response
	response := RunResponse{
		TaskID:  taskInfo.ID,
		Status:  "queued",
		Message: "Code execution queued successfully",
	}
	c.JSON(http.StatusOK, response)
}

// processCodeRun handles code execution without creating submission records
func (h *ItemHandler) processCodeRun(ctx context.Context, item *itemInterface.Item, userID uuid.UUID, data DataType, inputs string) (*asynq.TaskInfo, *appError.Error) {
	// 1. Generate a unique task ID for tracking
	taskID, err := uuid.NewRandom()
	if err != nil {
		return nil, appError.ErrorInternal().WithError(err).WithMessage("Failed to generate task ID")
	}

	// 2. Validate and Extract User-Submitted Files
	userFiles, err := validateUserFiles(data)
	if err != nil {
		return nil, appError.ErrorInvalidInput().WithError(err).WithMessage(err.Error())
	}

	// 3. Fetch Exercise Base Files
	exerciseData, err := exerciseRepo.GetCode(ctx, h.S3Client, h.Config.S3.PvBucketName, *item.S3Path)
	if err != nil {
		return nil, appError.ErrorInternal().WithError(err).WithMessage("Failed to get code for exercise")
	}

	// 4. Combine Files for the Final Payload
	finalFiles := make(map[string]string)
	maps.Copy(finalFiles, exerciseData.Files)
	userFiles = cleanUserFiles(exerciseData.Config.Files, userFiles)
	maps.Copy(finalFiles, userFiles)

	// 5. Determine Docker Image and Invocation Arguments
	if exerciseData.Config.Image == "" {
		return nil, appError.ErrorInternal().WithMessage("Missing Docker image configuration")
	}

	// 6. Construct the Run-Only Payload

	args := []string{
		"solution",
		"/app",
		"/tmp",
		"run-only",
		inputs,
	}

	payload := &CodeExecutionPayload{
		TaskID:         taskID.String(),
		UserID:         userID.String(),
		Image:          exerciseData.Config.Image,
		InvocationArgs: args,
		Files:          finalFiles,
		Metadata:       map[string]string{"exercise_id": item.ID.String(), "run_type": "test_only"},
		IsRunOnly:      true,
	}

	// 7. Enqueue the Run-Only Task
	taskInfo, err := enqueueCodeExecutionTask(h, "default", payload)
	if err != nil {
		return nil, appError.ErrorInternal().WithError(err).WithMessage("Failed to queue code execution")
	}

	h.Logger.Info().
		Str("task_id", taskInfo.ID).
		Str("user_id", userID.String()).
		Str("item_id", item.ID.String()).
		Msg("Successfully queued code run task")

	return taskInfo, nil
}
