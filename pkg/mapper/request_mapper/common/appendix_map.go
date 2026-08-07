package common

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
)

// MapCommonRequestAppendix maps an array of appendixes to an appendix model -> Source: Request
func MapCommonRequestAppendix(request []structs.AppendixRequest) ([]models.Appendix, error) {
	result := make([]models.Appendix, len(request))

	for i, appendix := range request {

		if appendix.Field == "" || appendix.Label == "" || appendix.Value == "" {
			return nil, shared_error.NewGeneralServiceError("CommonMapper", "MapCommonRequestAppendix", "Field, Label and Value are required and cannot be empty in appendix", nil)
		}

		field, err := document.NewAppendixField(appendix.Field)
		if err != nil {
			return nil, err
		}

		label, err := document.NewAppendixLabel(appendix.Label)
		if err != nil {
			return nil, err
		}

		value, err := document.NewAppendixValue(appendix.Value)
		if err != nil {
			return nil, err
		}

		result[i] = models.Appendix{
			Field: *field,
			Label: *label,
			Value: *value,
		}
	}

	return result, nil
}
