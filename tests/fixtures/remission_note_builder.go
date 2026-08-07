package fixtures

import (
	"fmt"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/remission_note/remission_note_models"
)

// RemissionNoteBuilder - Specialized builder for Electronic Remission Note
type RemissionNoteBuilder struct {
	baseBuilder       *DTEBuilder
	document          *remission_note_models.RemissionNoteModel
	remissionReceiver *remission_note_models.RemissionNoteReceiver
	err               error
}

func NewRemissionNoteBuilder() *RemissionNoteBuilder {
	return &RemissionNoteBuilder{
		baseBuilder: NewDTEBuilder(),
		document: &remission_note_models.RemissionNoteModel{
			DTEDocument:    &models.DTEDocument{},
			RemissionItems: make([]remission_note_models.RemissionNoteItem, 0),
		},
		err: nil,
	}
}

func (b *RemissionNoteBuilder) Document() *remission_note_models.RemissionNoteModel {
	return b.document
}

func (b *RemissionNoteBuilder) Build() (*remission_note_models.RemissionNoteModel, error) {
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

func (b *RemissionNoteBuilder) BuildWithoutValidation() (*remission_note_models.RemissionNoteModel, error) {
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

func (b *RemissionNoteBuilder) setError(err error) *RemissionNoteBuilder {
	if b.err == nil && err != nil {
		b.err = err
	}
	if err != nil {
		b.baseBuilder.setError(err)
	}
	return b
}

func (b *RemissionNoteBuilder) AddIdentification() *RemissionNoteBuilder {
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
	b.setError(baseIdentification.SetDTEType(constants.NotaRemisionElectronica))
	controlNumber := baseIdentification.GetControlNumber()
	if len(controlNumber) > 4 {
		newControlNumber := "DTE-04" + controlNumber[6:]
		b.setError(baseIdentification.SetControlNumber(newControlNumber))
	}
	return b
}

func (b *RemissionNoteBuilder) AddIssuer() *RemissionNoteBuilder {
	b.baseBuilder.AddIssuer()
	if b.baseBuilder.err != nil {
		b.err = b.baseBuilder.err
	}
	return b
}

func (b *RemissionNoteBuilder) AddReceiver() *RemissionNoteBuilder {
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
	nit := "06141804941035"
	b.setError(baseReceiver.SetNIT(&nit))
	bienTitulo := "01"
	b.remissionReceiver = &remission_note_models.RemissionNoteReceiver{
		Receiver:   baseReceiver,
		BienTitulo: &bienTitulo,
	}
	return b
}

func (b *RemissionNoteBuilder) AddItems() *RemissionNoteBuilder {
	b.baseBuilder.AddItems()
	if b.baseBuilder.err != nil {
		b.err = b.baseBuilder.err
		return b
	}
	baseItems := b.baseBuilder.document.GetItems()
	remissionItems := make([]remission_note_models.RemissionNoteItem, 0, len(baseItems))
	for _, baseItem := range baseItems {
		item, ok := baseItem.(*models.Item)
		if !ok {
			b.setError(fmt.Errorf("failed to convert item to proper type"))
			return b
		}
		item.Taxes = []string{constants.TaxIVA}
		taxedAmount := item.GetQuantity() * item.GetUnitPrice() * (1 - item.GetDiscount()/100)
		taxedSaleObj, err := financial.NewAmount(taxedAmount)
		if err != nil {
			b.setError(err)
			return b
		}
		zeroAmount, err := financial.NewAmount(0.0)
		if err != nil {
			b.setError(err)
			return b
		}
		remissionItem := remission_note_models.RemissionNoteItem{
			Item:           item,
			TaxedSale:      *taxedSaleObj,
			NonSubjectSale: *zeroAmount,
			ExemptSale:     *zeroAmount,
		}
		remissionItems = append(remissionItems, remissionItem)
	}
	b.document.RemissionItems = remissionItems
	return b
}

func (b *RemissionNoteBuilder) AddSummary() *RemissionNoteBuilder {
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
	taxedTotal, err := financial.NewAmount(baseSummary.GetTotalTaxed())
	if err != nil {
		b.setError(err)
		return b
	}
	subtotalSales, err := financial.NewAmount(baseSummary.GetSubtotalSales())
	if err != nil {
		b.setError(err)
		return b
	}
	subtotal, err := financial.NewAmount(baseSummary.GetSubTotal())
	if err != nil {
		b.setError(err)
		return b
	}
	totalAmount, err := financial.NewAmount(baseSummary.GetSubTotal())
	if err != nil {
		b.setError(err)
		return b
	}
	zeroAmount, err := financial.NewAmount(0.0)
	if err != nil {
		b.setError(err)
		return b
	}
	b.document.Summary = &remission_note_models.RemissionNoteSummary{
		Summary:            baseSummary,
		TaxedTotal:         taxedTotal,
		NonSubjectTotal:    zeroAmount,
		ExemptTotal:        zeroAmount,
		SubtotalSales:      subtotalSales,
		NonSubjectDiscount: zeroAmount,
		ExemptDiscount:     zeroAmount,
		TaxedDiscount:      zeroAmount,
		TotalDiscount:      zeroAmount,
		Subtotal:           subtotal,
		TotalAmount:        totalAmount,
	}
	return b
}

// BuildAsRemissionNoteInput converts a RemissionNoteModel to RemissionNoteInput for service testing.
func BuildAsRemissionNoteInput(remissionNote *remission_note_models.RemissionNoteModel, receiver *remission_note_models.RemissionNoteReceiver) *remission_note_models.RemissionNoteInput {
	var relatedDocs []models.RelatedDocument
	for _, rd := range remissionNote.RelatedDocuments {
		if r, ok := rd.(*models.RelatedDocument); ok {
			relatedDocs = append(relatedDocs, *r)
		}
	}
	var extension *models.Extension
	if remissionNote.Extension != nil {
		if e, ok := remissionNote.Extension.(*models.Extension); ok {
			extension = e
		}
	}
	return &remission_note_models.RemissionNoteInput{
		InputDataCommon: &models.InputDataCommon{
			Identification: remissionNote.GetIdentification().(*models.Identification),
			Issuer:         remissionNote.GetIssuer().(*models.Issuer),
			Extension:      extension,
			RelatedDocs:    relatedDocs,
		},
		Receiver:         receiver,
		Items:            remissionNote.RemissionItems,
		RemissionSummary: remissionNote.Summary,
	}
}

func BuildValidRemissionNote() (*remission_note_models.RemissionNoteModel, *remission_note_models.RemissionNoteReceiver, error) {
	builder := NewRemissionNoteBuilder()
	builder.AddIdentification().
		AddIssuer().
		AddReceiver().
		AddItems().
		AddSummary()
	doc, err := builder.Build()
	if err != nil {
		return nil, nil, err
	}
	return doc, builder.remissionReceiver, nil
}
