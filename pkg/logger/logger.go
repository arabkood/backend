package logger

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	appConfig "github.com/arabkood/backend/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	clt "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/rs/zerolog"
)

type Logger struct {
	ZL            *zerolog.Logger
	cwClient      *cloudwatchlogs.Client
	logGroupName  string
	logStreamName string
	sequenceToken *string
}

type cloudWatchWriter struct {
	logger *Logger
}

func (w *cloudWatchWriter) Write(p []byte) (n int, err error) {
	if w.logger.cwClient == nil {
		return len(p), nil // Skip if not configured for CloudWatch
	}

	logEvent := &cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  &w.logger.logGroupName,
		LogStreamName: &w.logger.logStreamName,
		LogEvents: []clt.InputLogEvent{
			{
				Message:   aws.String(string(p)),
				Timestamp: aws.Int64(time.Now().UnixNano() / int64(time.Millisecond)),
			},
		},
	}

	if w.logger.sequenceToken != nil {
		logEvent.SequenceToken = w.logger.sequenceToken
	}

	output, err := w.logger.cwClient.PutLogEvents(context.Background(), logEvent)
	if err != nil {
		// Log error to stdout but don't fail
		os.Stdout.Write([]byte("Failed to send logs to CloudWatch: " + err.Error() + "\n"))
		return len(p), nil
	}

	w.logger.sequenceToken = output.NextSequenceToken
	return len(p), nil
}

func NewLogger(cfg *appConfig.Config) (*Logger, error) {
	logger := &Logger{}

	if cfg.App.Environment == "local" {
		// Local development logging
		output := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05",
			NoColor:    false,
		}
		zl := zerolog.New(output).
			Level(getLogLevel(cfg)).
			With().
			Timestamp().
			Str("service", cfg.App.Name).
			Str("env", cfg.App.Environment).
			Logger()
		logger.ZL = &zl
		return logger, nil
	}

	// Production CloudWatch setup
	awsCfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(cfg.Aws.Region))
	if err != nil {
		return nil, err
	}

	logger.cwClient = cloudwatchlogs.NewFromConfig(awsCfg)
	logger.logGroupName = "/aws/apps/" + cfg.App.Name
	logger.logStreamName = time.Now().Format("2006/01/02") + "/" + cfg.App.Environment

	// Ensure log group exists
	if err := logger.createLogGroupIfNotExists(); err != nil {
		return nil, err
	}

	// Create log stream
	if err := logger.createLogStream(); err != nil {
		return nil, err
	}

	// Setup multi-writer for both local and CloudWatch output
	cwWriter := &cloudWatchWriter{logger: logger}
	multiWriter := zerolog.MultiLevelWriter(os.Stdout, cwWriter)

	zl := zerolog.New(multiWriter).
		Level(getLogLevel(cfg)).
		With().
		Timestamp().
		Str("service", cfg.App.Name).
		Str("env", cfg.App.Environment).
		Logger()
	logger.ZL = &zl

	return logger, nil
}

func (l *Logger) createLogGroupIfNotExists() error {
	_, err := l.cwClient.CreateLogGroup(context.Background(), &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: &l.logGroupName,
	})
	// Ignore if log group already exists
	if err != nil {
		var aerr *clt.ResourceAlreadyExistsException
		if ok := errors.As(err, &aerr); !ok {
			return err
		}
	}

	// Set retention policy for the log group
	_, err = l.cwClient.PutRetentionPolicy(context.Background(), &cloudwatchlogs.PutRetentionPolicyInput{
		LogGroupName:    &l.logGroupName,
		RetentionInDays: aws.Int32(1),
	})
	if err != nil {
		return fmt.Errorf("failed to set log group retention policy: %w", err)
	}

	return nil
}

func (l *Logger) createLogStream() error {
	_, err := l.cwClient.CreateLogStream(context.Background(), &cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  &l.logGroupName,
		LogStreamName: &l.logStreamName,
	})
	// Ignore if stream already exists
	if err != nil {
		var aerr *clt.ResourceAlreadyExistsException
		if ok := errors.As(err, &aerr); !ok {
			return err
		}
	}
	return nil
}

// Debug returns a debug event logger
func (l *Logger) Debug() *zerolog.Event {
	return l.ZL.Debug()
}

// Info returns an info event logger
func (l *Logger) Info() *zerolog.Event {
	return l.ZL.Info()
}

// Warn returns a warn event logger
func (l *Logger) Warn() *zerolog.Event {
	return l.ZL.Warn()
}

// Error returns an error event logger with caller info
func (l *Logger) Error() *zerolog.Event {
	return l.ZL.Error().Caller(1)
}

// Fatal returns a fatal event logger with caller info and stack trace
func (l *Logger) Fatal() *zerolog.Event {
	return l.ZL.Fatal().Caller(1).Stack()
}

func (l *Logger) HTTPRequest(r *http.Request, status int, latency time.Duration, ip string) {
	if r.URL.Path == "/health" || r.URL.Path == "/ready" {
		return
	}

	requestID := r.Header.Get("X-Request-ID")
	event := l.createBaseEvent(r, status, latency, ip, requestID)

	switch {
	case status >= 500:
		event(l.Error()).Msg("Server error")
	case status >= 400:
		event(l.Warn()).Msg("Client error")
	default:
		event(l.Info()).Msg("Request completed")
	}
}

func (l *Logger) createBaseEvent(r *http.Request, status int, latency time.Duration, ip, requestID string) func(*zerolog.Event) *zerolog.Event {
	return func(e *zerolog.Event) *zerolog.Event {
		e.Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", status).
			Str("latency", latency.String()).
			Str("ip", ip).
			Str("user_agent", r.UserAgent())

		if requestID != "" {
			e.Str("request_id", requestID)
		}
		if len(r.URL.RawQuery) > 0 {
			e.Str("query", r.URL.RawQuery)
		}

		return e
	}
}

func getLogLevel(cfg *appConfig.Config) zerolog.Level {
	if cfg.App.Debug {
		return zerolog.DebugLevel
	}
	return zerolog.InfoLevel
}
