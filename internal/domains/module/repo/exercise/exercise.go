package exercise

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"

	"github.com/arabkood/backend/internal/domains/module/interfaces/exercise"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func GetExercise(ctx context.Context, client *s3.Client, bucketName string, source string) (*exercise.Exercise, error) {
	exr := &exercise.Exercise{
		Image:        "",
		Instructions: "",
		Hints:        "",
		Files:        nil,
	}

	exCfgBody, err := fetchFile(ctx, client, bucketName, path.Join(source, "+config.json"))
	if err != nil {
		return exr, err
	}
	var exCfg *exercise.ExerciseConfig
	if err := json.Unmarshal([]byte(exCfgBody), &exCfg); err != nil {
		return exr, fmt.Errorf("failed to parse JSON: %w", err)
	}

	exr.Image = exCfg.Image

	// fetch solution files
	for _, f := range exCfg.Files.Solution {
		content, err := fetchFile(ctx, client, bucketName, path.Join(source, f.Path))
		if err != nil {
			return exr, err
		}
		exr.Files = append(exr.Files, exercise.File{
			Idx:      f.Idx,
			Language: f.Language,
			Path:     f.Path,
			Content:  &content,
		})
	}

	// fetch text files
	for _, f := range *exCfg.Files.Test {
		content, err := fetchFile(ctx, client, bucketName, path.Join(source, f.Path))
		if err != nil {
			return exr, err
		}
		exr.TestFiles = append(exr.TestFiles, exercise.File{
			Idx:      f.Idx,
			Language: f.Language,
			Path:     f.Path,
			Content:  &content,
		})
	}

	// fetch instruction file
	content, err := fetchFile(ctx, client, bucketName, path.Join(source, ".docs/instructions.md"))
	if err != nil {
		var notFoundErr *types.NoSuchKey
		if !errors.As(err, &notFoundErr) {
			return exr, err
		}
	}
	exr.Instructions = content

	// fetch hints file
	content, err = fetchFile(ctx, client, bucketName, path.Join(source, ".docs/hints.md"))
	if err != nil {
		var notFoundErr *types.NoSuchKey
		if !errors.As(err, &notFoundErr) {
			return exr, err
		}
	}
	exr.Hints = content

	return exr, nil
}

func fetchFile(ctx context.Context, client *s3.Client, bucketName, objectKey string) (string, error) {
	resp, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		var notFoundErr *types.NoSuchKey
		if errors.As(err, &notFoundErr) {
			return "", notFoundErr
		}
		return "", fmt.Errorf("failed to fetch object: %w", err)
	}
	defer resp.Body.Close()

	// Read the file content
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read S3 response body: %w", err)
	}

	return string(body), nil
}
