// Package validate is a thin wrapper around go-playground/validator/v10.
//
// It adds:
//
//   - A package-level Default() validator with json-tag-aware field names, so
//     error messages reference the JSON name a client actually sees.
//   - FieldError / Errors types that play nicely with HTTP handlers: each
//     field has Field, Tag, Param, Value, Message.
//   - RegisterRule for adding custom validation rules without exposing the
//     underlying validator type.
package validate

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

// FieldError describes a single failed validation.
type FieldError struct {
	Field   string `json:"field"`             // dot-notated path using JSON names
	Tag     string `json:"tag"`               // rule tag, e.g. "required" / "min"
	Param   string `json:"param,omitempty"`   // rule parameter, e.g. "8"
	Value   any    `json:"value,omitempty"`   // the value that failed validation
	Message string `json:"message"`           // human-friendly description
}

// Errors is the aggregate result of Validate when one or more fields failed.
type Errors []FieldError

func (e Errors) Error() string {
	if len(e) == 0 {
		return "validate: no errors"
	}
	parts := make([]string, 0, len(e))
	for _, fe := range e {
		parts = append(parts, fe.Message)
	}
	return strings.Join(parts, "; ")
}

// AsErrors converts err to Errors when it was produced by Validate. Returns
// (nil, false) for any other error.
func AsErrors(err error) (Errors, bool) {
	var es Errors
	if errors.As(err, &es) {
		return es, true
	}
	return nil, false
}

// Validator wraps validator.Validate.
type Validator struct {
	v *validator.Validate
}

// New returns a Validator with json-tag-aware field naming.
func New() *Validator {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		tag := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if tag == "-" || tag == "" {
			return fld.Name
		}
		return tag
	})
	return &Validator{v: v}
}

// Raw exposes the underlying *validator.Validate for advanced features
// (custom translators, struct-level rules, etc).
func (v *Validator) Raw() *validator.Validate { return v.v }

// Validate runs the rules attached to s (a struct or pointer-to-struct) and
// returns Errors when one or more rules failed.
func (v *Validator) Validate(s any) error {
	if err := v.v.Struct(s); err != nil {
		var vErrs validator.ValidationErrors
		if errors.As(err, &vErrs) {
			out := make(Errors, 0, len(vErrs))
			for _, fe := range vErrs {
				out = append(out, FieldError{
					Field:   fe.Field(),
					Tag:     fe.Tag(),
					Param:   fe.Param(),
					Value:   fe.Value(),
					Message: humanise(fe),
				})
			}
			return out
		}
		return fmt.Errorf("validate: %w", err)
	}
	return nil
}

// RegisterRule registers a custom rule. See the go-playground/validator README
// for the function signature semantics.
func (v *Validator) RegisterRule(tag string, fn validator.Func) error {
	if err := v.v.RegisterValidation(tag, fn); err != nil {
		return fmt.Errorf("validate: register %q: %w", tag, err)
	}
	return nil
}

func humanise(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", fe.Field())
	case "email":
		return fmt.Sprintf("%s must be a valid email", fe.Field())
	case "min":
		return fmt.Sprintf("%s must be at least %s", fe.Field(), fe.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s", fe.Field(), fe.Param())
	case "len":
		return fmt.Sprintf("%s must be exactly %s long", fe.Field(), fe.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of [%s]", fe.Field(), fe.Param())
	case "url":
		return fmt.Sprintf("%s must be a valid URL", fe.Field())
	case "uuid":
		return fmt.Sprintf("%s must be a valid UUID", fe.Field())
	case "gte":
		return fmt.Sprintf("%s must be >= %s", fe.Field(), fe.Param())
	case "lte":
		return fmt.Sprintf("%s must be <= %s", fe.Field(), fe.Param())
	default:
		return fmt.Sprintf("%s failed %s validation", fe.Field(), fe.Tag())
	}
}

// --- Package default ---------------------------------------------------------

var (
	defaultOnce sync.Once
	defaultV    *Validator
)

// Default returns the package-level Validator. It is lazily initialised on
// first use, so registering custom rules via Default().RegisterRule(...)
// affects all subsequent Validate calls in the process.
func Default() *Validator {
	defaultOnce.Do(func() { defaultV = New() })
	return defaultV
}

// Validate is a shortcut for Default().Validate(s).
func Validate(s any) error { return Default().Validate(s) }
