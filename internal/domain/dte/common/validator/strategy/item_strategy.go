package strategy

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
)

// ItemValidationStrategy implements the validation strategy for items
type ItemValidationStrategy struct {
	Document interfaces.DTEDocument
}

func (s *ItemValidationStrategy) Validate() *dte_errors.DTEError {
	if s.Document == nil || len(s.Document.GetItems()) == 0 {
		return dte_errors.NewDTEErrorSimple("RequiredField", "Items")
	}

	var validationErrors []*dte_errors.DTEError

	if len(s.Document.GetItems()) > 2000 {
		return dte_errors.NewDTEErrorSimple("ExceededItemsLimit", len(s.Document.GetItems()))
	}

	for _, item := range s.Document.GetItems() {
		if err := s.validateItem(item); err != nil {
			validationErrors = append(validationErrors, err)
		}
	}

	if len(validationErrors) > 0 {
		return dte_errors.NewDTEErrorComposite(validationErrors)
	}

	return nil
}

func (s *ItemValidationStrategy) validateItem(item interfaces.Item) *dte_errors.DTEError {
	if item.GetNumber() < 1 || item.GetNumber() > 2000 {
		logs.Error("Invalid item number", map[string]interface{}{
			"number": item.GetNumber(),
		})
		return dte_errors.NewDTEErrorSimple("InvalidItemNumber", item.GetNumber())
	}

	itemType := item.GetType()
	validType := false
	for _, t := range constants.AllowedItemTypes {
		if itemType == t {
			validType = true
			break
		}
	}
	if !validType {
		logs.Error("Invalid item type", map[string]interface{}{
			"type": itemType,
		})
		return dte_errors.NewDTEErrorSimple("InvalidItemType", itemType)
	}

	if len(item.GetDescription()) == 0 || len(item.GetDescription()) > 1000 {
		logs.Error("Invalid description length", map[string]interface{}{
			"length": len(item.GetDescription()),
		})
		return dte_errors.NewDTEErrorSimple("InvalidLength", "Item description", "1-1000", item.GetDescription())
	}

	if item.GetUnitMeasure() < 1 || item.GetUnitMeasure() > 99 {
		logs.Error("Invalid unit measure", map[string]interface{}{
			"measure": item.GetUnitMeasure(),
		})
		return dte_errors.NewDTEErrorSimple("InvalidNumberRange", "UnitMeasure", "1-99", item.GetUnitMeasure())
	}

	if item.GetItemCode() != "" && len(item.GetItemCode()) > 25 {
		logs.Error("Invalid item code length", map[string]interface{}{
			"length": len(item.GetItemCode()),
		})
		return dte_errors.NewDTEErrorSimple("InvalidLength", "Item code", "1-25", string(rune(len(item.GetItemCode()))))
	}

	return nil
}
