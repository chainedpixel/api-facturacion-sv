package validator

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
)

// ValidateDTEDocument validates a DTE document and returns an error if it does not comply with business rules.
// Three types of errors are handled:
// 1. DTE validation errors (DTE business rule validation errors)
// 2. Value object validation errors
// 3. Both value object and DTE validation errors, returns a CompositeError with the validation errors
func ValidateDTEDocument[T interfaces.DTEValidator](doc T) error {
	validationErrors := ValidateModel(doc)

	if dteErr := doc.ValidateDTERules(); dteErr != nil {
		logs.Error("Failed to validate DTE rules", map[string]interface{}{"error": dteErr.Error()})
		dteErr.ValidationErrors = append(dteErr.ValidationErrors, validationErrors...)
		return dteErr
	}

	if len(validationErrors) > 0 {
		logs.Error("Failed to validate DTE document", map[string]interface{}{"error": dte_errors.NewCompositeError(validationErrors...).Error()})
		return dte_errors.NewCompositeError(validationErrors...)
	}

	return nil
}
