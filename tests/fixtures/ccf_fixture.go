package fixtures

import (
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
)

// CreateDefaultCreditItem creates a valid default CCF item
func CreateDefaultCreditItem(index int) structs.CreditItemRequest {
	code := "CCF" + string(rune(65+index))

	return structs.CreditItemRequest{
		ItemRequest: structs.ItemRequest{
			Number:      index + 1,
			Type:        1,
			Description: "Producto CCF " + string(rune(65+index)),
			Quantity:    15,
			UnitMeasure: 59,
			UnitPrice:   10.0,
			Discount:    0,
			Code:        &code,
			Taxes:       []string{"20"},
		},
		NonSubjectSale: 0,
		ExemptSale:     0,
		TaxedSale:      150.0,
		SuggestedPrice: 0,
		NonTaxed:       0,
	}
}

// CreateDefaultCreditSummary creates a valid default CCF summary
func CreateDefaultCreditSummary() *structs.CreditSummaryRequest {
	return &structs.CreditSummaryRequest{
		SummaryRequest: structs.SummaryRequest{
			TotalNonSubject:    0,
			TotalExempt:        0,
			TotalTaxed:         300.0,
			SubTotal:           300.0,
			NonSubjectDiscount: 0,
			ExemptDiscount:     0,
			DiscountPercentage: 0,
			TotalDiscount:      0,
			SubTotalSales:      300.0,
			TotalOperation:     300.0,
			TotalNonTaxed:      0,
			TotalToPay:         300.0,
			OperationCondition: 1,
			Taxes: []structs.TaxRequest{
				{
					Code:        "20",
					Description: "IVA",
					Value:       39.0,
				},
			},
			PaymentTypes: []structs.PaymentRequest{
				{
					Code:   "01",
					Amount: 300.0,
				},
			},
		},
		TaxedDiscount:   0,
		IVAPerception:   0,
		IVARetention:    0,
		IncomeRetention: 0,
		BalanceInFavor:  0,
	}
}

// CreateDefaultCreditFiscalRequest creates a valid default CCF request
func CreateDefaultCreditFiscalRequest() *structs.CreateCreditFiscalRequest {
	items := []structs.CreditItemRequest{
		CreateDefaultCreditItem(1),
		CreateDefaultCreditItem(2),
	}

	return &structs.CreateCreditFiscalRequest{
		Items:     items,
		Receiver:  CreateDefaultReceiverWithoutDocsFields(),
		ModelType: 1,
		Summary:   CreateDefaultCreditSummary(),
	}
}

// CreateCreditFiscalWithInvalidItems creates a CCF request with invalid items
func CreateCreditFiscalWithInvalidItems() *structs.CreateCreditFiscalRequest {
	req := CreateDefaultCreditFiscalRequest()
	item := CreateDefaultCreditItem(0)
	item.Type = 99
	req.Items = []structs.CreditItemRequest{item}
	return req
}

// CreateCreditFiscalWithNonSubjectSale creates a CCF request with a non-subject sale (invalid)
func CreateCreditFiscalWithNonSubjectSale() *structs.CreateCreditFiscalRequest {
	req := CreateDefaultCreditFiscalRequest()
	item := CreateDefaultCreditItem(0)
	item.NonSubjectSale = 50.0
	req.Items = []structs.CreditItemRequest{item}
	return req
}

// CreateCCFRequestWithAllOptionalFields creates a CCF request with all optional fields
func CreateCCFRequestWithAllOptionalFields() *structs.CreateCreditFiscalRequest {
	req := CreateDefaultCreditFiscalRequest()
	req.Extension = CreateDefaultExtension()
	req.ThirdPartySale = CreateDefaultThirdPartySale()
	req.RelatedDocs = []structs.RelatedDocRequest{CreateDefaultRelatedDocument()}
	req.OtherDocs = []structs.OtherDocRequest{CreateDefaultOtherDocument()}
	req.Appendixes = []structs.AppendixRequest{CreateDefaultAppendix()}
	return req
}
