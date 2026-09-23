package fixtures

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
)

// CreatePhysicalDocumentsRetentionRequest creates a default retention request with physical documents
func CreatePhysicalDocumentsRetentionRequest() *structs.CreateRetentionRequest {
	return createRequest(constants.PhysicalDocument)
}

// CreateElectronicDocumentsRetentionRequest creates a default retention request with electronic documents
func CreateElectronicDocumentsRetentionRequest() *structs.CreateRetentionRequest {
	return createRequest(constants.ElectronicDocument)
}

func createRequest(genType int) *structs.CreateRetentionRequest {
	taxedAmount1 := 115.25
	ivaAmount1 := 15.00
	emissionDate1 := "2025-03-20"
	dteType1 := "03"
	docNumber1 := "FF54E9DB-79A3-42CE-B432-EC522C97EFB9"

	taxedAmount2 := 226.50
	ivaAmount2 := 29.43
	emissionDate2 := "2025-03-22"
	dteType2 := "03"
	docNumber2 := "AD54E9BB-79A3-42AE-B432-EC522C97EFB7"

	if genType == constants.PhysicalDocument {
		docNumber1 = "S221001347"
		docNumber2 = "S221001348"
	}

	items := []structs.RetentionItem{
		{
			DocumentType:   genType,
			DocumentNumber: docNumber1,
			Description:    "Compra de equipos informáticos",
			RetentionCode:  "22",
			TaxedAmount:    &taxedAmount1,
			IvaAmount:      &ivaAmount1,
			EmissionDate:   &emissionDate1,
			DTEType:        &dteType1,
		},
		{
			DocumentType:   genType,
			DocumentNumber: docNumber2,
			Description:    "Mantenimiento de servidores",
			RetentionCode:  "C4",
			TaxedAmount:    &taxedAmount2,
			IvaAmount:      &ivaAmount2,
			EmissionDate:   &emissionDate2,
			DTEType:        &dteType2,
		},
	}

	summary := &structs.RetentionSummary{
		TotalRetentionAmount: 341.75,
		TotalRetentionIVA:    44.43,
	}

	docType := "36"
	docNumber := "06141804941035"
	nrc := "123456"
	name := "Empresa Servicios Generales, S.A. de C.V."
	commercialName := "ServiGeneral"
	activityCode := "46900"
	activityDesc := "Venta al por mayor de otros productos"
	phone := "22123456"
	email := "info@gmail.com"

	receiver := &structs.ReceiverRequest{
		DocumentType:   &docType,
		DocumentNumber: &docNumber,
		NRC:            &nrc,
		Name:           &name,
		CommercialName: &commercialName,
		ActivityCode:   &activityCode,
		ActivityDesc:   &activityDesc,
		Address: &structs.AddressRequest{
			Department:   "06",
			Municipality: "20",
			Complement:   "Colonia Escalón, Calle La Reforma #123, San Salvador",
		},
		Phone: &phone,
		Email: &email,
	}

	observation := "Retención por servicios tecnológicos primer trimestre"
	summary.Observations = &observation

	return &structs.CreateRetentionRequest{
		Items:    items,
		Receiver: receiver,
		Summary:  summary,
	}
}

// CreateMixedDocumentsRetentionRequest creates a default retention request with mixed documents
func CreateMixedDocumentsRetentionRequest() *structs.CreateRetentionRequest {
	taxedAmount1 := 450.00
	ivaAmount1 := 58.50
	emissionDate1 := "2025-03-15"
	dteType1 := "03"

	taxedAmount2 := 300.00
	ivaAmount2 := 39.00
	emissionDate2 := "2025-03-18"
	dteType2 := "03"

	items := []structs.RetentionItem{
		{
			DocumentType:   1,
			DocumentNumber: "S221001347",
			Description:    "Consultoría financiera",
			RetentionCode:  "C9",
			TaxedAmount:    &taxedAmount1,
			IvaAmount:      &ivaAmount1,
			EmissionDate:   &emissionDate1,
			DTEType:        &dteType1,
		},
		{
			DocumentType:   2,
			DocumentNumber: "FF32E9DB-79C3-42CE-B432-EC522C97EFB2",
			Description:    "Servicios de auditoría",
			RetentionCode:  "C4",
			TaxedAmount:    &taxedAmount2,
			IvaAmount:      &ivaAmount2,
			EmissionDate:   &emissionDate2,
			DTEType:        &dteType2,
		},
	}

	docType := "36"
	docNumber := "06141804941035"
	nrc := "123456"
	name := "Empresa Servicios Generales, S.A. de C.V."
	commercialName := "ServiGeneral"
	activityCode := "46900"
	activityDesc := "Venta al por mayor de otros productos"
	phone := "22123456"
	email := "info@gmail.com"

	summary := &structs.RetentionSummary{
		TotalRetentionAmount: 750.00,
		TotalRetentionIVA:    97.50,
	}

	receiver := &structs.ReceiverRequest{
		DocumentType:   &docType,
		DocumentNumber: &docNumber,
		NRC:            &nrc,
		Name:           &name,
		CommercialName: &commercialName,
		ActivityCode:   &activityCode,
		ActivityDesc:   &activityDesc,
		Address: &structs.AddressRequest{
			Department:   "06",
			Municipality: "20",
			Complement:   "Colonia Escalón, Calle La Reforma #123, San Salvador",
		},
		Phone: &phone,
		Email: &email,
	}

	appendixes := []structs.AppendixRequest{
		{
			Field: "nota_interna",
			Label: "Nota interna",
			Value: "Retención realizada según contrato marco",
		},
	}

	return &structs.CreateRetentionRequest{
		Items:      items,
		Summary:    summary,
		Receiver:   receiver,
		Appendixes: appendixes,
	}
}
