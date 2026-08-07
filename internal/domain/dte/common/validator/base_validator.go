package validator

import (
	"reflect"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

// ValidateModel validates a model and returns the validation errors found.
// It validates that fields of type ValueObject comply with the defined business rules.
func ValidateModel[T any](model T) []error {
	var validationErrors []error
	v := reflect.ValueOf(model)

	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return validationErrors
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return validationErrors
	}

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := v.Type().Field(i)

		if !fieldType.IsExported() {
			continue
		}

		if field.Kind() == reflect.Slice {
			sliceErrors := validateSlice(field)
			validationErrors = append(validationErrors, sliceErrors...)
			continue
		}

		if field.Kind() == reflect.Ptr {
			if field.IsNil() {
				continue
			}
			field = field.Elem()
		}

		if !field.IsValid() {
			continue
		}

		if field.Kind() == reflect.Struct {
			if field.CanInterface() {
				structErrors := ValidateModel(field.Interface())
				validationErrors = append(validationErrors, structErrors...)
			}
			continue
		}

		if field.CanInterface() {
			if validator, ok := field.Interface().(interfaces.ValueObject[any]); ok {
				if !validator.IsValid() {
					validationErrors = append(validationErrors,
						dte_errors.NewValidationError("InvalidField", fieldType.Name))
				}
			}
		}
	}

	return validationErrors
}

// validateSlice validates the elements of a slice and returns the validation errors found.
// It validates that fields of type ValueObject comply with the defined business rules.
func validateSlice(field reflect.Value) []error {
	var sliceErrors []error

	for i := 0; i < field.Len(); i++ {
		element := field.Index(i)

		if element.Kind() == reflect.Ptr {
			if element.IsNil() {
				continue
			}
			element = element.Elem()
		}

		if !element.IsValid() {
			continue
		}

		if element.Kind() == reflect.Struct {
			if element.CanInterface() {
				elementErrors := ValidateModel(element.Interface())
				sliceErrors = append(sliceErrors, elementErrors...)
			}
		}
	}

	return sliceErrors
}
