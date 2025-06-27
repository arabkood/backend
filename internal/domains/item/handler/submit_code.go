package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/arabkood/backend/internal/domains/item/interfaces/exercise"
	itemInterface "github.com/arabkood/backend/internal/domains/item/interfaces/item"
	exerciseRepo "github.com/arabkood/backend/internal/domains/item/repo/exercise"
	"github.com/arabkood/backend/internal/domains/item/runner"
	appError "github.com/arabkood/backend/pkg/errors"
	"github.com/google/uuid"
)

const (
	MaxFileSize    = 50 * 1024      // 50KB per file
	MaxTotalSize   = 50 * 50 * 1024 // 50 * 50KB total
	MaxFileCount   = 50             // Maximum number of files
	MaxFilenameLen = 255            // Maximum filename length
)

func handleCode(c context.Context, h *ItemHandler, data DataType, item *itemInterface.Item, subID uuid.UUID) *appError.Error {
	subId := subID.String()
	if subId == "" {
		return appError.ErrorInternal()
	}

	userFiles, err := validateDataType(data)
	if err != nil {
		return appError.ErrorInvalidInput().WithError(err).WithMessage(err.Error())
	}

	// 2. Save User Files to EFS
	safeJoin := func(basePath, userPath string) (string, error) {
		// Clean and resolve absolute paths
		basePath, err := filepath.Abs(filepath.Clean(basePath))
		if err != nil {
			return "", err
		}

		fullPath := filepath.Join(basePath, filepath.Clean(userPath))
		fullPath, err = filepath.Abs(fullPath)
		if err != nil {
			return "", err
		}

		// Resolve symlinks
		if _, err := os.Lstat(fullPath); err == nil {
			resolvedPath, err := filepath.EvalSymlinks(fullPath)
			if err != nil {
				return "", err
			}
			fullPath = resolvedPath
		}

		// Check if still within base directory
		rel, err := filepath.Rel(basePath, fullPath)
		if err != nil {
			return "", err
		}
		if strings.HasPrefix(rel, "..") || strings.Contains(rel, string(filepath.Separator)+"..") {
			return "", fmt.Errorf("path has '..'")
		}

		return fullPath, nil
	}

	submissionPath, err := safeJoin(h.config.Other.SubmissionsDirectory, subId)
	if err != nil {
		return appError.ErrorInternal().WithError(err).WithMessage("Wrong path when writing submission to EFS")
	}

	// Remove directory if it exists
	if _, err := os.Stat(submissionPath); err == nil {
		if err := os.RemoveAll(submissionPath); err != nil {
			return appError.ErrorInternal().WithMeta("submissionPath", submissionPath).WithError(err).WithMessage("Failed to remove old submission directory")
		}
	}

	// Create a new directory
	if err := os.Mkdir(submissionPath, 0750); err != nil {
		return appError.ErrorInternal().WithMeta("submissionPath", submissionPath).WithError(err).WithMessage("Failed to create submission directory")
	}

	// Get exercise files
	code := &exercise.Code{}
	code, err = exerciseRepo.GetCode(c, h.s3Client, h.config.Aws.TopicsBucketName, *item.S3Path)
	if err != nil {
		return appError.ErrorInternal().WithError(err).WithMessage("Failed to get code for exercice")
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
		if err := os.MkdirAll(dir, 0755); err != nil {
			errors[path] = "Failed to create directory"
			continue
		}

		// Write file content
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			errors[path] = "Failed to write file"
		}
	}

	for path, content := range userFiles {
		// Validate and sanitize the file path
		fullPath, err := safeJoin(submissionPath, path)
		if err != nil {
			errors[path] = "Invalid path"
			continue
		}

		// Create directory if it doesn't exist
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			errors[path] = "Failed to create directory"
			continue
		}

		// Write/Replace file content
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			errors[path] = "Failed to write file"
		}
	}

	if len(errors) > 0 {
		return appError.ErrorInvalidInput().WithMeta("errors", errors).WithMessage("Failed to create user files")
	}

	img := "hello-world"
	if code.Config.Image != "" {
		img = code.Config.Image
	}

	// 4. Queue the job to SQS
	job := &runner.SubmissionJob{
		ID:     subId,
		Type:   "test",
		Runner: img,
		InvocationArgs: []string{
			"/mnt/kood-iteration",
			"/mnt/kood-iteration",
		},
	}

	// Send the job
	err = runner.SendJob(context.Background(), h.sqsClient, h.config.Aws.SubmissionQueueURL, job)
	if err != nil {
		return appError.ErrorInternal().WithError(err).WithMessage("Something went wrong with sending submission job to sqs")
	}

	return nil
}

func validateDataType(data DataType) (map[string]string, error) {
	// Check if files key exists
	filesRaw, exists := data["files"]
	if !exists {
		return nil, fmt.Errorf("missing 'files' field")
	}

	// Assert that files is a map[string]any first
	filesAny, ok := filesRaw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("'files' must be a map[string]any, got %T", filesRaw)
	}

	// Convert and validate each file entry
	files := make(map[string]string)
	var totalSize int64

	if len(filesAny) > MaxFileCount {
		return nil, fmt.Errorf("too many files: %d (max %d)", len(filesAny), MaxFileCount)
	}

	for filename, contentRaw := range filesAny {
		// Validate filename
		if filename == "" {
			return nil, fmt.Errorf("empty filename not allowed")
		}
		if !utf8.ValidString(filename) {
			return nil, fmt.Errorf("invalid UTF-8 in filename: %s", filename)
		}

		if len(filename) > MaxFilenameLen {
			return nil, fmt.Errorf("filename too long: %d chars (max %d)", len(filename), MaxFilenameLen)
		}

		// Assert content is string
		content, ok := contentRaw.(string)
		if !ok {
			return nil, fmt.Errorf("file content for '%s' must be string, got %T", filename, contentRaw)
		}

		// Validate content is valid UTF-8
		if !utf8.ValidString(content) {
			return nil, fmt.Errorf("file content for '%s' contains invalid UTF-8", filename)
		}

		// Check individual file size
		fileSize := int64(len(content))
		if fileSize > MaxFileSize {
			return nil, fmt.Errorf("file '%s' too large: %d bytes (max %d)", filename, fileSize, MaxFileSize)
		}

		totalSize += fileSize
		files[filename] = content
	}

	// Check total size
	if totalSize > MaxTotalSize {
		return nil, fmt.Errorf("total files size too large: %d bytes (max %d)", totalSize, MaxTotalSize)
	}

	return files, nil
}
