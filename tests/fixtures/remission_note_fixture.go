package fixtures

import (
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

// CreateDefaultRemissionNoteItem creates a valid default remission note item.
func CreateDefaultRemissionNoteItem(index int) *structs.RemissionNoteItemRequest {
	num := index + 1
	itemType := 1
	desc := "Artículo remisión " + string(rune(65+index))
	qty := 5.0
	unitMeasure := 59
	unitPrice := 20.0
	discount := 0.0
	nonSubject := 0.0
	exempt := 0.0
	taxed := 100.0
	return &structs.RemissionNoteItemRequest{
		ItemNumber:     &num,
		ItemType:       &itemType,
		Description:    &desc,
		Quantity:       &qty,
		UnitMeasure:    &unitMeasure,
		UnitPrice:      &unitPrice,
		DiscountAmount: &discount,
		NonSubjectSale: &nonSubject,
		ExemptSale:     &exempt,
		TaxedSale:      &taxed,
	}
}

// CreateDefaultRemissionNoteSummary creates a valid default remission note summary.
func CreateDefaultRemissionNoteSummary() *structs.RemissionNoteSummaryRequest {
	nonSubjectTotal := 0.0
	exemptTotal := 0.0
	taxedTotal := 200.0
	subtotalSales := 200.0
	nonSubjectDiscount := 0.0
	exemptDiscount := 0.0
	taxedDiscount := 0.0
	totalDiscount := 0.0
	subtotal := 200.0
	totalAmount := 226.0
	amountWords := "DOSCIENTOS VEINTISEIS DOLARES CON 00/100"
	return &structs.RemissionNoteSummaryRequest{
		NonSubjectTotal:    &nonSubjectTotal,
		ExemptTotal:        &exemptTotal,
		TaxedTotal:         &taxedTotal,
		SubtotalSales:      &subtotalSales,
		NonSubjectDiscount: &nonSubjectDiscount,
		ExemptDiscount:     &exemptDiscount,
		TaxedDiscount:      &taxedDiscount,
		TotalDiscount:      &totalDiscount,
		Subtotal:           &subtotal,
		TotalAmount:        &totalAmount,
		AmountInWords:      &amountWords,
		Tributes: []*structs.TaxRequest{
			{Code: "20", Description: "IVA", Value: 26.0},
		},
		Payments: []*structs.PaymentRequest{
			{Code: "01", Amount: 226.0},
		},
	}
}

// CreateDefaultRemissionNoteReceiver creates a valid default remission note receiver.
func CreateDefaultRemissionNoteReceiver() *structs.RemissionNoteReceiverRequest {
	bienTitulo := "BM"
	receiver := CreateDefaultReceiver()
	receiver.NIT = utils.ToStringPointer("06141804941035")
	return &structs.RemissionNoteReceiverRequest{
		ReceiverRequest: receiver,
		BienTitulo:      &bienTitulo,
	}
}

// CreateDefaultRemissionNoteRequest creates a valid default remission note request.
func CreateDefaultRemissionNoteRequest() *structs.CreateRemissionNoteRequest {
	items := []*structs.RemissionNoteItemRequest{
		CreateDefaultRemissionNoteItem(1),
		CreateDefaultRemissionNoteItem(2),
	}
	return &structs.CreateRemissionNoteRequest{
		Items:    items,
		Receiver: CreateDefaultRemissionNoteReceiver(),
		Summary:  CreateDefaultRemissionNoteSummary(),
	}
}

// CreateRemissionNoteRequestWithAllOptionalFields creates a remission note request with all optional fields populated.
func CreateRemissionNoteRequestWithAllOptionalFields() *structs.CreateRemissionNoteRequest {
	req := CreateDefaultRemissionNoteRequest()
	req.ThirdPartySale = CreateDefaultThirdPartySale()
	relatedDoc := CreateDefaultRelatedDocument()
	req.RelatedDocs = []*structs.RelatedDocRequest{&relatedDoc}
	appendix := CreateDefaultAppendix()
	req.Appendixes = []*structs.AppendixRequest{&appendix}
	return req
}
