package appError

// Errors - Domain-specific error definitions
const (
	// Authentication/Authorization errors
	ErrBadCredentials ErrorType = "BAD_CREDENTIALS"
	ErrAccountLocked  ErrorType = "ACCOUNT_LOCKED"

	// User-related validation errors
	ErrUsernameConflict     ErrorType = "USERNAME_CONFLICT"
	ErrEmailConflict        ErrorType = "EMAIL_CONFLICT"
	ErrInvalidEmail         ErrorType = "INVALID_EMAIL"
	ErrInvalidUsername      ErrorType = "INVALID_USERNAME"
	ErrInvalidPassword      ErrorType = "INVALID_PASSWORD"
	ErrUserNotFound         ErrorType = "USER_NOT_FOUND"
	ErrPrivateUser          ErrorType = "PRIVATE_USER"
	ErrEmailAlreadyVerified ErrorType = "EMAIL_ALREADY_VERIFIED"
)

// Convenience constructors for specific error types
func ErrorBadCredentials() *Error       { return NewError(ErrBadCredentials).WithHttpCode(401) }
func ErrorAccountLocked() *Error        { return NewError(ErrAccountLocked).WithHttpCode(400) }
func ErrorInvalidEmail() *Error         { return NewError(ErrInvalidEmail).WithHttpCode(400) }
func ErrorInvalidUsername() *Error      { return NewError(ErrInvalidUsername).WithHttpCode(400) }
func ErrorInvalidPassword() *Error      { return NewError(ErrInvalidPassword).WithHttpCode(400) }
func ErrorPrivateUser() *Error          { return NewError(ErrPrivateUser).WithHttpCode(403) }
func ErrorUserNotFound() *Error         { return NewError(ErrUserNotFound).WithHttpCode(404) }
func ErrorUsernameConflict() *Error     { return NewError(ErrUsernameConflict).WithHttpCode(409) }
func ErrorEmailConflict() *Error        { return NewError(ErrEmailConflict).WithHttpCode(409) }
func ErrorEmailAlreadyVerified() *Error { return NewError(ErrEmailAlreadyVerified).WithHttpCode(409) }
