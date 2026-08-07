package strategy

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/fse/fse_models"
)

type FSEItemStrategy struct {
	Document *fse_models.FSEModel
}

func NewFSEItemStrategy(document *fse_models.FSEModel) *FSEItemStrategy {
	return &FSEItemStrategy{
		Document: document,
	}
}

func (v *FSEItemStrategy) Validate() *dte_errors.DTEError {
	var validationErrors []*dte_errors.DTEError

	if v.Document == nil || len(v.Document.FSEItems) == 0 {
		validationErrors = append(validationErrors, dte_errors.NewDTEErrorSimple(
			"FSEItemInvalidEmpty",
			"Es requerido al menos un ítem para FSE",
		))
		return dte_errors.NewDTEErrorComposite(validationErrors)
	}

	for i, item := range v.Document.FSEItems {
		if err := v.validateItem(item, i+1); err != nil {
			validationErrors = append(validationErrors, err)
		}
	}

	if len(validationErrors) > 0 {
		return dte_errors.NewDTEErrorComposite(validationErrors)
	}

	return nil
}

func (v *FSEItemStrategy) validateItem(item fse_models.FSEItem, itemNumber int) *dte_errors.DTEError {
	if !item.Purchase.IsValid() || item.Purchase.GetValue() <= 0 {
		return dte_errors.NewDTEErrorSimple(
			"FSEItemInvalidPurchase",
			"El campo compra debe ser mayor a cero",
		)
	}

	if item.Item.Description == "" {
		return dte_errors.NewDTEErrorSimple(
			"FSEItemInvalidDescription",
			"La descripción del ítem no puede estar vacía",
		)
	}

	if !item.Item.Quantity.IsValid() || item.Item.Quantity.GetValue() <= 0 {
		return dte_errors.NewDTEErrorSimple(
			"FSEItemInvalidQuantity",
			"La cantidad debe ser mayor a cero",
		)
	}

	if !item.Item.UnitPrice.IsValid() || item.Item.UnitPrice.GetValue() <= 0 {
		return dte_errors.NewDTEErrorSimple(
			"FSEItemInvalidUnitPrice",
			"El precio unitario debe ser mayor a cero",
		)
	}

	return nil
}
