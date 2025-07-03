package s3internal

import (
	"context"
	"fmt"
	"log"
	"net/url"

	"github.com/arabkood/backend/config"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	smithyendpoints "github.com/aws/smithy-go/endpoints"
)

type customEndpointResolver struct {
	baseEndpoint string
}

// ResolveEndpoint now dynamically constructs the URL
func (r *customEndpointResolver) ResolveEndpoint(ctx context.Context, params s3.EndpointParameters) (
	smithyendpoints.Endpoint, error,
) {
	if params.Bucket != nil && *params.Bucket != "" {
		// If a bucket is specified, build the virtual-hosted URL.
		// e.g., "https://my-bucket.fsn1.your-objectstorage.com"
		virtualHostedURL := fmt.Sprintf("https://%s.%s", *params.Bucket, r.baseEndpoint)

		u, err := url.Parse(virtualHostedURL)
		if err != nil {
			return smithyendpoints.Endpoint{}, fmt.Errorf("failed to parse virtual-hosted URL: %w", err)
		}

		return smithyendpoints.Endpoint{URI: *u}, nil
	}

	// If no bucket is specified (e.g., for ListBuckets operation), return the base endpoint.
	// e.g., "https://fsn1.your-objectstorage.com"
	baseURL := "https://" + r.baseEndpoint
	u, err := url.Parse(baseURL)
	if err != nil {
		return smithyendpoints.Endpoint{}, fmt.Errorf("failed to parse base URL: %w", err)
	}

	return smithyendpoints.Endpoint{URI: *u}, nil
}

// NewS3Client creates a new S3 client configured for either AWS or a compatible provider.
func NewS3Client(cfg config.S3Config) *s3.Client {
	creds := credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")

	// // **** ADD LOGGING HERE ****
	// logMode := aws.LogRequest | aws.LogResponse | aws.LogRequestWithBody | aws.LogResponseWithBody
	// // **************************

	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithCredentialsProvider(creds),
		awsconfig.WithRegion(cfg.Region),
		// awsconfig.WithClientLogMode(logMode),
	)
	if err != nil {
		log.Fatalf("failed to load base S3 config: %v", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.EndpointResolverV2 = &customEndpointResolver{baseEndpoint: cfg.Endpoint}
		o.UsePathStyle = cfg.UsePathStyle
	})

	return client
}
