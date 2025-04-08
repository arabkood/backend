package runner

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// SubmissionJob represents the code execution request
type SubmissionJob struct {
	ID             string            `json:"id"`
	Type           string            `json:"type"` // test, run
	Runner         string            `json:"runner"`
	InvocationArgs []string          `json:"invocation_args"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// Send sends a single job to SQS
func SendJob(ctx context.Context, sqsClient *sqs.Client, queueURL string, job *SubmissionJob) error {
	// Convert job to JSON
	jobBytes, err := json.Marshal(job)
	if err != nil {
		return err
	}

	// Create send message input
	input := &sqs.SendMessageInput{
		MessageBody: aws.String(string(jobBytes)),
		QueueUrl:    aws.String(queueURL),
		// Add message attributes for filtering if needed
		MessageAttributes: map[string]types.MessageAttributeValue{
			"runner": {
				DataType:    aws.String("String"),
				StringValue: aws.String(job.Runner),
			},
		},
	}

	// Send message to SQS
	_, err = sqsClient.SendMessage(ctx, input)
	return err
}
