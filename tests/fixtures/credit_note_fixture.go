package fixtures

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

// CreateDefaultCreditNoteItem creates a valid default credit note item
func CreateDefaultCreditNoteItem(index int) structs.CreditNoteItemRequest {
	code := "CN" + string(rune(65+index))

	return structs.CreditNoteItemRequest{
		ItemRequest: structs.ItemRequest{
			Number:      index + 1,
			Type:        1,
			Description: "Devolución Producto " + string(rune(65+index)),
			Quantity:    5,
			UnitMeasure: 59,
			UnitPrice:   5.0,
			Discount:    0,
			Code:        &code,
			Taxes:       []string{"20"},
		},
		NonSubjectSale: 0,
		ExemptSale:     0,
		TaxedSale:      25.0,
	}
}

// CreateDefaultCreditNoteSummary creates a valid default credit note summary
func CreateDefaultCreditNoteSummary() *structs.CreditNoteSummaryRequest {
	return &structs.CreditNoteSummaryRequest{
		SummaryRequest: structs.SummaryRequest{
			TotalNonSubject:    0,
			TotalExempt:        0,
			TotalTaxed:         50.0,
			SubTotal:           50.0,
			NonSubjectDiscount: 0,
			ExemptDiscount:     0,
			DiscountPercentage: 0,
			TotalDiscount:      0,
			SubTotalSales:      50.0,
			TotalOperation:     50.0,
			TotalNonTaxed:      0,
			TotalToPay:         1,
			OperationCondition: 1,
			Taxes: []structs.TaxRequest{
				{
					Code:        "20",
					Description: "IVA",
					Value:       6.5,
				},
			},
			PaymentTypes: []structs.PaymentRequest{},
		},
		TaxedDiscount:   0,
		IVAPerception:   0,
		IVARetention:    0,
		IncomeRetention: 0,
	}
}

// CreateDefaultCreditNoteRequest creates a valid default credit note request
func CreateDefaultCreditNoteRequest() *structs.CreateCreditNoteRequest {
	items := []structs.CreditNoteItemRequest{
		CreateDefaultCreditNoteItem(1),
		CreateDefaultCreditNoteItem(2),
	}

	receiver := CreateDefaultReceiver()
	receiver.NIT = utils.ToStringPointer("06141804941035")
	receiver.DocumentType = nil
	receiver.DocumentNumber = nil

	return &structs.CreateCreditNoteRequest{
		Items:     items,
		Receiver:  receiver,
		ModelType: constants.ModeloFacturacionPrevio,
		Summary:   CreateDefaultCreditNoteSummary(),
		RelatedDocs: []structs.RelatedDocRequest{
			CreateDefaultRelatedDocument(),
		},
	}
}

// CreateCreditNoteWithoutRelatedDocs creates a credit note request without related documents
func CreateCreditNoteWithoutRelatedDocs() *structs.CreateCreditNoteRequest {
	req := CreateDefaultCreditNoteRequest()
	req.RelatedDocs = nil
	return req
}

// CreateCreditNoteRequestWithAllOptionalFields creates a credit note request with all optional fields
func CreateCreditNoteRequestWithAllOptionalFields() *structs.CreateCreditNoteRequest {
	req := CreateDefaultCreditNoteRequest()
	req.ThirdPartySale = CreateDefaultThirdPartySale()
	req.RelatedDocs = []structs.RelatedDocRequest{CreateDefaultRelatedDocument()}
	req.OtherDocs = []structs.OtherDocRequest{CreateDefaultOtherDocument()}
	req.Appendixes = []structs.AppendixRequest{CreateDefaultAppendix()}
	return req
}
