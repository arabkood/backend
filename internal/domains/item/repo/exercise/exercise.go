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

	basePath := path.Join("topics", source)

	// NOTE: config.json has been moved inside files/.meta/config.json
	// 1. Fetch and parse config.json
	// cfgContent, err := fetchS3File(ctx, client, bucketName, path.Join(basePath, "config.json"))
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to fetch config.json: %w", err)
	// } else {
	// 	if err := json.Unmarshal([]byte(cfgContent), &code.Config); err != nil {
	// 		return nil, fmt.Errorf("invalid config.json format: %w", err)
	// 	}
	// }

	// 2. Fetch and unzip files.bundle.zip
	filesZip, err := fetchS3Bytes(ctx, client, bucketName, path.Join(basePath, "files.bundle.zip"))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch files.bundle.zip: %w", err)
	}
	filesMap, err := unzipToMap(filesZip)
	if err != nil {
		return nil, fmt.Errorf("could not unzip files.bundle.zip: %w", err)
	}

	cfgContent, ok := filesMap[".meta/config.json"]
	if !ok {
		return nil, fmt.Errorf(".meta/config.json not found: %w", err)
	}
	if err := json.Unmarshal([]byte(cfgContent), &code.Config); err != nil {
		return nil, fmt.Errorf("invalid .meta/config.json format: %w", err)
	}

	// Remove all entries where the key starts with ".meta"
	// for name := range filesMap {
	// 	if strings.HasPrefix(name, ".meta") {
	// 		delete(filesMap, name)
	// 	}
	// }

	code.Files = filesMap

	// 3. Fetch and unzip docs.bundle.zip
	docsZip, err := fetchS3Bytes(ctx, client, bucketName, path.Join(basePath, "docs.bundle.zip"))
	if err != nil {
		// Documentation is optional. If not found, proceed with an empty map.
		var notFoundErr *types.NoSuchKey
		if errors.As(err, &notFoundErr) {
			code.Docs = make(map[string]string)
		} else {
			return nil, fmt.Errorf("failed to fetch docs.bundle.zip: %w", err)
		}
	} else {
		docsMap, err := unzipToMap(docsZip)
		if err != nil {
			return nil, fmt.Errorf("could not unzip docs.bundle.zip: %w", err)
		}
		code.Docs = docsMap
	}

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
		return "", fmt.Errorf("failed to fetch object '%s': %w", objectKey, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read S3 response body for '%s': %w", objectKey, err)
	}

	return string(body), nil
}

func fetchS3Bytes(ctx context.Context, client *s3.Client, bucketName, objectKey string) ([]byte, error) {
	resp, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		var notFoundErr *types.NoSuchKey
		if errors.As(err, &notFoundErr) {
			return nil, notFoundErr
		}
		return nil, fmt.Errorf("failed to fetch binary object '%s': %w", objectKey, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read binary response body for '%s': %w", objectKey, err)
	}
	return body, nil
}

// unzipToMap takes zip data as bytes and returns a map of filenames to their string content.
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
			return nil, fmt.Errorf("failed to open zip file entry '%s': %w", f.Name, err)
		}
		// Use a closure with defer to ensure each file's reader is closed correctly in the loop.
		func() {
			defer rc.Close()
			content, rerr := io.ReadAll(rc)
			if rerr != nil {
				// We can't recover from a read error mid-zip, so we store it and break.
				err = fmt.Errorf("failed to read zip content for '%s': %w", f.Name, err)
				return
			}
			files[f.Name] = string(content)
		}()
		if err != nil {
			return nil, err
		}
	}

	return files, nil
}
