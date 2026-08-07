package fixtures

import (
	"fmt"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/debit_note/debit_note_models"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

// DebitNoteBuilder - Specialized builder for Electronic Debit Note
type DebitNoteBuilder struct {
	baseBuilder *DTEBuilder
	document    *debit_note_models.DebitNoteModel
	err         error
}

func NewDebitNoteBuilder() *DebitNoteBuilder {
	return &DebitNoteBuilder{
		baseBuilder: NewDTEBuilder(),
		document: &debit_note_models.DebitNoteModel{
			DTEDocument: &models.DTEDocument{},
			DebitItems:  make([]debit_note_models.DebitNoteItem, 0),
		},
		err: nil,
	}
}

func (b *DebitNoteBuilder) Document() *debit_note_models.DebitNoteModel {
	return b.document
}

func (b *DebitNoteBuilder) Build() (*debit_note_models.DebitNoteModel, error) {
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

func (b *DebitNoteBuilder) BuildWithoutValidation() (*debit_note_models.DebitNoteModel, error) {
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

func (b *DebitNoteBuilder) setError(err error) *DebitNoteBuilder {
	if b.err == nil && err != nil {
		b.err = err
	}
	if err != nil {
		b.baseBuilder.setError(err)
	}
	return b
}

func (b *DebitNoteBuilder) AddIdentification() *DebitNoteBuilder {
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
	b.setError(baseIdentification.SetDTEType(constants.NotaDebitoElectronica))
	controlNumber := baseIdentification.GetControlNumber()
	if len(controlNumber) > 4 {
		newControlNumber := "DTE-05" + controlNumber[6:]
		b.setError(baseIdentification.SetControlNumber(newControlNumber))
	}
	return b
}

func (b *DebitNoteBuilder) AddIssuer() *DebitNoteBuilder {
	b.baseBuilder.AddIssuer()
	if b.baseBuilder.err != nil {
		b.err = b.baseBuilder.err
	}
	return b
}

func (b *DebitNoteBuilder) AddReceiver() *DebitNoteBuilder {
	b.baseBuilder.AddReceiver()
	if b.baseBuilder.err != nil {
		b.err = b.baseBuilder.err
	}
	return b
}

func (b *DebitNoteBuilder) AddReceiverForCompany() *DebitNoteBuilder {
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

func (b *DebitNoteBuilder) AddItems() *DebitNoteBuilder {
	b.baseBuilder.AddItems()
	if b.baseBuilder.err != nil {
		b.err = b.baseBuilder.err
		return b
	}
	relatedDocNumber := "0ACAD9C9-81B0-4D9B-98A7-E85387673875"
	baseItems := b.baseBuilder.document.GetItems()
	debitItems := make([]debit_note_models.DebitNoteItem, 0, len(baseItems))
	for _, baseItem := range baseItems {
		item, ok := baseItem.(*models.Item)
		if !ok {
			b.setError(fmt.Errorf("failed to convert item to proper type"))
			return b
		}
		item.Taxes = []string{constants.TaxIVA}
		b.setError(item.SetRelatedDoc(&relatedDocNumber))
		debitItem := debit_note_models.DebitNoteItem{Item: item}
		taxedAmount := item.GetQuantity() * item.GetUnitPrice() * (1 - item.GetDiscount()/100)
		taxedSaleObj, err := financial.NewAmount(taxedAmount)
		if err != nil {
			b.setError(err)
			return b
		}
		debitItem.TaxedSale = *taxedSaleObj
		zeroAmount, err := financial.NewAmount(0.0)
		if err != nil {
			b.setError(err)
			return b
		}
		debitItem.NonSubjectSale = *zeroAmount
		debitItem.ExemptSale = *zeroAmount
		debitItems = append(debitItems, debitItem)
	}
	b.document.DebitItems = debitItems
	return b
}

func (b *DebitNoteBuilder) AddSummary() *DebitNoteBuilder {
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
	debitSummary := debit_note_models.DebitNoteSummary{Summary: baseSummary}
	zeroAmount, err := financial.NewAmount(0.0)
	if err != nil {
		b.setError(err)
		return b
	}
	debitSummary.TaxedDiscount = *zeroAmount
	debitSummary.IVAPerception = *zeroAmount
	debitSummary.IVARetention = *zeroAmount
	debitSummary.IncomeRetention = *zeroAmount
	b.document.DebitSummary = debitSummary
	return b
}

func (b *DebitNoteBuilder) AddRelatedDocuments() *DebitNoteBuilder {
	relatedDoc := &models.RelatedDocument{}
	b.setError(relatedDoc.SetDocumentType(constants.CCFElectronico))
	b.setError(relatedDoc.SetGenerationType(constants.ElectronicDocument))
	b.setError(relatedDoc.SetDocumentNumber("0ACAD9C9-81B0-4D9B-98A7-E85387673875"))
	b.setError(relatedDoc.SetEmissionDate(utils.TimeNow().AddDate(0, 0, -1)))
	if b.err == nil {
		relatedDocs := []interfaces.RelatedDocument{relatedDoc}
		b.setError(b.baseBuilder.document.SetRelatedDocuments(relatedDocs))
	}
	return b
}

// BuildAsDebitNoteInput converts a DebitNoteModel to DebitNoteInput for service testing.
func BuildAsDebitNoteInput(debitNote *debit_note_models.DebitNoteModel) *debit_note_models.DebitNoteInput {
	var relatedDocs []models.RelatedDocument
	for _, rd := range debitNote.RelatedDocuments {
		if r, ok := rd.(*models.RelatedDocument); ok {
			relatedDocs = append(relatedDocs, *r)
		}
	}
	var extension *models.Extension
	if debitNote.Extension != nil {
		if e, ok := debitNote.Extension.(*models.Extension); ok {
			extension = e
		}
	}
	return &debit_note_models.DebitNoteInput{
		InputDataCommon: &models.InputDataCommon{
			Identification: debitNote.GetIdentification().(*models.Identification),
			Issuer:         debitNote.GetIssuer().(*models.Issuer),
			Receiver:       debitNote.GetReceiver().(*models.Receiver),
			Extension:      extension,
			RelatedDocs:    relatedDocs,
		},
		Items:        debitNote.DebitItems,
		DebitSummary: &debitNote.DebitSummary,
	}
}

func BuildValidDebitNote() (*debit_note_models.DebitNoteModel, error) {
	builder := NewDebitNoteBuilder()
	builder.AddIdentification().
		AddIssuer().
		AddReceiverForCompany().
		AddItems().
		AddSummary().
		AddRelatedDocuments()
	return builder.Build()
}
