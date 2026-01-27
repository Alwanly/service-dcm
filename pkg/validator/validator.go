package validator

import (
	"sync"

	"github.com/go-playground/validator/v10"
)

var lock = &sync.Mutex{}
var validate *validator.Validate

func getValidator() *validator.Validate {
	if validate == nil {
		lock.Lock()
		defer lock.Unlock()
		if validate == nil {
			validate = validator.New(validator.WithRequiredStructEnabled())
		}
	}
	return validate
}

func ValidateStruct(s interface{}) error {
	return getValidator().Struct(s)
}

func TranslateError(err error) map[string]string {
	errors := make(map[string]string)
	if err == nil {
		return errors
	}
	for _, err := range err.(validator.ValidationErrors) {
		errors[err.Field()] = err.Error()
	}
	return errors
}
