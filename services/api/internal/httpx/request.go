package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

var (
	validate      = validator.New()
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
)

func init() {
	// Use JSON field names in validation error keys instead of struct field names.
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return fld.Name
		}
		return name
	})

	_ = validate.RegisterValidation("username", func(fl validator.FieldLevel) bool {
		return usernameRegex.MatchString(fl.Field().String())
	})
}

// Decode parses the JSON request body into v.
func Decode(r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}
	return nil
}

// DecodeValidate decodes the JSON body and runs struct validation.
// Returns (nil, nil) on success, (nil, err) on JSON error, (map, nil) on validation failure.
func DecodeValidate(r *http.Request, v any) (map[string]string, error) {
	if err := Decode(r, v); err != nil {
		return nil, err
	}

	if err := validate.Struct(v); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			errs := make(map[string]string, len(ve))
			for _, fe := range ve {
				errs[fe.Field()] = validationMessage(fe)
			}
			return errs, nil
		}
		return nil, err
	}

	return nil, nil
}

func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "обязательное поле"
	case "min":
		return fmt.Sprintf("минимум %s символов", fe.Param())
	case "max":
		return fmt.Sprintf("максимум %s символов", fe.Param())
	case "username":
		return "только буквы, цифры и нижнее подчёркивание"
	default:
		return "некорректное значение"
	}
}
