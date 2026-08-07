package fixtures

import (
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
)

// CreateAddressWithEmptyFields creates an address with empty fields
func CreateAddressWithEmptyFields() *structs.AddressRequest {
	return &structs.AddressRequest{
		Department:   "",
		Municipality: "",
		Complement:   "",
	}
}

// CreateAddressWithInvalidMunicipality creates an address with an invalid municipality
func CreateAddressWithInvalidMunicipality() *structs.AddressRequest {
	address := CreateDefaultAddress()
	address.Municipality = "99"
	return address
}

// CreateReceiverWithInvalidEmail creates a receiver with an invalid email
func CreateReceiverWithInvalidEmail() *structs.ReceiverRequest {
	receiver := CreateDefaultReceiver()
	invalidEmail := "not-an-email"
	receiver.Email = &invalidEmail
	return receiver
}

// CreateReceiverWithoutRequiredFields creates a receiver without required fields
func CreateReceiverWithoutRequiredFields() *structs.ReceiverRequest {
	receiver := CreateDefaultReceiver()
	receiver.Name = nil
	receiver.NRC = nil
	return receiver
}

// CreateExtensionWithMissingFields creates an extension with missing fields
func CreateExtensionWithMissingFields() *structs.ExtensionRequest {
	ext := CreateDefaultExtension()
	ext.DeliveryName = ""
	ext.ReceiverName = ""
	return ext
}

// CreateAppendixWithInvalidField creates an appendix with an invalid field
func CreateAppendixWithInvalidField() structs.AppendixRequest {
	appendix := CreateDefaultAppendix()
	appendix.Field = ""
	return appendix
}

// CreatePaymentWithInvalidCode creates a payment with an invalid code
func CreatePaymentWithInvalidCode() structs.PaymentRequest {
	payment := CreateDefaultPayment()
	payment.Code = "100"
	return payment
}

// CreateThirdPartySaleWithEmptyNIT creates a third-party sale with an empty NIT
func CreateThirdPartySaleWithEmptyNIT() *structs.ThirdPartySaleRequest {
	sale := CreateDefaultThirdPartySale()
	sale.NIT = ""
	return sale
}
