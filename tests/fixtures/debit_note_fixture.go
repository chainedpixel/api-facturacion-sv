package fixtures

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

// CreateDefaultDebitNoteItem creates a valid default debit note item.
func CreateDefaultDebitNoteItem(index int) structs.DebitNoteItemRequest {
	code := "DN" + string(rune(65+index))

	return structs.DebitNoteItemRequest{
		ItemRequest: structs.ItemRequest{
			Number:      index + 1,
			Type:        1,
			Description: "Cargo adicional " + string(rune(65+index)),
			Quantity:    5,
			UnitMeasure: 59,
			UnitPrice:   10.0,
			Discount:    0,
			Code:        &code,
			Taxes:       []string{"20"},
		},
		NonSubjectSale: 0,
		ExemptSale:     0,
		TaxedSale:      50.0,
	}
}

// CreateDefaultDebitNoteSummary creates a valid default debit note summary.
func CreateDefaultDebitNoteSummary() *structs.DebitNoteSummaryRequest {
	return &structs.DebitNoteSummaryRequest{
		SummaryRequest: structs.SummaryRequest{
			TotalNonSubject:    0,
			TotalExempt:        0,
			TotalTaxed:         100.0,
			SubTotal:           100.0,
			NonSubjectDiscount: 0,
			ExemptDiscount:     0,
			DiscountPercentage: 0,
			TotalDiscount:      0,
			SubTotalSales:      100.0,
			TotalOperation:     113.0,
			TotalNonTaxed:      0,
			TotalToPay:         113.0,
			OperationCondition: 1,
			Taxes: []structs.TaxRequest{
				{
					Code:        "20",
					Description: "IVA",
					Value:       13.0,
				},
			},
			PaymentTypes: []structs.PaymentRequest{
				{
					Code:   "01",
					Amount: 113.0,
				},
			},
		},
		TaxedDiscount:   0,
		IVAPerception:   0,
		IVARetention:    0,
		IncomeRetention: 0,
	}
}

// CreateDefaultDebitNoteRequest creates a valid default debit note request.
func CreateDefaultDebitNoteRequest() *structs.CreateDebitNoteRequest {
	items := []structs.DebitNoteItemRequest{
		CreateDefaultDebitNoteItem(1),
		CreateDefaultDebitNoteItem(2),
	}

	receiver := CreateDefaultReceiver()
	receiver.NIT = utils.ToStringPointer("06141804941035")
	receiver.DocumentType = nil
	receiver.DocumentNumber = nil

	return &structs.CreateDebitNoteRequest{
		Items:     items,
		Receiver:  receiver,
		ModelType: constants.ModeloFacturacionPrevio,
		Summary:   CreateDefaultDebitNoteSummary(),
		RelatedDocs: []structs.RelatedDocRequest{
			CreateDefaultRelatedDocument(),
		},
	}
}

// CreateDebitNoteRequestWithAllOptionalFields creates a debit note request with all optional fields populated.
func CreateDebitNoteRequestWithAllOptionalFields() *structs.CreateDebitNoteRequest {
	req := CreateDefaultDebitNoteRequest()
	req.ThirdPartySale = CreateDefaultThirdPartySale()
	req.OtherDocs = []structs.OtherDocRequest{CreateDefaultOtherDocument()}
	req.Appendixes = []structs.AppendixRequest{CreateDefaultAppendix()}
	return req
}
