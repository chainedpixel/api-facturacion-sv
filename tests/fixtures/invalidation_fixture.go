package fixtures

import (
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
)

// CreateDefaultReasonRequest creates a valid default invalidation reason
func CreateDefaultReasonRequest() *structs.ReasonRequest {
	reason := "Factura con datos incorrectos"

	return &structs.ReasonRequest{
		Type:               1,
		ResponsibleName:    "Juan Responsable",
		ResponsibleDocType: "13",
		ResponsibleNumDoc:  "12345678-9",
		RequestorName:      "Ana Solicitante",
		RequestorDocType:   "13",
		RequestorNumDoc:    "98765432-1",
		Reason:             &reason,
	}
}

// CreateDefaultInvalidationRequest creates a valid default invalidation request
func CreateDefaultInvalidationRequest() *structs.CreateInvalidationRequest {
	replacementCode := "FF54E9DB-79C3-42CE-B432-EC522C97EFB9"

	return &structs.CreateInvalidationRequest{
		GenerationCode:            "AD54E9BB-79A3-42AE-B432-EC522C97EFB7",
		Reason:                    CreateDefaultReasonRequest(),
		ReplacementGenerationCode: &replacementCode,
	}
}

// CreateInvalidationWithInvalidType creates an invalidation request with an invalid type
func CreateInvalidationWithInvalidType() *structs.CreateInvalidationRequest {
	req := CreateDefaultInvalidationRequest()
	req.Reason.Type = 99
	return req
}

// CreateInvalidationTypeWithoutReason creates a type 3 invalidation request without a reason
func CreateInvalidationTypeWithoutReason() *structs.CreateInvalidationRequest {
	req := CreateDefaultInvalidationRequest()
	req.Reason.Type = 3
	req.Reason.Reason = nil
	return req
}

// CreateInvalidationType2WithReplacementCode creates a type 2 invalidation request with a replacement code
func CreateInvalidationType2WithReplacementCode() *structs.CreateInvalidationRequest {
	req := CreateDefaultInvalidationRequest()
	req.Reason.Type = 2
	replacementCode := "FF54E9DB-79C3-42CE-B432-EC522C97EFB9"
	req.ReplacementGenerationCode = &replacementCode
	return req
}
