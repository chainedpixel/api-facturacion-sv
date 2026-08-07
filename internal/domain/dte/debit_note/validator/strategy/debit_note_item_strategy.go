package strategy

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/debit_note/debit_note_models"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
)

type DebitNoteItemStrategy struct {
	Document *debit_note_models.DebitNoteModel
}

func (s *DebitNoteItemStrategy) Validate() *dte_errors.DTEError {
	if s.Document == nil || len(s.Document.DebitItems) == 0 {
		return dte_errors.NewDTEErrorSimple("RequiredField", "DebitItems")
	}

	if len(s.Document.DebitItems) > 2000 {
		return dte_errors.NewDTEErrorSimple("ExceededItemsLimit", len(s.Document.DebitItems))
	}

	for _, item := range s.Document.DebitItems {
		if err := s.validateItemSaleTypes(&item); err != nil {
			return err
		}

		if err := s.validateItem(&item); err != nil {
			return err
		}

		if err := s.validateDebitNoteType4Rules(&item); err != nil {
			return err
		}
	}

	return nil
}

func (s *DebitNoteItemStrategy) validateItem(item *debit_note_models.DebitNoteItem) *dte_errors.DTEError {
	if item.TaxedSale.GetValue() > 0 && item.GetUnitPrice() == 0 {
		logs.Error("Unit price cannot be zero when taxed sale is present", map[string]interface{}{
			"itemNumber": item.GetNumber(),
			"taxedSale":  item.TaxedSale.GetValue(),
		})
		return dte_errors.NewDTEErrorSimple("InvalidUnitPriceZero",
			item.GetNumber(), item.GetUnitPrice(), item.TaxedSale.GetValue())
	}

	if item.GetRelatedDoc() == nil {
		logs.Error("Related document is required for debit note items", map[string]interface{}{
			"itemNumber": item.GetNumber(),
		})
		return dte_errors.NewDTEErrorSimple("MissingItemRelatedDoc", item.GetNumber())
	}

	if item.TaxedSale.GetValue() > 0 {
		if item.GetTaxes() == nil || len(item.GetTaxes()) == 0 {
			logs.Error("At least one tax is required for debit note items", map[string]interface{}{
				"itemNumber": item.GetNumber(),
			})
			return dte_errors.NewDTEErrorSimple("MissingItemTaxes", item.GetNumber())
		}
	}

	if item.GetType() != constants.Impuesto {
		for _, tax := range item.GetTaxes() {
			if !constants.MapAllowedTaxTypes[tax] {
				logs.Error("Invalid tax type", map[string]interface{}{
					"itemNumber": item.GetNumber(),
					"tax":        tax,
				})
				return dte_errors.NewDTEErrorSimple("InvalidTaxType", item.GetNumber(), tax)
			}
		}
	}

	return nil
}

func (s *DebitNoteItemStrategy) validateItemSaleTypes(item *debit_note_models.DebitNoteItem) *dte_errors.DTEError {
	salesTypes := 0
	if item.TaxedSale.GetValue() > 0 {
		salesTypes++
	}
	if item.ExemptSale.GetValue() > 0 {
		salesTypes++
	}
	if item.NonSubjectSale.GetValue() > 0 {
		salesTypes++
	}

	if salesTypes > 1 {
		logs.Error("Mixed sales types in single item", map[string]interface{}{
			"itemNumber":     item.GetNumber(),
			"taxedSale":      item.TaxedSale.GetValue(),
			"exemptSale":     item.ExemptSale.GetValue(),
			"nonSubjectSale": item.NonSubjectSale.GetValue(),
		})
		return dte_errors.NewDTEErrorSimple("MixedSalesTypesNotAllowed", item.GetNumber())
	}

	return nil
}

func (s *DebitNoteItemStrategy) validateDebitNoteType4Rules(item interfaces.Item) *dte_errors.DTEError {
	if item.GetType() == constants.Impuesto {
		if item.GetUnitMeasure() != 99 {
			return dte_errors.NewDTEErrorSimple("InvalidUnitMeasure", item.GetUnitMeasure())
		}

		if len(item.GetTaxes()) != 1 || item.GetTaxes()[0] != constants.TaxIVA {
			return dte_errors.NewDTEErrorSimple("InvalidTaxRulesCCF")
		}
	}

	return nil
}
