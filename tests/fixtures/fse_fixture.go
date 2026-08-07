package fixtures

import (
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

// CreateDefaultFSEItem creates a valid default FSE item.
func CreateDefaultFSEItem(index int) structs.FSEItemRequest {
	code := "FSE" + string(rune(65+index))
	return structs.FSEItemRequest{
		ItemRequest: structs.ItemRequest{
			Number:      index + 1,
			Type:        1,
			Description: "Servicio excluido " + string(rune(65+index)),
			Quantity:    1,
			UnitMeasure: 59,
			UnitPrice:   100.0,
			Discount:    0,
			Code:        &code,
		},
		Purchase: 100.0,
	}
}

// CreateDefaultFSESummary creates a valid default FSE summary.
func CreateDefaultFSESummary() *structs.FSESummaryRequest {
	obs := "Servicio de consultoría"
	return &structs.FSESummaryRequest{
		SummaryRequest: structs.SummaryRequest{
			TotalNonSubject:    0,
			TotalExempt:        0,
			TotalTaxed:         0,
			SubTotal:           200.0,
			NonSubjectDiscount: 0,
			ExemptDiscount:     0,
			DiscountPercentage: 0,
			TotalDiscount:      0,
			SubTotalSales:      200.0,
			TotalOperation:     200.0,
			TotalNonTaxed:      200.0,
			TotalToPay:         200.0,
			OperationCondition: 1,
			Taxes:              []structs.TaxRequest{},
			PaymentTypes:       []structs.PaymentRequest{{Code: "01", Amount: 200.0}},
		},
		TotalPurchase:   200.0,
		IVARetention:    0,
		IncomeRetention: 0,
		Observations:    &obs,
	}
}

// CreateDefaultFSEReceiver creates a valid default FSE excluded-subject receiver.
func CreateDefaultFSEReceiver() *structs.FSEReceiverRequest {
	name := "Juan Carlos Pérez"
	phone := "22345678"
	email := "juan@example.com"
	actCode := "46900"
	actDesc := "Servicios profesionales"
	return &structs.FSEReceiverRequest{
		ReceiverRequest: structs.ReceiverRequest{
			Name:    &name,
			Address: CreateDefaultAddress(),
			Phone:   &phone,
			Email:   &email,
		},
		DocumentType:        "13",
		DocumentNumber:      utils.PointerToString(utils.ToStringPointer("00000000-0")),
		ActivityCode:        &actCode,
		ActivityDescription: &actDesc,
	}
}

// CreateDefaultFSERequest creates a valid default FSE request.
func CreateDefaultFSERequest() *structs.CreateFSERequest {
	items := []structs.FSEItemRequest{
		CreateDefaultFSEItem(1),
		CreateDefaultFSEItem(2),
	}
	return &structs.CreateFSERequest{
		Items:    items,
		Receiver: CreateDefaultFSEReceiver(),
		Summary:  CreateDefaultFSESummary(),
	}
}

// CreateFSERequestWithAllOptionalFields creates an FSE request with all optional fields populated.
func CreateFSERequestWithAllOptionalFields() *structs.CreateFSERequest {
	req := CreateDefaultFSERequest()
	req.Extension = CreateDefaultExtension()
	req.Appendixes = []structs.AppendixRequest{CreateDefaultAppendix()}
	return req
}
