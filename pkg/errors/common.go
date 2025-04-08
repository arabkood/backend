package appError

// Common errors
const (
	ErrInternal     ErrorType = "INTERNAL"
	ErrInvalidInput ErrorType = "INVALID_INPUT"
	ErrNotFound     ErrorType = "NOT_FOUND"
	ErrUnauthorized ErrorType = "UNAUTHORIZED"
)

func ErrorNotFound() *Error {
	return NewError(ErrNotFound).WithHttpCode(404)
}

func ErrorInternal() *Error {
	return NewError(ErrInternal).WithHttpCode(500)
}

func ErrorInvalidInput() *Error {
	return NewError(ErrInvalidInput).WithHttpCode(400)
}

func ErrorUnauthorized() *Error {
	return NewError(ErrUnauthorized).WithHttpCode(401)
}
