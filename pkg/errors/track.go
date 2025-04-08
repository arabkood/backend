package appError

const (
	ErrTrackAlreadyStarted ErrorType = "TRACK_ALREADY_STARTED"
	ErrTrackNotFound       ErrorType = "TRACK_NOT_FOUND"
)

func ErrorTrackAlreadyStarted() *Error {
	return NewError(ErrTrackAlreadyStarted).WithHttpCode(409)
}

func ErrorTrackNotFound() *Error {
	return NewError(ErrTrackNotFound).WithHttpCode(404)
}
