package appError

const (
	ErrInvalidToken ErrorType = "INVALID_TOKEN"

	ErrInvalidEmailVerificationToken = "INVALID_EMAIL_VERIFICATION_TOKEN"
	ErrInvalidRecoveryToken          = "INVALID_RECOVERY_TOKEN"
	ErrInvalidAuthSessionToken       = "INVALID_SESSION_TOKEN"

	ErrMissingAuthToken ErrorType = "MISSING_AUTHORIZATION_TOKEN"
	ErrInvalidAuthToken ErrorType = "INVALID_AUTHORIZATION_TOKEN"
	ErrExpiredAuthToken ErrorType = "EXPIRED_AUTHORIZATION_TOKEN"
)

func ErrorInvalidEmailVerificationToken() *Error {
	return NewError(ErrInvalidEmailVerificationToken).WithHttpCode(404)
}

func ErrorInvalidToken() *Error {
	return NewError(ErrInvalidToken).WithHttpCode(400)
}

func ErrorMissingAuthToken() *Error { return NewError(ErrMissingAuthToken).WithHttpCode(401) }
func ErrorInvalidAuthToken() *Error { return NewError(ErrInvalidAuthToken).WithHttpCode(401) }
func ErrorExpiredAuthToken() *Error { return NewError(ErrExpiredAuthToken).WithHttpCode(401) }
func ErrorInvalidAuthSessionToken() *Error {
	return NewError(ErrInvalidAuthSessionToken).WithHttpCode(401)
}

func ErrorInvalidRecoveryToken() *Error {
	return NewError(ErrInvalidRecoveryToken).WithHttpCode(404)
}
