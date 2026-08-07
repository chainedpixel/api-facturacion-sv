package fixtures

import (
	"fmt"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/credit_note/credit_note_models"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

// CreditNoteBuilder - Specialized builder for Electronic Credit Note
// that reuses the base DTEBuilder to avoid code duplication
type CreditNoteBuilder struct {
	baseBuilder *DTEBuilder
	document    *credit_note_models.CreditNoteModel
	err         error
}

func NewCreditNoteBuilder() *CreditNoteBuilder {
	return &CreditNoteBuilder{
		baseBuilder: NewDTEBuilder(),
		document: &credit_note_models.CreditNoteModel{
			DTEDocument: &models.DTEDocument{},
			CreditItems: make([]credit_note_models.CreditNoteItem, 0),
		},
		err: nil,
	}
}

func (b *CreditNoteBuilder) Document() *credit_note_models.CreditNoteModel {
	return b.document
}

func (b *CreditNoteBuilder) Build() (*credit_note_models.CreditNoteModel, error) {
	if b.err != nil {
		return nil, b.err
	}

	baseDoc, err := b.baseBuilder.BuildWithoutValidation()
	if err != nil {
		return nil, err
	}

	b.document.DTEDocument = baseDoc

	err = b.document.DTEDocument.Validate()
	if err != nil {
		return nil, err
	}

	dteErr := b.document.DTEDocument.ValidateDTERules()
	if dteErr != nil {
		return nil, dteErr
	}

	return b.document, nil
}

func (b *CreditNoteBuilder) BuildWithoutValidation() (*credit_note_models.CreditNoteModel, error) {
	if b.err != nil {
		return nil, b.err
	}

	baseDoc, err := b.baseBuilder.BuildWithoutValidation()
	if err != nil {
		return nil, err
	}

	b.document.DTEDocument = baseDoc

	return b.document, nil
}

func (b *CreditNoteBuilder) setError(err error) *CreditNoteBuilder {
	if b.err == nil && err != nil {
		b.err = err
	}

	if err != nil {
		b.baseBuilder.setError(err)
	}

	return b
}

func (b *CreditNoteBuilder) AddIdentification() *CreditNoteBuilder {
	b.baseBuilder.AddIdentification()

	if b.baseBuilder.err != nil {
		b.err = b.baseBuilder.err
		return b
	}

	baseIdentification, ok := b.baseBuilder.document.GetIdentification().(*models.Identification)
	if !ok || baseIdentification == nil {
		b.setError(fmt.Errorf("failed to get identification from base builder"))
		return b
	}

	b.setError(baseIdentification.SetDTEType(constants.NotaCreditoElectronica))

	controlNumber := baseIdentification.GetControlNumber()
	if len(controlNumber) > 4 {
		newControlNumber := "DTE-04" + controlNumber[6:]
		b.setError(baseIdentification.SetControlNumber(newControlNumber))
	}

	return b
}

func (b *CreditNoteBuilder) AddIssuer() *CreditNoteBuilder {
	b.baseBuilder.AddIssuer()

	if b.baseBuilder.err != nil {
		b.err = b.baseBuilder.err
	}

	return b
}

func (b *CreditNoteBuilder) AddReceiver() *CreditNoteBuilder {
	b.baseBuilder.AddReceiver()

	if b.baseBuilder.err != nil {
		b.err = b.baseBuilder.err
	}

	return b
}

func (b *CreditNoteBuilder) AddReceiverForCompany() *CreditNoteBuilder {
	b.baseBuilder.AddReceiverForCCF()

	if b.baseBuilder.err != nil {
		b.err = b.baseBuilder.err
		return b
	}

	baseReceiver, ok := b.baseBuilder.document.GetReceiver().(*models.Receiver)
	if !ok || baseReceiver == nil {
		b.setError(fmt.Errorf("failed to get receiver from base builder"))
		return b
	}

	name := "EMPRESA CLIENTE, S.A. DE C.V."
	nrc := "1234567"
	nit := "06140101901011"
	docType := constants.NIT
	activityCode := "12345"
	activityDescription := "Compra de bienes y servicios"
	commercialName := "ENTERPRISE CORP"

	b.setError(baseReceiver.SetName(&name))
	b.setError(baseReceiver.SetDocumentType(&docType))
	b.setError(baseReceiver.SetDocumentNumber(&nit))
	b.setError(baseReceiver.SetNRC(&nrc))
	b.setError(baseReceiver.SetNIT(&nit))
	b.setError(baseReceiver.SetActivityCode(&activityCode))
	b.setError(baseReceiver.SetActivityDescription(&activityDescription))
	b.setError(baseReceiver.SetCommercialName(&commercialName))

	return b
}

func (b *CreditNoteBuilder) AddItems() *CreditNoteBuilder {
	b.baseBuilder.AddItems()

	if b.baseBuilder.err != nil {
		b.err = b.baseBuilder.err
		return b
	}

	baseItems := b.baseBuilder.document.GetItems()
	creditItems := make([]credit_note_models.CreditNoteItem, 0, len(baseItems))

	for _, baseItem := range baseItems {
		item, ok := baseItem.(*models.Item)
		if !ok {
			b.setError(fmt.Errorf("failed to convert item to proper type"))
			return b
		}

		item.Taxes = []string{constants.TaxIVA}
		creditItem := credit_note_models.CreditNoteItem{
			Item: item,
		}

		taxedAmount := item.GetQuantity() * item.GetUnitPrice() * (1 - item.GetDiscount()/100)

		taxedSaleObj, err := financial.NewAmount(taxedAmount)
		if err != nil {
			b.setError(err)
			return b
		}
		creditItem.TaxedSale = *taxedSaleObj

		zeroAmount, err := financial.NewAmount(0.0)
		if err != nil {
			b.setError(err)
			return b
		}
		creditItem.NonSubjectSale = *zeroAmount
		creditItem.ExemptSale = *zeroAmount
		creditItems = append(creditItems, creditItem)
	}

	b.document.CreditItems = creditItems

	return b
}

func (b *CreditNoteBuilder) AddSummary() *CreditNoteBuilder {
	b.baseBuilder.AddSummary()

	if b.baseBuilder.err != nil {
		b.err = b.baseBuilder.err
		return b
	}

	baseSummary, ok := b.baseBuilder.document.GetSummary().(*models.Summary)
	if !ok || baseSummary == nil {
		b.setError(fmt.Errorf("failed to get summary from base builder"))
		return b
	}

	creditSummary := credit_note_models.CreditNoteSummary{
		Summary: baseSummary,
	}

	zeroAmount, err := financial.NewAmount(0.0)
	if err != nil {
		b.setError(err)
		return b
	}

	creditSummary.TaxedDiscount = *zeroAmount
	creditSummary.IVAPerception = *zeroAmount
	creditSummary.IVARetention = *zeroAmount
	creditSummary.IncomeRetention = *zeroAmount

	b.document.CreditSummary = creditSummary

	return b
}

func (b *CreditNoteBuilder) AddSummaryWithCreditOperation() *CreditNoteBuilder {
	b.AddSummary()

	if b.baseBuilder.err != nil {
		b.err = b.baseBuilder.err
		return b
	}

	baseSummary, ok := b.baseBuilder.document.GetSummary().(*models.Summary)
	if !ok || baseSummary == nil {
		b.setError(fmt.Errorf("failed to get summary from base builder"))
		return b
	}

	payments := baseSummary.GetPaymentTypes()
	for _, payment := range payments {
		if payment.GetCode() == constants.BilletesMonedas {
			period := 2
			term := "02"

			b.setError(payment.SetCode(constants.TarjetaCredito))
			b.setError(payment.SetPeriod(&period))
			b.setError(payment.SetTerm(&term))
		}
	}

	b.setError(baseSummary.SetPaymentTypes(payments))
	b.setError(baseSummary.SetOperationCondition(constants.Credit))

	return b
}

func (b *CreditNoteBuilder) AddRelatedDocuments() *CreditNoteBuilder {
	relatedDoc := &models.RelatedDocument{}

	b.setError(relatedDoc.SetDocumentType(constants.CCFElectronico))
	b.setError(relatedDoc.SetGenerationType(constants.ElectronicDocument))
	b.setError(relatedDoc.SetDocumentNumber("0ACAD9C9-81B0-4D9B-98A7-E85387673875"))
	b.setError(relatedDoc.SetEmissionDate(utils.TimeNow().AddDate(0, 0, -1)))

	if b.err == nil {
		relatedDocs := []interfaces.RelatedDocument{relatedDoc}
		b.setError(b.baseBuilder.document.SetRelatedDocuments(relatedDocs))
	}

	for _, item := range b.document.CreditItems {
		docRef := "0ACAD9C9-81B0-4D9B-98A7-E85387673875"
		b.setError(item.SetRelatedDoc(&docRef))
	}

	return b
}

func BuildValidCreditNote() (*credit_note_models.CreditNoteModel, error) {
	builder := NewCreditNoteBuilder()

	builder.AddIdentification().
		AddIssuer().
		AddReceiverForCompany().
		AddItems().
		AddSummary().
		AddRelatedDocuments()

	return builder.Build()
}

func BuildCreditNoteWithCompanyReceiver() (*credit_note_models.CreditNoteModel, error) {
	builder := NewCreditNoteBuilder()

	builder.AddIdentification().
		AddIssuer().
		AddReceiverForCompany().
		AddItems().
		AddSummary().
		AddRelatedDocuments()

	return builder.Build()
}

func BuildCreditNoteWithCreditOperation() (*credit_note_models.CreditNoteModel, error) {
	builder := NewCreditNoteBuilder()

	builder.AddIdentification().
		AddIssuer().
		AddReceiverForCompany().
		AddItems().
		AddSummaryWithCreditOperation().
		AddRelatedDocuments()

	return builder.Build()
}

func (b *CreditNoteBuilder) BuildAsCreditNoteInput() (*credit_note_models.CreditNoteInput, error) {
	if b == nil {
		return nil, fmt.Errorf("CreditNoteBuilder is nil")
	}

	if b.err != nil {
		return nil, b.err
	}

	creditNote := b.document
	var appendixes []models.Appendix
	for _, app := range creditNote.Appendix {
		if a, ok := app.(*models.Appendix); ok {
			appendixes = append(appendixes, *a)
		}
	}

	var relatedDocs []models.RelatedDocument
	for _, rd := range creditNote.RelatedDocuments {
		if r, ok := rd.(*models.RelatedDocument); ok {
			relatedDocs = append(relatedDocs, *r)
		}
	}

	var otherDocs []models.OtherDocument
	for _, od := range creditNote.OtherDocuments {
		if o, ok := od.(*models.OtherDocument); ok {
			otherDocs = append(otherDocs, *o)
		}
	}

	var thirdPartySale *models.ThirdPartySale
	if creditNote.ThirdPartySale != nil {
		if t, ok := creditNote.ThirdPartySale.(*models.ThirdPartySale); ok {
			thirdPartySale = t
		}
	}

	var extension *models.Extension
	if creditNote.Extension != nil {
		if e, ok := creditNote.Extension.(*models.Extension); ok {
			extension = e
		}
	}

	return &credit_note_models.CreditNoteInput{
		InputDataCommon: &models.InputDataCommon{
			Identification: creditNote.GetIdentification().(*models.Identification),
			Issuer:         creditNote.GetIssuer().(*models.Issuer),
			Receiver:       creditNote.GetReceiver().(*models.Receiver),
			Extension:      extension,
			RelatedDocs:    relatedDocs,
			OtherDocs:      otherDocs,
			ThirdPartySale: thirdPartySale,
			Appendixes:     appendixes,
		},
		Items:         creditNote.CreditItems,
		CreditSummary: &creditNote.CreditSummary,
	}, nil
}

func BuildAsCreditNoteInput(creditNote *credit_note_models.CreditNoteModel) *credit_note_models.CreditNoteInput {
	var appendixes []models.Appendix
	for _, app := range creditNote.Appendix {
		if a, ok := app.(*models.Appendix); ok {
			appendixes = append(appendixes, *a)
		}
	}

	var relatedDocs []models.RelatedDocument
	for _, rd := range creditNote.RelatedDocuments {
		if r, ok := rd.(*models.RelatedDocument); ok {
			relatedDocs = append(relatedDocs, *r)
		}
	}

	var otherDocs []models.OtherDocument
	for _, od := range creditNote.OtherDocuments {
		if o, ok := od.(*models.OtherDocument); ok {
			otherDocs = append(otherDocs, *o)
		}
	}

	var thirdPartySale *models.ThirdPartySale
	if creditNote.ThirdPartySale != nil {
		if t, ok := creditNote.ThirdPartySale.(*models.ThirdPartySale); ok {
			thirdPartySale = t
		}
	}

	var extension *models.Extension
	if creditNote.Extension != nil {
		if e, ok := creditNote.Extension.(*models.Extension); ok {
			extension = e
		}
	}

	return &credit_note_models.CreditNoteInput{
		InputDataCommon: &models.InputDataCommon{
			Identification: creditNote.GetIdentification().(*models.Identification),
			Issuer:         creditNote.GetIssuer().(*models.Issuer),
			Receiver:       creditNote.GetReceiver().(*models.Receiver),
			Extension:      extension,
			RelatedDocs:    relatedDocs,
			OtherDocs:      otherDocs,
			ThirdPartySale: thirdPartySale,
			Appendixes:     appendixes,
		},
		Items:         creditNote.CreditItems,
		CreditSummary: &creditNote.CreditSummary,
	}
}

func BuildCreditNote() (*credit_note_models.CreditNoteModel, error) {
	return BuildValidCreditNote()
}
