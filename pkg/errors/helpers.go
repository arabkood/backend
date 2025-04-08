package appError

import (
	"strings"

	"github.com/go-playground/validator/v10"
)

func StandarizeValidationErrors(ve validator.ValidationErrors) *Error {
	for _, fe := range ve {
		switch strings.ToLower(fe.Field()) {
		case "email":
			return ErrorInvalidEmail().WithMessage(fe.Error())
		case "username":
			return ErrorInvalidUsername().WithMessage(fe.Error())
		case "password":
			return ErrorInvalidPassword().WithMessage(fe.Error())
		}
	}
	if len(ve) > 0 {
		return ErrorInvalidInput().WithMessage(ve[0].Error())
	}
	return ErrorInvalidInput()
}
