package utils

import (
	"encoding/base64"

	"github.com/go-playground/validator/v10"
)

func SetupValidator() *validator.Validate {
	v := validator.New()
	v.RegisterValidation("token_plain", validateTokenPlain)
	return v
}

func validateTokenPlain(fl validator.FieldLevel) bool {
	tokenPlain := fl.Field().String()
	if len(tokenPlain) != 43 {
		return false
	}
	_, err := base64.StdEncoding.WithPadding(base64.NoPadding).DecodeString(tokenPlain)
	return err == nil
}
