package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"time"
	"unicode/utf8"

	"github.com/arabkood/backend/internal/domains/item/interfaces/exercise"
	itemInterface "github.com/arabkood/backend/internal/domains/item/interfaces/item"
	exerciseRepo "github.com/arabkood/backend/internal/domains/item/repo/exercise"
	appError "github.com/arabkood/backend/pkg/errors"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

const (
	MaxFileSize    = 50 * 1024      // 50KB per file
	MaxTotalSize   = 50 * 50 * 1024 // 50 * 50KB total
	MaxFileCount   = 50             // Maximum number of files
	MaxFilenameLen = 255            // Maximum filename length
)

const (
	TypeCodeExecution = "code:execute"
)

type CodeExecutionPayload struct {
	SubmissionID   string            `json:"submission_id"`
	UserID         string            `json:"user_id"`
	Image          string            `json:"image"`
	InvocationArgs []string          `json:"invocation_args"`
	Files          map[string]string `json:"files"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

func enqueueCodeExecutionTask(h *ItemHandler, queueName string, payloadStruct *CodeExecutionPayload) (*asynq.TaskInfo, error) {
	payload, err := json.Marshal(payloadStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task payload: %w", err)
	}
	task := asynq.NewTask(TypeCodeExecution, payload, asynq.TaskID(payloadStruct.SubmissionID))

	return h.AsynqClient.Enqueue(task,
		asynq.Queue(queueName),
		asynq.MaxRetry(0),
		asynq.Deadline(time.Now().Add(5*time.Minute)),
		asynq.Timeout(5*time.Minute),
		asynq.Retention(1*time.Hour))
}

func handleCodeSubmission(c context.Context, h *ItemHandler, data DataType, item *itemInterface.Item, userID uuid.UUID, subID uuid.UUID) (*asynq.TaskInfo, *appError.Error) {
	// 1. Validate Inputs (User and Submission IDs)
	subIdStr := subID.String()
	userIdStr := userID.String()
	if subIdStr == "" || userIdStr == "" {
		return nil, appError.ErrorInternal().WithMessage("Missing user or submission ID")
	}

	// 2. Validate and Extract User-Submitted Files
	userFiles, err := validateUserFiles(data)
	if err != nil {
		return nil, appError.ErrorInvalidInput().WithError(err).WithMessage(err.Error())
	}

	// 3. Fetch Exercise Base Files
	// We fetch the exercise files to add them to the payload.
	exerciseData, err := exerciseRepo.GetCode(c, h.S3Client, h.Config.S3.PvBucketName, *item.S3Path)
	if err != nil {
		return nil, appError.ErrorInternal().WithError(err).WithMessage("Failed to get code for exercise")
	}

	// 4. Combine Files for the Final Payload
	// The user's files overwrite the exercise's base files, which is the desired behavior.
	finalFiles := make(map[string]string)
	maps.Copy(finalFiles, exerciseData.Files)
	userFiles = cleanUserFiles(exerciseData.Config.Files, userFiles)
	maps.Copy(finalFiles, userFiles)

	// 5. Determine Docker Image and Invocation Arguments
	if exerciseData.Config.Image == "" {
		return nil, appError.ErrorInternal().WithError(err).WithMessage("Null image")
	}
	image := exerciseData.Config.Image
	invocationArgs := []string{
		"hello-world",
		"/app/",
		"/tmp/iteration/",
	}

	// 6. Construct the Final Payload
	payload := &CodeExecutionPayload{
		SubmissionID:   subIdStr,
		UserID:         userIdStr,
		Image:          image,
		InvocationArgs: invocationArgs,
		Files:          finalFiles,
		Metadata:       map[string]string{"exercise_id": item.ID.String()},
	}

	// 7. Enqueue the Task
	// For now, we hardcode the "default" queue. In future, we can add a premium queue here.
	taskInfo, err := enqueueCodeExecutionTask(h, "default", payload)
	if err != nil {
		log.Printf("ERROR: Failed to enqueue code execution task: %v", err)
		return nil, appError.ErrorInternal().WithError(err).WithMessage("Failed to queue submission")
	}

	log.Printf("Successfully enqueued task %s for submission %s", taskInfo.ID, subIdStr)
	return taskInfo, nil
}

func validateUserFiles(data DataType) (map[string]string, error) {
	filesRaw, exists := data["files"]
	if !exists {
		return nil, fmt.Errorf("missing 'files' field in request")
	}

	filesAny, ok := filesRaw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("'files' must be a map[string]any, got %T", filesRaw)
	}

	if len(filesAny) > MaxFileCount {
		return nil, fmt.Errorf("too many files: %d (max %d)", len(filesAny), MaxFileCount)
	}

	files := make(map[string]string)
	var totalSize int64
	for filename, contentRaw := range filesAny {
		if filename == "" || !utf8.ValidString(filename) || len(filename) > MaxFilenameLen {
			return nil, fmt.Errorf("invalid filename: %s", filename)
		}

		content, ok := contentRaw.(string)
		if !ok {
			return nil, fmt.Errorf("file content for '%s' must be a string", filename)
		}

		if !utf8.ValidString(content) {
			return nil, fmt.Errorf("file content for '%s' contains invalid UTF-8", filename)
		}

		fileSize := int64(len(content))
		if fileSize > MaxFileSize {
			return nil, fmt.Errorf("file '%s' is too large: %d bytes (max %d)", filename, fileSize, MaxFileSize)
		}

		totalSize += fileSize
		files[filename] = content
	}

	if totalSize > MaxTotalSize {
		return nil, fmt.Errorf("total files size is too large: %d bytes (max %d)", totalSize, MaxTotalSize)
	}

	return files, nil
}

func cleanUserFiles(configFiles []exercise.CodeFileConfig, userFiles map[string]string) map[string]string {
	newFiles := map[string]string{}
	for _, f := range configFiles {
		if f.Ro != nil && *f.Ro {
			continue
		}
		val, ok := userFiles[f.Path]
		if ok {
			newFiles[f.Path] = val
		}
	}
	return newFiles
}
