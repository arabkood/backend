package exercise

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"

	"github.com/arabkood/backend/internal/domains/item/interfaces/exercise"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func GetCode(ctx context.Context, client *s3.Client, bucketName string, source string) (*exercise.Code, error) {
	code := &exercise.Code{}

	// 1. Fetch and parse config.json
	cfgContent, err := fetchS3File(ctx, client, bucketName, path.Join(source, "config.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch config.json: %w", err)
	}
	if err := json.Unmarshal([]byte(cfgContent), &code.Config); err != nil {
		return nil, fmt.Errorf("invalid config.json: %w", err)
	}

	// 2. Fetch and unzip files.bundle.zip
	filesZip, err := fetchS3Bytes(ctx, client, bucketName, path.Join(source, "files.bundle.zip"))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch files.bundle.zip: %w", err)
	}
	filesMap, err := unzipToMap(filesZip)
	if err != nil {
		return nil, fmt.Errorf("bad zip files.bundle.zip: %w", err)
	}
	code.Files = filesMap

	// 3. Fetch and unzip docs.bundle.zip
	docsZip, err := fetchS3Bytes(ctx, client, bucketName, path.Join(source, "docs.bundle.zip"))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch docs.bundle.zip: %w", err)
	}
	docsMap, err := unzipToMap(docsZip)
	if err != nil {
		return nil, fmt.Errorf("bad zip docs.bundle.zip: %w", err)
	}
	code.Docs = docsMap

	return code, nil
}

func fetchS3File(ctx context.Context, client *s3.Client, bucketName, objectKey string) (string, error) {
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read S3 response body: %w", err)
	}

	return string(body), nil
}

func fetchS3Bytes(ctx context.Context, client *s3.Client, bucketName, objectKey string) ([]byte, error) {
	resp, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch binary object: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read binary response body: %w", err)
	}
	return body, nil
}

func unzipToMap(zipData []byte) (map[string]string, error) {
	r, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("unable to read zip data: %w", err)
	}

	files := make(map[string]string)
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open zip file entry: %w", err)
		}
		defer rc.Close()

		content, err := io.ReadAll(rc)
		if err != nil {
			return nil, fmt.Errorf("failed to read zip content: %w", err)
		}
		files[f.Name] = string(content)
	}

	return files, nil
}
