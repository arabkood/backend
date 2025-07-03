package logger

import (
	"net/http"
	"os"
	"time"

	appConfig "github.com/arabkood/backend/config"
	"github.com/rs/zerolog"
)

type Logger struct {
	ZL            *zerolog.Logger
	logGroupName  string
	logStreamName string
	sequenceToken *string
}

func NewLogger(cfg *appConfig.Config) (*Logger, error) {
	logger := &Logger{}

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
		Logger()
	logger.ZL = &zl
	return logger, nil
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
