package appError

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type ErrorType string

// Error represents a domain-specific error
type Error struct {
	Type     ErrorType
	Message  *string
	Err      error
	Meta     map[string]interface{}
	HttpCode int // Optional HTTP status code
}

// Error implements the error interface
func (e *Error) Error() string {
	var parts []string
	if e.Message != nil {
		parts = append(parts, *e.Message)
	}
	if e.Err != nil {
		parts = append(parts, e.Err.Error())
	}
	if len(parts) == 0 {
		return string(e.Type)
	}
	return strings.Join(parts, ": ")
}

// Unwrap implements the errors.Unwrap interface
func (e *Error) Unwrap() error {
	return e.Err
}

// Is implements the errors.Is interface
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Type == t.Type
}

// Is implements the errors.Is interface
func (e *Error) IsInternal() bool {
	return e.Type == ErrInternal
}

// WithMessage sets the error message and returns the error for chaining
func (e *Error) WithMessage(message string) *Error {
	e.Message = &message
	return e
}

// WithMeta adds metadata to the error and returns the error for chaining
func (e *Error) WithMeta(key string, value interface{}) *Error {
	if e.Meta == nil {
		e.Meta = make(map[string]interface{})
	}
	e.Meta[key] = value
	return e
}

// WithError adds an underlying error and returns the error for chaining
func (e *Error) WithError(err error) *Error {
	e.Err = err
	return e
}

// WithHttpCode sets the HTTP status code and returns the error for chaining
func (e *Error) WithHttpCode(code int) *Error {
	e.HttpCode = code
	return e
}

// GetHttpCode returns the HTTP status code if set, or a default code based on the error type
func (e *Error) GetHttpCode() int {
	if e.HttpCode != 0 {
		return e.HttpCode
	}

	// Default HTTP codes based on error type
	switch e.Type {
	case ErrInternal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// Logs the error
func (e *Error) Log(logger *zerolog.Event, internalOnly bool) *zerolog.Event {
	code := e.GetHttpCode()
	if internalOnly {
		if code >= 500 {
			return logger.Err(e.Err)
		}
		return nil
	}
	return logger.Err(e.Err)
}

// Adds an error JSON response to Gin Context
func (e *Error) AbortWithErrorJson(c *gin.Context) {
	res := gin.H{
		"error": e.Type,
	}
	if e.Message != nil {
		res = gin.H{
			"error":   e.Type,
			"message": e.Message,
		}
	}
	c.AbortWithStatusJSON(e.GetHttpCode(), res)
}

// New creates a new AppError with the given type
func NewError(errType ErrorType) *Error {
	return &Error{
		Type: errType,
		Meta: make(map[string]interface{}),
	}
}
