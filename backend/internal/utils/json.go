package utils

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type Envelope map[string]any

var (
	ErrInvalidJSON = errors.New("invalid JSON request body")
)

func BindAndValidate(c echo.Context, dst any, validate *validator.Validate) error {

	decoder := json.NewDecoder(c.Request().Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, ErrInvalidJSON.Error())
	}

	if err := validate.Struct(dst); err != nil {
		var validateErrs validator.ValidationErrors
		if errors.As(err, &validateErrs) {
			invalidFields := []string{}
			for _, e := range validateErrs {
				invalidFields = append(invalidFields, e.Field())
			}
			return echo.NewHTTPError(http.StatusBadRequest, Envelope{
				"error":          "validation failed",
				"invalid_fields": invalidFields,
			})
		}
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	return nil

}

func ReadJson(src io.Reader, dst any) error {
	dec := json.NewDecoder(src)
	dec.DisallowUnknownFields()

	err := dec.Decode(dst)

	if err != nil {
		var invalidUnmarshalError *json.InvalidUnmarshalError
		if errors.As(err, &invalidUnmarshalError) {
			return invalidUnmarshalError
		}
		return ErrInvalidJSON
	}

	if dec.More() {
		return ErrInvalidJSON
	}

	return nil
}
