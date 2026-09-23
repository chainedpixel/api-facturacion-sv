package fixtures

import (
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/base"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/identification"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/temporal"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invalidation/invalidation_models"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

// InvalidationBuilder - Specialized builder for invalidation documents
type InvalidationBuilder struct {
	document *invalidation_models.InvalidationDocument
	err      error
}

func NewInvalidationBuilder() *InvalidationBuilder {
	return &InvalidationBuilder{
		document: &invalidation_models.InvalidationDocument{
			Identification: &models.Identification{},
			Issuer:         &models.Issuer{},
			Document:       &invalidation_models.InvalidatedDocument{},
			Reason:         &invalidation_models.InvalidationReason{},
		},
		err: nil,
	}
}

func (b *InvalidationBuilder) Document() *invalidation_models.InvalidationDocument {
	return b.document
}

func (b *InvalidationBuilder) Build() (*invalidation_models.InvalidationDocument, error) {
	if b.err != nil {
		return nil, b.err
	}

	return b.document, nil
}

func (b *InvalidationBuilder) BuildWithoutValidation() (*invalidation_models.InvalidationDocument, error) {
	if b.err != nil {
		return nil, b.err
	}

	return b.document, nil
}

func (b *InvalidationBuilder) setError(err error) *InvalidationBuilder {
	if b.err == nil && err != nil {
		b.err = err
	}
	return b
}

func (b *InvalidationBuilder) AddIdentification() *InvalidationBuilder {
	if b.err != nil {
		return b
	}

	b.setError(b.document.Identification.SetVersion(1))
	b.setError(b.document.Identification.SetAmbient(constants.Testing))
	b.setError(b.document.Identification.SetDTEType(constants.FacturaElectronica))
	b.setError(b.document.Identification.SetControlNumber("DTE-05-00000001-000000000000001"))
	b.setError(b.document.Identification.GenerateCode())
	b.setError(b.document.Identification.SetModelType(constants.ModeloFacturacionPrevio))
	b.setError(b.document.Identification.SetOperationType(1))
	b.setError(b.document.Identification.SetEmissionDate(utils.TimeNow()))
	b.setError(b.document.Identification.SetEmissionTime(utils.TimeNow()))
	b.setError(b.document.Identification.SetCurrency("USD"))

	return b
}

func (b *InvalidationBuilder) AddIssuer() *InvalidationBuilder {
	if b.err != nil {
		return b
	}

	b.setError(b.document.Issuer.SetNIT("12345678901234"))
	b.setError(b.document.Issuer.SetNRC("12345678"))
	b.setError(b.document.Issuer.SetName("EMPRESA EMISORA, S.A. DE C.V."))
	b.setError(b.document.Issuer.SetActivityCode("12345"))
	b.setError(b.document.Issuer.SetActivityDescription("Venta de productos electrónicos"))
	b.setError(b.document.Issuer.SetEstablishmentType(constants.CasaMatriz))

	address := &models.Address{}
	b.setError(address.SetDepartment("06"))
	b.setError(address.SetMunicipality("21"))
	b.setError(address.SetDistrict("01"))
	b.setError(address.SetComplement("Calle Principal, Edificio Central #123"))

	if b.err == nil {
		b.setError(b.document.Issuer.SetAddress(address))
	}

	b.setError(b.document.Issuer.SetPhone("22225555"))
	b.setError(b.document.Issuer.SetEmail("info@google.com"))
	b.setError(b.document.Issuer.SetCommercialName("EMPRESA TECH"))

	establishmentCode := "001"
	establishmentMHCode := "EST001"
	posCode := "POS01"
	posMHCode := "POS001"
	b.setError(b.document.Issuer.SetEstablishmentCode(&establishmentCode))
	b.setError(b.document.Issuer.SetEstablishmentMHCode(&establishmentMHCode))
	b.setError(b.document.Issuer.SetPOSCode(&posCode))
	b.setError(b.document.Issuer.SetPOSMHCode(&posMHCode))

	return b
}

func (b *InvalidationBuilder) AddInvalidatedDocument() *InvalidationBuilder {
	if b.err != nil {
		return b
	}

	docType, err := document.NewDTEType(constants.FacturaElectronica)
	if err != nil {
		b.setError(err)
		return b
	}
	b.document.Document.Type = *docType

	generationCode, err := identification.NewGenerationCode()
	if err != nil {
		b.setError(err)
		return b
	}
	b.document.Document.GenerationCode = *generationCode

	controlNumber, err := identification.NewControlNumber("DTE-01-00000000-000000000000001")
	if err != nil {
		b.setError(err)
		return b
	}
	b.document.Document.ControlNumber = *controlNumber

	b.document.Document.ReceptionStamp = "2025AAFEEE1A566A44F19A622C0C35C8A1B6FAZM"

	emissionDate, err := temporal.NewEmissionDate(time.Now().Add(-24 * time.Hour))
	if err != nil {
		b.setError(err)
		return b
	}
	b.document.Document.EmissionDate = *emissionDate

	ivaAmount, err := financial.NewAmount(13.00)
	if err != nil {
		b.setError(err)
		return b
	}
	b.document.Document.IVAAmount = ivaAmount

	docTypeReceiver, err := document.NewDTEType(constants.FacturaElectronica)
	if err != nil {
		b.setError(err)
		return b
	}
	b.document.Document.DocumentType = docTypeReceiver

	docNumber, err := identification.NewDocumentNumber("01234567-8", constants.DUI)
	if err != nil {
		b.setError(err)
		return b
	}
	b.document.Document.DocumentNumber = docNumber

	name := "CLIENTE EJEMPLO"
	b.document.Document.Name = &name

	email, err := base.NewEmail("cliente@ejemplo.com")
	if err != nil {
		b.setError(err)
		return b
	}
	b.document.Document.Email = email

	phone, err := base.NewPhone("77778888")
	if err != nil {
		b.setError(err)
		return b
	}
	b.document.Document.Phone = phone

	return b
}

func (b *InvalidationBuilder) AddReplacementCode() *InvalidationBuilder {
	if b.err != nil {
		return b
	}

	replacementCode, err := identification.NewGenerationCode()
	if err != nil {
		b.setError(err)
		return b
	}
	b.document.Document.ReplacementCode = replacementCode

	return b
}

func (b *InvalidationBuilder) AddInvalidationReason(invalidationType int) *InvalidationBuilder {
	if b.err != nil {
		return b
	}

	invalidationTypeObj, err := document.NewInvalidationType(invalidationType)
	if err != nil {
		b.setError(err)
		return b
	}
	b.document.Reason.Type = *invalidationTypeObj

	b.document.Reason.ResponsibleName = "JUAN RESPONSABLE"

	responsibleDocType, err := document.NewDTETypeForReceiver(constants.DUI)
	if err != nil {
		b.setError(err)
		return b
	}
	b.document.Reason.ResponsibleDocType = *responsibleDocType

	responsibleDocNum, err := identification.NewDocumentNumber("01234567-8", constants.DUI)
	if err != nil {
		b.setError(err)
		return b
	}
	b.document.Reason.ResponsibleDocNum = *responsibleDocNum

	b.document.Reason.RequesterName = "ANA SOLICITANTE"

	requesterDocType, err := document.NewDTETypeForReceiver(constants.DUI)
	if err != nil {
		b.setError(err)
		return b
	}
	b.document.Reason.RequesterDocType = *requesterDocType

	requesterDocNum, err := identification.NewDocumentNumber("98765432-1", constants.DUI)
	if err != nil {
		b.setError(err)
		return b
	}
	b.document.Reason.RequesterDocNum = *requesterDocNum

	if invalidationType == 3 {
		reasonText := "Documento con errores graves que impiden su utilización"
		invalidReason, err := document.NewInvalidationReason(reasonText)
		if err != nil {
			b.setError(err)
			return b
		}
		b.document.Reason.Reason = invalidReason
	} else {
		b.document.Reason.Reason = nil
	}

	return b
}

// BuildInvalidationWithReplacement builds a type 1 invalidation document (with replacement)
func BuildInvalidationWithReplacement() (*invalidation_models.InvalidationDocument, error) {
	builder := NewInvalidationBuilder()

	builder.AddIdentification().
		AddIssuer().
		AddInvalidatedDocument().
		AddReplacementCode().
		AddInvalidationReason(1)

	return builder.Build()
}

// BuildInvalidationWithAnnulment builds a type 2 invalidation document (annulment)
func BuildInvalidationWithAnnulment() (*invalidation_models.InvalidationDocument, error) {
	builder := NewInvalidationBuilder()

	builder.AddIdentification().
		AddIssuer().
		AddInvalidatedDocument().
		AddInvalidationReason(2)

	return builder.Build()
}

// BuildInvalidationDefinitive builds a type 3 invalidation document (definitive)
func BuildInvalidationDefinitive() (*invalidation_models.InvalidationDocument, error) {
	builder := NewInvalidationBuilder()

	builder.AddIdentification().
		AddIssuer().
		AddInvalidatedDocument().
		AddReplacementCode().
		AddInvalidationReason(3)

	return builder.Build()
}

// BuildInvalidInvalidation builds an invalid invalidation document (type 2 with replacement code)
func BuildInvalidInvalidation() (*invalidation_models.InvalidationDocument, error) {
	builder := NewInvalidationBuilder()

	builder.AddIdentification().
		AddIssuer().
		AddInvalidatedDocument().
		AddReplacementCode().
		AddInvalidationReason(2)

	return builder.BuildWithoutValidation()
}

// BuildInvalidationWithInvalidReason builds an invalid invalidation document due to having a reason for type 1/2
func BuildInvalidationWithInvalidReason() (*invalidation_models.InvalidationDocument, error) {
	builder := NewInvalidationBuilder()

	builder.AddIdentification().
		AddIssuer().
		AddInvalidatedDocument().
		AddInvalidationReason(1)

	reasonText := "Esta razón no debería existir para tipo 1"
	invalidReason, _ := document.NewInvalidationReason(reasonText)
	builder.document.Reason.Reason = invalidReason

	return builder.BuildWithoutValidation()
}

// BuildInvalidationWithMissingReason builds a type 3 invalidation without a reason (invalid)
func BuildInvalidationWithMissingReason() (*invalidation_models.InvalidationDocument, error) {
	builder := NewInvalidationBuilder()

	builder.AddIdentification().
		AddIssuer().
		AddInvalidatedDocument().
		AddReplacementCode()

	invalidationType, _ := document.NewInvalidationType(3)
	builder.document.Reason.Type = *invalidationType

	builder.document.Reason.ResponsibleName = "JUAN RESPONSABLE"

	responsibleDocType, _ := document.NewDTETypeForReceiver(constants.DUI)
	builder.document.Reason.ResponsibleDocType = *responsibleDocType

	responsibleDocNum, _ := identification.NewDocumentNumber("01234567-8", constants.DUI)
	builder.document.Reason.ResponsibleDocNum = *responsibleDocNum

	builder.document.Reason.RequesterName = "ANA SOLICITANTE"

	requesterDocType, _ := document.NewDTETypeForReceiver(constants.DUI)
	builder.document.Reason.RequesterDocType = *requesterDocType

	requesterDocNum, _ := identification.NewDocumentNumber("98765432-1", constants.DUI)
	builder.document.Reason.RequesterDocNum = *requesterDocNum

	builder.document.Reason.Reason = nil

	return builder.BuildWithoutValidation()
}

// BuildInvalidation is a generic method to build a valid invalidation
func BuildInvalidation() (*invalidation_models.InvalidationDocument, error) {
	return BuildInvalidationWithReplacement()
}
