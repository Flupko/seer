package utils

import (
	"encoding/base64"
	"fmt"
	"reflect"
	"seer/internal/numeric"
	"strconv"
	"strings"

	"github.com/ericlagergren/decimal"
	"github.com/go-playground/validator/v10"
)

func SetupValidator() *validator.Validate {
	v := validator.New()

	v.RegisterTagNameFunc(func(fld reflect.StructField) string {

		if name := fld.Tag.Get("json"); name != "" {
			return name
		}

		if name := fld.Tag.Get("query"); name != "" {
			return name
		}

		if name := fld.Tag.Get("form"); name != "-" {
			return name
		}

		return ""
	})

	v.RegisterCustomTypeFunc(func(field reflect.Value) any {
		if !field.IsValid() {
			return nil
		}

		bd, ok := field.Interface().(*numeric.BigDecimal)
		if !ok || bd == nil {
			return nil
		}

		return bd
	}, (*numeric.BigDecimal)(nil))

	v.RegisterValidation("dec_min", BigDecimalMinValidation)
	v.RegisterValidation("dec_max", BigDecimalMaxValidation)
	v.RegisterValidation("dec_scale", BigDecimalScaleValidation)

	v.RegisterValidation("token_plain", validateTokenPlain)
	v.RegisterValidation("slug", validateSlug)
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

func validateSlug(fl validator.FieldLevel) bool {
	slug := fl.Field().String()
	for _, ch := range slug {
		if (ch < 'a' || ch > 'z') && (ch < '0' || ch > '9') && ch != '-' {
			return false
		}
	}
	return true
}

func BigDecimalMinValidation(fl validator.FieldLevel) bool {
	bd, ok := (fl.Field().Interface()).(numeric.BigDecimal)
	if !ok {
		return false
	}

	min := new(decimal.Big)
	if _, ok := min.SetString(fl.Param()); !ok {
		return false
	}
	return bd.Big.Cmp(min) >= 0
}

func BigDecimalMaxValidation(fl validator.FieldLevel) bool {
	bd, ok := (fl.Field().Interface()).(numeric.BigDecimal)
	if !ok {
		return false
	}

	max := new(decimal.Big)
	if _, ok := max.SetString(fl.Param()); !ok {
		return false
	}
	return bd.Big.Cmp(max) <= 0
}

func BigDecimalScaleValidation(fl validator.FieldLevel) bool {

	bd, ok := (fl.Field().Interface()).(numeric.BigDecimal)
	if !ok {
		return false
	}

	bdStr := bd.String()
	fmt.Println(bdStr)
	idx := strings.Index(bdStr, ".")
	if idx == -1 {
		// No decimal point so scale is 0
		return true
	}
	n, _ := strconv.Atoi(fl.Param())
	return len(bdStr)-idx-1 <= n
}
