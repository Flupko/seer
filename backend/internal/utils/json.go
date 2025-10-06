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
	ErrInvalidJSON        = errors.New("invalid JSON request body")
	ErrInvalidQueryParams = errors.New("invalid query parameters")
	ErrInvalidPathParams  = errors.New("invalid path parameters")
)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Message string            `json:"message"`
	Errors  []ValidationError `json:"errors,omitempty"`
}

func ParseAndValidateJSON(src io.Reader, dst any, validate *validator.Validate) error {

	decoder := json.NewDecoder(src)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, ErrInvalidJSON.Error())
	}

	return validateStruct(dst, validate)

}

func ParseAndValidateQueryParams(c echo.Context, dst any, validate *validator.Validate) error {

	if err := (&echo.DefaultBinder{}).BindQueryParams(c, dst); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, ErrInvalidQueryParams.Error())
	}

	return validateStruct(dst, validate)

}

func ParseAndValidadePathParams(c echo.Context, dst any, validate *validator.Validate) error {
	if err := (&echo.DefaultBinder{}).BindPathParams(c, dst); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, ErrInvalidPathParams.Error())
	}

	return validateStruct(dst, validate)
}

func validateStruct(v any, validate *validator.Validate) error {

	if err := validate.Struct(v); err != nil {
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
