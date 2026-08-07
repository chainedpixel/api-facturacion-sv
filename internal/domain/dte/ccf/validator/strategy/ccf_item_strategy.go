package strategy

import (
	"github.com/shopspring/decimal"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/ccf/ccf_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
)

type CCFItemStrategy struct {
	Document *ccf_models.CreditFiscalDocument
}

func (s *CCFItemStrategy) Validate() *dte_errors.DTEError {
	if s.Document == nil {
		return nil
	}

	for _, item := range s.Document.CreditItems {
		if err := s.validateItemSaleTypes(&item); err != nil {
			return err
		}

		if err := s.validateCCFTaxRules(&item); err != nil {
			return err
		}

		if err := s.validateCCFType4Rules(&item); err != nil {
			return err
		}

		if err := s.validateItemNonTaxedRules(&item); err != nil {
			return err
		}

		if err := s.validateItem(&item); err != nil {
			return err
		}
	}

	if err := s.validateTotalNonTaxed(); err != nil {
		return err
	}

	return nil
}

func (s *CCFItemStrategy) validateItem(item *ccf_models.CreditItem) *dte_errors.DTEError {
	if item.TaxedSale.GetValue() > 0 && item.GetUnitPrice() == 0 {
		logs.Error("Unit price cannot be zero when taxed sale is present", map[string]interface{}{
			"itemNumber": item.GetNumber(),
			"taxedSale":  item.TaxedSale.GetValue(),
		})
		return dte_errors.NewDTEErrorSimple("InvalidUnitPriceZero",
			item.GetNumber(), item.GetUnitPrice(), item.TaxedSale.GetValue())
	}
	return nil
}

func (s *CCFItemStrategy) validateItemNonTaxedRules(item *ccf_models.CreditItem) *dte_errors.DTEError {
	nonTaxed := item.NonTaxed.GetValue()
	if nonTaxed > 0 && item.GetUnitPrice() == 0 {
		unitPrice := item.GetUnitPrice()
		taxedSale := item.TaxedSale.GetValue()

		if unitPrice != taxedSale {
			logs.Error("Unit price must equal taxed sale when non_taxed > 0", map[string]interface{}{
				"itemNumber": item.GetNumber(),
				"unitPrice":  unitPrice,
				"taxedSale":  taxedSale,
			})
			return dte_errors.NewDTEErrorSimple("InvalidUnitPrice",
				item.GetNumber(), unitPrice, taxedSale)
		}
	}
	return nil
}

func (s *CCFItemStrategy) validateTotalNonTaxed() *dte_errors.DTEError {
	var totalNonTaxed float64
	for _, item := range s.Document.CreditItems {
		totalNonTaxed += item.NonTaxed.GetValue()
	}

	totalToPay := decimal.NewFromFloat(s.Document.CreditSummary.TotalToPay.GetValue())
	totalOperation := decimal.NewFromFloat(s.Document.CreditSummary.TotalOperation.GetValue())
	totalNonTaxedDecimal := decimal.NewFromFloat(totalNonTaxed)
	perception := decimal.NewFromFloat(s.Document.CreditSummary.IVAPerception.GetValue())
	ivaRetention := decimal.NewFromFloat(s.Document.CreditSummary.IVARetention.GetValue())
	incomeRetention := decimal.NewFromFloat(s.Document.CreditSummary.IncomeRetention.GetValue())

	expectedTotalToPay := totalOperation.
		Add(totalNonTaxedDecimal).
		Add(perception).
		Sub(ivaRetention).
		Sub(incomeRetention)

	diff := totalToPay.Sub(expectedTotalToPay).Abs()
	if diff.GreaterThan(decimal.NewFromFloat(0.01)) {
		logs.Error("Total to pay must be equal to total operation plus sum of non_taxed amounts", map[string]interface{}{
			"totalToPay":     totalToPay,
			"totalOperation": totalOperation,
			"totalNonTaxed":  totalNonTaxedDecimal,
			"perception":     perception,
			"expectedTotal":  expectedTotalToPay,
			"difference":     diff,
		})
		return dte_errors.NewDTEErrorSimple("InvalidTotalToPayNonTaxedCCF",
			totalToPay.InexactFloat64(),
			totalOperation.InexactFloat64(),
			totalNonTaxedDecimal.InexactFloat64(),
			perception.InexactFloat64(),
			expectedTotalToPay.InexactFloat64())
	}

	return nil
}

func (s *CCFItemStrategy) validateCCFTaxRules(item *ccf_models.CreditItem) *dte_errors.DTEError {
	if item.TaxedSale.GetValue() > 0 && len(item.GetTaxes()) == 0 {
		logs.Error("No taxes present with non-zero taxed sale")
		return dte_errors.NewDTEErrorSimple("MissingTaxesItem", item.GetNumber())
	}

	for _, tax := range item.GetTaxes() {
		if item.GetType() == constants.Producto && tax != constants.TaxIVA {
			logs.Error("Invalid tax code, only IVA (20) is allowed", map[string]interface{}{
				"itemNumber": item.GetNumber(),
				"taxCode":    tax,
			})
			return dte_errors.NewDTEErrorSimple("InvalidTaxCodeOnly20", tax, item.GetType())
		}
	}

	if len(s.Document.RelatedDocuments) > 0 {
		if item.GetRelatedDoc() == nil {
			logs.Error("Missing related document in item when related_docs is present", map[string]interface{}{
				"itemNumber": item.GetNumber(),
			})
			return dte_errors.NewDTEErrorSimple("MissingItemRelatedDoc", item.GetNumber())
		}

		found := false
		itemRelatedDoc := *item.GetRelatedDoc()

		for _, relatedDoc := range s.Document.RelatedDocuments {
			if relatedDoc.GetDocumentNumber() == itemRelatedDoc {
				found = true
				break
			}
		}

		if !found {
			logs.Error("Item related document not found in document related docs", map[string]interface{}{
				"itemNumber": item.GetNumber(),
				"relatedDoc": itemRelatedDoc,
			})
			return dte_errors.NewDTEErrorSimple("InvalidItemRelatedDoc",
				item.GetNumber(),
				itemRelatedDoc)
		}
	}

	if item.GetType() == constants.Impuesto {
		for _, tax := range item.GetTaxes() {
			if tax != constants.TaxIVA {
				return dte_errors.NewDTEErrorSimple("InvalidTaxCodeOnly20", item.GetType(), tax)
			}
		}
	}

	return nil
}

func (s *CCFItemStrategy) validateItemSaleTypes(item *ccf_models.CreditItem) *dte_errors.DTEError {
	if item.NonTaxed.GetValue() > 0 {
		if item.TaxedSale.GetValue() > 0 || item.ExemptSale.GetValue() > 0 || item.NonSubjectSale.GetValue() > 0 {
			logs.Error("Items with non-taxed amount cannot have other sale types", map[string]interface{}{
				"itemNumber":     item.GetNumber(),
				"nonTaxed":       item.NonTaxed.GetValue(),
				"taxedSale":      item.TaxedSale.GetValue(),
				"exemptSale":     item.ExemptSale.GetValue(),
				"nonSubjectSale": item.NonSubjectSale.GetValue(),
			})
			return dte_errors.NewDTEErrorSimple("InvalidMixedSalesWithNonTaxed", item.GetNumber())
		}

		if len(item.GetTaxes()) > 0 {
			logs.Error("Items with non-taxed amount cannot have taxes", map[string]interface{}{
				"itemNumber": item.GetNumber(),
				"taxes":      item.GetTaxes(),
			})
			return dte_errors.NewDTEErrorSimple("InvalidTaxesWithNonTaxed", item.GetNumber())
		}

		if item.GetUnitPrice() != 0 {
			logs.Error("Items with non-taxed amount must have zero unit price", map[string]interface{}{
				"itemNumber": item.GetNumber(),
				"unitPrice":  item.GetUnitPrice(),
			})
			return dte_errors.NewDTEErrorSimple("InvalidUnitPriceWithNonTaxed", item.GetNumber())
		}
	}

	if item.ExemptSale.GetValue() > 0 {
		if item.TaxedSale.GetValue() > 0 || item.NonSubjectSale.GetValue() > 0 || item.NonTaxed.GetValue() > 0 {
			logs.Error("Items with exempt sale cannot have other sale types", map[string]interface{}{
				"itemNumber":     item.GetNumber(),
				"exemptSale":     item.ExemptSale.GetValue(),
				"taxedSale":      item.TaxedSale.GetValue(),
				"nonSubjectSale": item.NonSubjectSale.GetValue(),
				"nonTaxed":       item.NonTaxed.GetValue(),
			})
			return dte_errors.NewDTEErrorSimple("InvalidMixedSalesWithExempt", item.GetNumber())
		}

		if len(item.GetTaxes()) > 0 {
			logs.Error("Items with exempt sale cannot have taxes", map[string]interface{}{
				"itemNumber": item.GetNumber(),
				"taxes":      item.GetTaxes(),
			})
			return dte_errors.NewDTEErrorSimple("InvalidTaxesWithExempt", item.GetNumber())
		}
	}

	if item.NonSubjectSale.GetValue() > 0 {
		if item.TaxedSale.GetValue() > 0 || item.ExemptSale.GetValue() > 0 || item.NonTaxed.GetValue() > 0 {
			logs.Error("Items with non-subject sale cannot have other sale types", map[string]interface{}{
				"itemNumber":     item.GetNumber(),
				"nonSubjectSale": item.NonSubjectSale.GetValue(),
				"taxedSale":      item.TaxedSale.GetValue(),
				"exemptSale":     item.ExemptSale.GetValue(),
				"nonTaxed":       item.NonTaxed.GetValue(),
			})
			return dte_errors.NewDTEErrorSimple("InvalidMixedSalesWithNonSubject", item.GetNumber())
		}

		if len(item.GetTaxes()) > 0 {
			logs.Error("Items with non-subject sale cannot have taxes", map[string]interface{}{
				"itemNumber": item.GetNumber(),
				"taxes":      item.GetTaxes(),
			})
			return dte_errors.NewDTEErrorSimple("InvalidTaxesWithNonSubject", item.GetNumber())
		}
	}

	return nil
}

func (s *CCFItemStrategy) validateCCFType4Rules(item interfaces.Item) *dte_errors.DTEError {
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
