package interfaces

import "github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"

// Validator Interface that defines the methods that must be implemented by objects that validate a field
type Validator interface {
	IsValid() bool
}

// DTEValidator Interface that defines the methods that must be implemented by objects that validate a DTE
type DTEValidator interface {
	ValidateDTERules() *dte_errors.DTEError
}

// ValueObject Interface that defines the methods that must be implemented by value objects
type ValueObject[T any] interface {
	Validator
	ToString() string
	Equals(value ValueObject[T]) bool
	GetValue() T
}

// DTEValidationStrategy Interface that defines the methods that must be implemented by DTE validation strategies
type DTEValidationStrategy interface {
	Validate() *dte_errors.DTEError
}
