package fixtures

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/user"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/temporal"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

// CreateDefaultAddress creates a valid default address
func CreateDefaultAddress() *structs.AddressRequest {
	return &structs.AddressRequest{
		Department:   "06",
		Municipality: "20",
		District:     "01",
		Complement:   "Colonia Escalón, Calle La Reforma #123, San Salvador",
	}
}

// CreateDefaultAppendix creates a valid default appendix
func CreateDefaultAppendix() structs.AppendixRequest {
	return structs.AppendixRequest{
		Field: "nota_interna",
		Label: "Nota interna",
		Value: "Información adicional para el documento",
	}
}

// CreateDefaultExtension creates a valid default extension
func CreateDefaultExtension() *structs.ExtensionRequest {
	observation := "Observación de prueba"
	vehiculePlate := "P123-456"

	return &structs.ExtensionRequest{
		DeliveryName:     "Juan Pérez",
		DeliveryDocument: "12345678-9",
		ReceiverName:     "Ana López",
		ReceiverDocument: "98765432-1",
		Observation:      &observation,
		VehiculePlate:    &vehiculePlate,
	}
}

// CreateDefaultReceiver creates a valid default receiver
func CreateDefaultReceiver() *structs.ReceiverRequest {
	docType := "36"
	docNumber := "06141804941035"
	name := "Empresa Servicios Generales, S.A. de C.V."
	nrc := "123456"
	phone := "22123456"
	email := "empresa@example.com"
	activityCode := "46900"
	activityDesc := "Venta al por mayor de otros productos"
	commercialName := "ServiGeneral"

	return &structs.ReceiverRequest{
		DocumentType:   &docType,
		DocumentNumber: &docNumber,
		Name:           &name,
		NRC:            &nrc,
		Address:        CreateDefaultAddress(),
		Phone:          &phone,
		Email:          &email,
		ActivityCode:   &activityCode,
		ActivityDesc:   &activityDesc,
		CommercialName: &commercialName,
	}
}

// CreateDefaultReceiverWithoutDocsFields creates a valid default receiver without document type and document number fields
func CreateDefaultReceiverWithoutDocsFields() *structs.ReceiverRequest {
	name := "Empresa Servicios Generales, S.A. de C.V."
	nrc := "123456"
	nit := "06141804941035"
	phone := "22123456"
	email := "empresa@example.com"
	activityCode := "46900"
	activityDesc := "Venta al por mayor de otros productos"
	commercialName := "ServiGeneral"

	return &structs.ReceiverRequest{
		Name:           &name,
		NRC:            &nrc,
		NIT:            &nit,
		Address:        CreateDefaultAddress(),
		Phone:          &phone,
		Email:          &email,
		ActivityCode:   &activityCode,
		ActivityDesc:   &activityDesc,
		CommercialName: &commercialName,
	}
}

// CreateDefaultPayment creates a valid default payment
func CreateDefaultPayment() structs.PaymentRequest {
	reference := "REF-123"

	return structs.PaymentRequest{
		Code:      "01",
		Amount:    100.0,
		Reference: &reference,
	}
}

// CreateDefaultThirdPartySale creates a valid default third-party sale
func CreateDefaultThirdPartySale() *structs.ThirdPartySaleRequest {
	return &structs.ThirdPartySaleRequest{
		NIT:  "06141804941035",
		Name: "Empresa Tercero S.A. de C.V.",
	}
}

// CreateDefaultRelatedDocument creates a valid default related document
func CreateDefaultRelatedDocument() structs.RelatedDocRequest {
	return structs.RelatedDocRequest{
		DocumentType:   "03",
		GenerationType: 1,
		DocumentNumber: "S221001346",
		EmissionDate:   "2025-03-22",
	}
}

// CreateDefaultIssuer creates a default issuer for use in tests
func CreateDefaultIssuer() *dte.IssuerDTE {
	return &dte.IssuerDTE{
		NIT:                  "11111111111111",
		NRC:                  "1111111",
		CommercialName:       "EJEMPLO SA",
		BusinessName:         "EMPRESA DE PRUEBAS SA DE CV",
		EconomicActivity:     "11111",
		EconomicActivityDesc: "Venta al por mayor de otros productos",
		EstablishmentCode:    utils.ToStringPointer("C001"),
		Email:                utils.ToStringPointer("email@gmail.com"),
		Phone:                utils.ToStringPointer("22567890"),
		Address: &user.Address{
			Department:   "06",
			Municipality: "20",
			District:     "01",
			Complement:   "BOULEVARD SANTA ELENA SUR, SANTA TECLA",
		},
		EstablishmentType:   "02",
		EstablishmentCodeMH: nil,
		POSCode:             nil,
		POSCodeMH:           nil,
	}
}

// CreateCustomIssuer creates a custom issuer for use in tests
func CreateCustomIssuer(nit, nrc, businessName string) *dte.IssuerDTE {
	issuer := CreateDefaultIssuer()
	issuer.NIT = nit
	issuer.NRC = nrc
	issuer.BusinessName = businessName
	return issuer
}

// CreateDefaultOtherDocument creates a valid default additional document
func CreateDefaultOtherDocument() structs.OtherDocRequest {
	description := "Documento adicional"
	detail := "Detalle del documento adicional"

	return structs.OtherDocRequest{
		DocumentCode: 1,
		Description:  &description,
		Detail:       &detail,
	}
}

// CreateIdentification creates an identification with a specific type and version
func CreateIdentification(dteType string, version int) (*models.Identification, error) {
	now := utils.TimeNow()

	versionValue := document.NewValidatedVersion(version)
	dteTypeValue := document.NewValidatedDTEType(dteType)
	currency := financial.NewValidatedCurrency("USD")

	ambient, err := document.NewAmbient()
	if err != nil {
		return nil, err
	}

	emissionDate, err := temporal.NewEmissionDate(now)
	if err != nil {
		return nil, err
	}

	emissionTime, err := temporal.NewEmissionTime(now)
	if err != nil {
		return nil, err
	}

	modelType, err := document.NewModelType(1)
	if err != nil {
		return nil, err
	}

	operationType, err := document.NewOperationType(1)
	if err != nil {
		return nil, err
	}

	return &models.Identification{
		Version:       *versionValue,
		Ambient:       *ambient,
		DTEType:       *dteTypeValue,
		Currency:      *currency,
		OperationType: *operationType,
		ModelType:     *modelType,
		EmissionDate:  *emissionDate,
		EmissionTime:  *emissionTime,
	}, nil
}

// CreateIdentificationWithInvalidVersion creates an identification with an invalid version
func CreateIdentificationWithInvalidVersion(dteType string) (*models.Identification, error) {
	id, err := CreateIdentification(dteType, 1)
	if err != nil {
		return nil, err
	}

	invalidVersion := document.NewValidatedVersion(99)
	id.Version = *invalidVersion

	return id, nil
}

// CreateIdentificationWithInvalidDTEType creates an identification with an invalid DTE type
func CreateIdentificationWithInvalidDTEType() (*models.Identification, error) {
	id, err := CreateIdentification("01", 1)
	if err != nil {
		return nil, err
	}

	invalidType := document.NewValidatedDTEType("99")
	id.DTEType = *invalidType

	return id, nil
}
