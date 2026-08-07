package fixtures

import (
	"fmt"
	"math"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invoice/invoice_models"
)

// InvoiceBuilder - Specialized builder for Electronic Invoice
// that reuses the base DTEBuilder to avoid code duplication
type InvoiceBuilder struct {
	baseBuilder *DTEBuilder
	document    *invoice_models.ElectronicInvoice
	err         error
}

func NewInvoiceBuilder() *InvoiceBuilder {
	return &InvoiceBuilder{
		baseBuilder: NewDTEBuilder(),
		document: &invoice_models.ElectronicInvoice{
			DTEDocument:  &models.DTEDocument{},
			InvoiceItems: make([]invoice_models.InvoiceItem, 0),
		},
		err: nil,
	}
}

func (b *InvoiceBuilder) Document() *invoice_models.ElectronicInvoice {
	return b.document
}

func (b *InvoiceBuilder) Build() (*invoice_models.ElectronicInvoice, error) {
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

func (b *InvoiceBuilder) BuildWithoutValidation() (*invoice_models.ElectronicInvoice, error) {
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

func (b *InvoiceBuilder) setError(err error) *InvoiceBuilder {
	if b.err == nil && err != nil {
		b.err = err
	}

	if err != nil {
		b.baseBuilder.setError(err)
	}

	return b
}

func (b *InvoiceBuilder) AddIdentification() *InvoiceBuilder {
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

	b.setError(baseIdentification.SetDTEType(constants.FacturaElectronica))

	controlNumber := baseIdentification.GetControlNumber()
	if len(controlNumber) > 4 {
		newControlNumber := "DTE-01" + controlNumber[6:]
		b.setError(baseIdentification.SetControlNumber(newControlNumber))
	}

	return b
}

func (b *InvoiceBuilder) AddIssuer() *InvoiceBuilder {
	b.baseBuilder.AddIssuer()

	if b.baseBuilder.err != nil {
		b.err = b.baseBuilder.err
	}

	return b
}

func (b *InvoiceBuilder) AddReceiver() *InvoiceBuilder {
	b.baseBuilder.AddReceiver()

	if b.baseBuilder.err != nil {
		b.err = b.baseBuilder.err
	}

	return b
}

func (b *InvoiceBuilder) AddReceiverForNaturalReceiver() *InvoiceBuilder {
	b.baseBuilder.AddReceiver()

	if b.baseBuilder.err != nil {
		b.err = b.baseBuilder.err
		return b
	}

	baseReceiver, ok := b.baseBuilder.document.GetReceiver().(*models.Receiver)
	if !ok || baseReceiver == nil {
		b.setError(fmt.Errorf("failed to get receiver from base builder"))
		return b
	}

	name := "CONSUMIDOR FINAL"
	docType := "13"
	docNumber := "00000000-0"

	b.setError(baseReceiver.SetName(&name))
	b.setError(baseReceiver.SetDocumentType(&docType))
	b.setError(baseReceiver.SetDocumentNumber(&docNumber))

	return b
}

func (b *InvoiceBuilder) AddItems() *InvoiceBuilder {
	b.baseBuilder.AddItems()

	if b.baseBuilder.err != nil {
		b.err = b.baseBuilder.err
		return b
	}

	baseItems := b.baseBuilder.document.GetItems()
	invoiceItems := make([]invoice_models.InvoiceItem, 0, len(baseItems))
	for _, baseItem := range baseItems {
		item, ok := baseItem.(*models.Item)
		if !ok {
			b.setError(fmt.Errorf("failed to convert item to proper type"))
			return b
		}

		invoiceItem := invoice_models.InvoiceItem{
			Item: item,
		}

		taxedAmount := item.GetQuantity() * item.GetUnitPrice() * (1 - item.GetDiscount()/100)
		ivaAmount := (taxedAmount / 1.13) * 0.13

		taxedSaleObj, err := financial.NewAmount(taxedAmount)
		if err != nil {
			b.setError(err)
			return b
		}
		invoiceItem.TaxedSale = *taxedSaleObj

		ivaObj, err := financial.NewAmount(ivaAmount)
		if err != nil {
			b.setError(err)
			return b
		}
		invoiceItem.IVAItem = *ivaObj

		zeroAmount, err := financial.NewAmount(0.0)
		if err != nil {
			b.setError(err)
			return b
		}
		invoiceItem.NonSubjectSale = *zeroAmount
		invoiceItem.ExemptSale = *zeroAmount
		invoiceItem.SuggestedPrice = *zeroAmount
		invoiceItem.NonTaxed = *zeroAmount

		invoiceItems = append(invoiceItems, invoiceItem)
	}

	b.document.InvoiceItems = invoiceItems
	return b
}

func (b *InvoiceBuilder) AddSummary() *InvoiceBuilder {
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

	invoiceSummary := invoice_models.InvoiceSummary{
		Summary: baseSummary,
	}

	var totalIva float64
	totalTaxed := financial.NewValidatedAmount(0)
	for _, item := range b.document.InvoiceItems {
		totalIva += item.IVAItem.GetValue()
		totalTaxed.Add(&item.TaxedSale)
	}
	totalIva = math.Round(totalIva*100) / 100
	ivaObj, err := financial.NewAmount(totalIva)
	if err != nil {
		b.setError(err)
		return b
	}
	invoiceSummary.TotalIva = *ivaObj

	zeroAmount, err := financial.NewAmount(0.0)
	if err != nil {
		b.setError(err)
		return b
	}
	invoiceSummary.TaxedDiscount = *zeroAmount
	invoiceSummary.IVARetention = *zeroAmount
	invoiceSummary.IncomeRetention = *zeroAmount
	invoiceSummary.BalanceInFavor = *zeroAmount
	invoiceSummary.TotalTaxed = *totalTaxed
	invoiceSummary.TotalOperation = *totalTaxed
	invoiceSummary.TotalToPay = invoiceSummary.TotalOperation
	invoiceSummary.TotalTaxes = nil

	payments := baseSummary.GetPaymentTypes()
	payments[0].SetAmount(invoiceSummary.TotalToPay.GetValue())

	b.document.InvoiceSummary = invoiceSummary
	return b
}

func (b *InvoiceBuilder) AddSummaryWithCreditOperation() *InvoiceBuilder {
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

func (b *InvoiceBuilder) AddSummaryWithElectronicPayment() *InvoiceBuilder {
	b.AddSummary()

	if b.baseBuilder.err != nil {
		b.err = b.baseBuilder.err
		return b
	}

	electronicPaymentNumber := "12345678901234567890"
	b.document.InvoiceSummary.ElectronicPaymentNumber = &electronicPaymentNumber

	return b
}

func BuildValidInvoice() (*invoice_models.ElectronicInvoice, error) {
	builder := NewInvoiceBuilder()

	builder.AddIdentification().
		AddIssuer().
		AddReceiver().
		AddItems().
		AddSummary().
		AddExtension()

	return builder.Build()
}

func BuildNaturalReceiverInvoice() (*invoice_models.ElectronicInvoice, error) {
	builder := NewInvoiceBuilder()

	builder.AddIdentification().
		AddIssuer().
		AddReceiverForNaturalReceiver().
		AddItems().
		AddSummary()

	return builder.Build()
}

func BuildCreditInvoice() (*invoice_models.ElectronicInvoice, error) {
	builder := NewInvoiceBuilder()

	builder.AddIdentification().
		AddIssuer().
		AddReceiver().
		AddItems().
		AddSummaryWithCreditOperation()

	return builder.Build()
}

func (b *InvoiceBuilder) BuildAsInvoiceData() (*invoice_models.InvoiceData, error) {
	if b == nil {
		return nil, fmt.Errorf("InvoiceBuilder is nil")
	}

	if b.err != nil {
		return nil, b.err
	}

	invoice := b.document
	var appendixes []models.Appendix
	for _, app := range invoice.Appendix {
		if a, ok := app.(*models.Appendix); ok {
			appendixes = append(appendixes, *a)
		}
	}

	var relatedDocs []models.RelatedDocument
	for _, rd := range invoice.RelatedDocuments {
		if r, ok := rd.(*models.RelatedDocument); ok {
			relatedDocs = append(relatedDocs, *r)
		}
	}

	var otherDocs []models.OtherDocument
	for _, od := range invoice.OtherDocuments {
		if o, ok := od.(*models.OtherDocument); ok {
			otherDocs = append(otherDocs, *o)
		}
	}

	var thirdPartySale *models.ThirdPartySale
	if invoice.ThirdPartySale != nil {
		if t, ok := invoice.ThirdPartySale.(*models.ThirdPartySale); ok {
			thirdPartySale = t
		}
	}

	var extension *models.Extension
	if invoice.Extension != nil {
		if e, ok := invoice.Extension.(*models.Extension); ok {
			extension = e
		}
	}

	return &invoice_models.InvoiceData{
		InputDataCommon: &models.InputDataCommon{
			Identification: invoice.GetIdentification().(*models.Identification),
			Issuer:         invoice.GetIssuer().(*models.Issuer),
			Receiver:       invoice.GetReceiver().(*models.Receiver),
			Extension:      extension,
			RelatedDocs:    relatedDocs,
			OtherDocs:      otherDocs,
			ThirdPartySale: thirdPartySale,
			Appendixes:     appendixes,
		},
		Items:          invoice.InvoiceItems,
		InvoiceSummary: &invoice.InvoiceSummary,
	}, nil
}

func (b *InvoiceBuilder) AddExtension() *InvoiceBuilder {
	b.baseBuilder.AddExtension()

	if b.baseBuilder.err != nil {
		b.err = b.baseBuilder.err
		return b
	}

	baseExtension, ok := b.baseBuilder.document.GetExtension().(*models.Extension)
	if !ok || baseExtension == nil {
		b.setError(fmt.Errorf("failed to get extension from base builder"))
		return b
	}

	b.document.Extension = baseExtension

	return b
}

func BuildAsInvoiceData(invoice *invoice_models.ElectronicInvoice) *invoice_models.InvoiceData {
	var appendixes []models.Appendix
	for _, app := range invoice.Appendix {
		if a, ok := app.(*models.Appendix); ok {
			appendixes = append(appendixes, *a)
		}
	}

	var relatedDocs []models.RelatedDocument
	for _, rd := range invoice.RelatedDocuments {
		if r, ok := rd.(*models.RelatedDocument); ok {
			relatedDocs = append(relatedDocs, *r)
		}
	}

	var otherDocs []models.OtherDocument
	for _, od := range invoice.OtherDocuments {
		if o, ok := od.(*models.OtherDocument); ok {
			otherDocs = append(otherDocs, *o)
		}
	}

	var thirdPartySale *models.ThirdPartySale
	if invoice.ThirdPartySale != nil {
		if t, ok := invoice.ThirdPartySale.(*models.ThirdPartySale); ok {
			thirdPartySale = t
		}
	}

	var extension *models.Extension
	if invoice.Extension != nil {
		if e, ok := invoice.Extension.(*models.Extension); ok {
			extension = e
		}
	}

	return &invoice_models.InvoiceData{
		InputDataCommon: &models.InputDataCommon{
			Identification: invoice.GetIdentification().(*models.Identification),
			Issuer:         invoice.GetIssuer().(*models.Issuer),
			Receiver:       invoice.GetReceiver().(*models.Receiver),
			Extension:      extension,
			RelatedDocs:    relatedDocs,
			OtherDocs:      otherDocs,
			ThirdPartySale: thirdPartySale,
			Appendixes:     appendixes,
		},
		Items:          invoice.InvoiceItems,
		InvoiceSummary: &invoice.InvoiceSummary,
	}
}
