package validators

import (
	"log"

	"github.com/go-playground/validator/v10"
)

// RegisterValidators registers custom validators with the validator engine
func RegisterValidators(v *validator.Validate) {
	err := v.RegisterValidation("app_username", validateUsername)
	if err != nil {
		log.Fatalln(err)
	}
	err = v.RegisterValidation("app_username_strict", validateUsernameStrict)
	if err != nil {
		log.Fatalln(err)
	}
	err = v.RegisterValidation("app_email", validateEmail)
	if err != nil {
		log.Fatalln(err)
	}
	err = v.RegisterValidation("app_password", validatePassword)
	if err != nil {
		log.Fatalln(err)
	}
}
