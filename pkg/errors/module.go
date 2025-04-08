package appError

const (
	ErrModuleNotFound ErrorType = "MODULE_NOT_FOUND"
)

func ErrorModuleNotFound() *Error {
	return NewError(ErrModuleNotFound).WithHttpCode(404)
}
