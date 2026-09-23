package fixtures

import (
	"fmt"
	"math"
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/ccf/ccf_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/base"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/identification"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/item"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/temporal"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/credit_note/credit_note_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invalidation/invalidation_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/invoice/invoice_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/retention/retention_models"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

type DTEBuilder struct {
	document *models.DTEDocument
	err      error
}

// NewDTEBuilder creates a new builder for DTEDocument
func NewDTEBuilder() *DTEBuilder {
	return &DTEBuilder{
		document: &models.DTEDocument{
			Items:            make([]interfaces.Item, 0),
			Appendix:         make([]interfaces.Appendix, 0),
			RelatedDocuments: make([]interfaces.RelatedDocument, 0),
			OtherDocuments:   make([]interfaces.OtherDocuments, 0),
		},
	}
}

func (b *DTEBuilder) Document() *models.DTEDocument {
	return b.document
}

// Build builds and validates the DTE document, returning any error that occurred
func (b *DTEBuilder) Build() (*models.DTEDocument, error) {
	if b.err != nil {
		return nil, b.err
	}

	err := b.document.Validate()
	if err != nil {
		return nil, err
	}

	dteErr := b.document.ValidateDTERules()
	if dteErr != nil {
		return nil, dteErr
	}

	return b.document, nil
}

// BuildWithoutValidation builds the document without validations
func (b *DTEBuilder) BuildWithoutValidation() (*models.DTEDocument, error) {
	if b.err != nil {
		return nil, b.err
	}
	return b.document, nil
}

// setError is a helper method that sets the error if one has not already been set
func (b *DTEBuilder) setError(err error) *DTEBuilder {
	if b.err == nil && err != nil {
		b.err = err
	}
	return b
}

// AddIdentification adds valid default identification data
func (b *DTEBuilder) AddIdentification() *DTEBuilder {
	if b.err != nil {
		return b
	}

	identification := &models.Identification{}

	b.setError(identification.SetVersion(1))
	b.setError(identification.SetAmbient(constants.Testing))
	b.setError(identification.SetDTEType(constants.FacturaElectronica))
	b.setError(identification.SetControlNumber("DTE-01-12345678-123456789012345"))
	b.setError(identification.GenerateCode())
	b.setError(identification.SetModelType(constants.ModeloFacturacionPrevio))
	b.setError(identification.SetOperationType(1))
	b.setError(identification.SetEmissionDate(utils.TimeNow()))
	b.setError(identification.SetEmissionTime(utils.TimeNow()))
	b.setError(identification.SetCurrency("USD"))

	if b.err == nil {
		b.setError(b.document.SetIdentification(identification))
	}

	return b
}

// AddIdentificationWithContingency adds identification data with contingency
func (b *DTEBuilder) AddIdentificationWithContingency() *DTEBuilder {
	b.AddIdentification()

	if b.err != nil {
		return b
	}

	identification, ok := b.document.GetIdentification().(*models.Identification)
	if !ok || identification == nil {
		b.setError(fmt.Errorf("failed to get identification"))
		return b
	}

	b.setError(identification.SetOperationType(constants.TransmisionContingencia))
	b.setError(identification.SetModelType(constants.ModeloFacturacionDiferido))

	contingencyType := constants.FallaServicioInternet
	contingencyReason := "Falla de servicio de internet del proveedor"
	b.setError(identification.SetContingencyType(&contingencyType))
	b.setError(identification.SetContingencyReason(&contingencyReason))

	return b
}

// AddIssuer adds valid default issuer data
func (b *DTEBuilder) AddIssuer() *DTEBuilder {
	if b.err != nil {
		return b
	}

	issuer := &models.Issuer{}

	b.setError(issuer.SetNIT("12345678901234"))
	b.setError(issuer.SetNRC("12345678"))
	b.setError(issuer.SetName("COMPANY EXAMPLE, S.A. DE C.V."))
	b.setError(issuer.SetActivityCode("12345"))
	b.setError(issuer.SetActivityDescription("Electronic products sales"))
	b.setError(issuer.SetEstablishmentType(constants.CasaMatriz))

	address := &models.Address{}
	b.setError(address.SetDepartment("06"))
	b.setError(address.SetMunicipality("21"))
	b.setError(address.SetDistrict("01"))
	b.setError(address.SetComplement("Example Street, Central Building #123"))

	if b.err == nil {
		b.setError(issuer.SetAddress(address))
	}

	b.setError(issuer.SetPhone("22225555"))
	b.setError(issuer.SetEmail("info@google.com"))
	b.setError(issuer.SetCommercialName("ELECTRO STORE"))

	establishmentCode := "001"
	establishmentMHCode := "EST001"
	posCode := "POS01"
	posMHCode := "POS001"
	b.setError(issuer.SetEstablishmentCode(&establishmentCode))
	b.setError(issuer.SetEstablishmentMHCode(&establishmentMHCode))
	b.setError(issuer.SetPOSCode(&posCode))
	b.setError(issuer.SetPOSMHCode(&posMHCode))

	if b.err == nil {
		b.setError(b.document.SetIssuer(issuer))
	}

	return b
}

// AddIssuerWithInvalidNIT adds issuer data with an invalid NIT for testing
func (b *DTEBuilder) AddIssuerWithInvalidNIT() *DTEBuilder {
	b.AddIssuer()

	if b.err != nil {
		return b
	}

	issuer, ok := b.document.GetIssuer().(*models.Issuer)
	if !ok || issuer == nil {
		b.setError(fmt.Errorf("failed to get issuer"))
		return b
	}

	b.setError(issuer.SetNIT("123456789"))

	return b
}

// AddReceiver adds valid default receiver data for an invoice
func (b *DTEBuilder) AddReceiver() *DTEBuilder {
	if b.err != nil {
		return b
	}

	receiver := &models.Receiver{}

	name := "John Albert Smith"
	email := "john.smith@email.com"
	phone := "77778888"
	docType := constants.DUI
	docNumber := "01234567-8"

	b.setError(receiver.SetName(&name))
	b.setError(receiver.SetDocumentType(&docType))
	b.setError(receiver.SetDocumentNumber(&docNumber))
	b.setError(receiver.SetEmail(&email))
	b.setError(receiver.SetPhone(&phone))

	address := &models.Address{}
	b.setError(address.SetDepartment("06"))
	b.setError(address.SetMunicipality("22"))
	b.setError(address.SetDistrict("01"))
	b.setError(address.SetComplement("Example Neighborhood, House #456"))

	if b.err == nil {
		b.setError(receiver.SetAddress(address))
	}

	if b.err == nil {
		b.setError(b.document.SetReceiver(receiver))
	}

	return b
}

// AddReceiverForCCF adds valid default receiver data for CCF
func (b *DTEBuilder) AddReceiverForCCF() *DTEBuilder {
	b.AddReceiver()

	if b.err != nil {
		return b
	}

	receiver, ok := b.document.GetReceiver().(*models.Receiver)
	if !ok || receiver == nil {
		b.setError(fmt.Errorf("failed to get receiver"))
		return b
	}

	nrc := "987654"
	activityDescription := "Purchase of goods and services"
	activityCode := "56789"
	commercialName := "CLIENT COMPANY INC."
	nit := "98765432101234"

	b.setError(receiver.SetNRC(&nrc))
	b.setError(receiver.SetActivityDescription(&activityDescription))
	b.setError(receiver.SetActivityCode(&activityCode))
	b.setError(receiver.SetCommercialName(&commercialName))
	b.setError(receiver.SetNIT(&nit))

	return b
}

// AddReceiverWithNoNRC adds receiver data without NRC (invalid for CCF)
func (b *DTEBuilder) AddReceiverWithNoNRC() *DTEBuilder {
	b.AddReceiver()
	return b
}

// AddItems adds valid default items
func (b *DTEBuilder) AddItems() *DTEBuilder {
	if b.err != nil {
		return b
	}

	items := make([]interfaces.Item, 0, 2)

	item1 := &models.Item{}
	b.setError(item1.SetNumber(1))
	b.setError(item1.SetType(constants.Producto))
	b.setError(item1.SetDescription("HP EliteBook 840 G8 Laptop"))
	b.setError(item1.SetQuantity(1.0))
	b.setError(item1.SetUnitMeasure(1))
	b.setError(item1.SetUnitPrice(899.99))
	b.setError(item1.SetDiscount(0.0))
	b.setError(item1.SetItemCode("LAPTOP-HP-001"))

	if b.err == nil {
		items = append(items, item1)
	}

	item2 := &models.Item{}
	b.setError(item2.SetNumber(2))
	b.setError(item2.SetType(constants.Servicio))
	b.setError(item2.SetDescription("Software configuration and installation"))
	b.setError(item2.SetQuantity(1.0))
	b.setError(item2.SetUnitMeasure(1))
	b.setError(item2.SetUnitPrice(50.00))
	b.setError(item2.SetDiscount(0.0))
	b.setError(item2.SetTaxes([]string{constants.TaxIVA}))
	b.setError(item2.SetItemCode("Licencia de Windows 11"))

	if b.err == nil {
		items = append(items, item2)
	}

	if b.err == nil {
		b.setError(b.document.SetItems(items))
	}

	return b
}

// AddItemsWithDiscount adds items with discounts
func (b *DTEBuilder) AddItemsWithDiscount() *DTEBuilder {
	if b.err != nil {
		return b
	}

	items := make([]interfaces.Item, 0, 2)

	item1 := &models.Item{}
	b.setError(item1.SetNumber(1))
	b.setError(item1.SetType(constants.Producto))
	b.setError(item1.SetDescription("HP EliteBook 840 G8 Laptop"))
	b.setError(item1.SetQuantity(1.0))
	b.setError(item1.SetUnitMeasure(1))
	b.setError(item1.SetUnitPrice(899.99))
	b.setError(item1.SetDiscount(10.0))
	b.setError(item1.SetTaxes([]string{constants.TaxIVA}))
	b.setError(item1.SetItemCode("LAPTOP-HP-001"))

	if b.err == nil {
		items = append(items, item1)
	}

	item2 := &models.Item{}
	b.setError(item2.SetNumber(2))
	b.setError(item2.SetType(constants.Servicio))
	b.setError(item2.SetDescription("Software configuration and installation"))
	b.setError(item2.SetQuantity(1.0))
	b.setError(item2.SetUnitMeasure(1))
	b.setError(item2.SetUnitPrice(50.00))
	b.setError(item2.SetDiscount(5.0))
	b.setError(item2.SetTaxes([]string{constants.TaxIVA}))

	if b.err == nil {
		items = append(items, item2)
	}

	if b.err == nil {
		b.setError(b.document.SetItems(items))
	}

	return b
}

// AddItemsWithInvalidTax adds items with invalid taxes for testing
func (b *DTEBuilder) AddItemsWithInvalidTax() *DTEBuilder {
	if b.err != nil {
		return b
	}

	invalidItem := &models.Item{}
	b.setError(invalidItem.SetNumber(1))
	b.setError(invalidItem.SetType(constants.Producto))
	b.setError(invalidItem.SetDescription("Product with invalid tax"))
	b.setError(invalidItem.SetQuantity(1.0))
	b.setError(invalidItem.SetUnitMeasure(1))
	b.setError(invalidItem.SetUnitPrice(100.00))
	b.setError(invalidItem.SetDiscount(0.0))
	b.setError(invalidItem.SetTaxes([]string{"ZZ"}))

	if b.err == nil {
		items := []interfaces.Item{invalidItem}
		b.setError(b.document.SetItems(items))
	}

	return b
}

// AddSummary adds a valid default summary
func (b *DTEBuilder) AddSummary() *DTEBuilder {
	if b.err != nil {
		return b
	}

	var totalTaxed float64
	for _, i := range b.document.GetItems() {
		item, ok := i.(*models.Item)
		if !ok {
			continue
		}

		discountFactor := 1.0 - (item.GetDiscount() / 100.0)
		itemTotal := item.GetQuantity() * item.GetUnitPrice() * discountFactor
		totalTaxed += itemTotal
	}

	iva := totalTaxed * 0.13
	totalOperation := totalTaxed + iva

	summary := &models.Summary{}

	b.setError(summary.SetTotalNonSubject(0.00))
	b.setError(summary.SetTotalExempt(0.00))
	b.setError(summary.SetTotalTaxed(math.Round(totalTaxed*100) / 100))
	b.setError(summary.SetSubTotal(math.Round(totalTaxed*100) / 100))
	b.setError(summary.SetSubtotalSales(math.Round(totalTaxed*100) / 100))
	b.setError(summary.SetNonSubjectDiscount(0.00))
	b.setError(summary.SetExemptDiscount(0.00))
	b.setError(summary.SetDiscountPercentage(0.00))
	b.setError(summary.SetTotalDiscount(0.00))
	b.setError(summary.SetTotalOperation(math.Round(totalOperation*100) / 100))
	b.setError(summary.SetTotalNotTaxed(0.00))
	b.setError(summary.SetOperationCondition(constants.Cash))
	b.setError(summary.SetTotalToPay(math.Round(totalOperation*100) / 100))
	b.setError(summary.SetTotalInWords("UN MIL DOLARES CON 00/100 CENTAVOS"))

	tax := &models.Tax{}
	b.setError(tax.SetCode(constants.TaxIVA))
	b.setError(tax.SetDescription("IVA 13%"))
	b.setError(tax.SetValue(iva))

	var taxes []interfaces.Tax
	if b.err == nil {
		taxes = append(taxes, tax)
		b.setError(summary.SetTotalTaxes(taxes))
	}

	payment := &models.PaymentType{}
	b.setError(payment.SetCode(constants.BilletesMonedas))
	b.setError(payment.SetAmount(math.Round(totalOperation*100) / 100))
	b.setError(payment.SetReference("Cash payment"))

	var payments []interfaces.PaymentType
	if b.err == nil {
		payments = append(payments, payment)
		b.setError(summary.SetPaymentTypes(payments))
	}

	if b.err == nil {
		b.setError(b.document.SetSummary(summary))
	}

	return b
}

// AddSummaryWithCredit adds a summary with a credit payment condition
func (b *DTEBuilder) AddSummaryWithCredit() *DTEBuilder {
	b.AddSummary()

	if b.err != nil {
		return b
	}

	summary, ok := b.document.GetSummary().(*models.Summary)
	if !ok || summary == nil {
		b.setError(fmt.Errorf("failed to get summary"))
		return b
	}

	b.setError(summary.SetOperationCondition(constants.Credit))

	if len(summary.GetPaymentTypes()) > 0 {
		paymentType, ok := summary.GetPaymentTypes()[0].(*models.PaymentType)
		if ok {
			b.setError(paymentType.SetCode(constants.TransBancaria))
			b.setError(paymentType.SetReference("Bank transfer"))

			term := "01"
			period := 30
			b.setError(paymentType.SetTerm(&term))
			b.setError(paymentType.SetPeriod(&period))
		}
	}

	return b
}

// AddSummaryWithInvalidPayment adds a summary with invalid payments for testing
func (b *DTEBuilder) AddSummaryWithInvalidPayment() *DTEBuilder {
	b.AddSummary()

	if b.err != nil {
		return b
	}

	summary, ok := b.document.GetSummary().(*models.Summary)
	if !ok || summary == nil {
		b.setError(fmt.Errorf("failed to get summary"))
		return b
	}

	b.setError(summary.SetOperationCondition(constants.Credit))

	return b
}

// AddSummaryWithInvalidTotal adds a summary with an incorrect total for testing
func (b *DTEBuilder) AddSummaryWithInvalidTotal() *DTEBuilder {
	b.AddSummary()

	if b.err != nil {
		return b
	}

	summary, ok := b.document.GetSummary().(*models.Summary)
	if !ok || summary == nil {
		b.setError(fmt.Errorf("failed to get summary"))
		return b
	}

	b.setError(summary.SetTotalOperation(1000.00))
	b.setError(summary.SetTotalToPay(1000.00))

	if len(summary.GetPaymentTypes()) > 0 {
		paymentType, ok := summary.GetPaymentTypes()[0].(*models.PaymentType)
		if ok {
			b.setError(paymentType.SetAmount(1000.00))
		}
	}

	return b
}

// AddExtension adds a valid default extension
func (b *DTEBuilder) AddExtension() *DTEBuilder {
	if b.err != nil {
		return b
	}

	extension := &models.Extension{}

	b.setError(extension.SetDeliveryName("Mary Rodriguez"))
	b.setError(extension.SetDeliveryDocument("12345678-9"))
	b.setError(extension.SetReceiverName("Louis Gonzalez"))
	b.setError(extension.SetReceiverDocument("98765432-1"))

	vehiculePlate := "P123-456"
	observation := "Delivery at building reception. Contact the recipient."
	b.setError(extension.SetVehiculePlate(&vehiculePlate))
	b.setError(extension.SetObservation(&observation))

	if b.err == nil {
		b.setError(b.document.SetExtension(extension))
	}

	return b
}

// AddAppendixes adds valid default appendixes
func (b *DTEBuilder) AddAppendixes() *DTEBuilder {
	if b.err != nil {
		return b
	}

	appendixCount := 3
	appendixesInterfaces := make([]interfaces.Appendix, 0, appendixCount)

	for i := 0; i < appendixCount; i++ {
		appendix := &models.Appendix{}

		b.setError(appendix.SetField(fmt.Sprintf("NOTE%d", i+1)))
		b.setError(appendix.SetLabel(fmt.Sprintf("Additional Information %d", i+1)))
		b.setError(appendix.SetValue(fmt.Sprintf("Additional information content %d", i+1)))

		if b.err == nil {
			appendixesInterfaces = append(appendixesInterfaces, appendix)
		}
	}

	if b.err == nil {
		b.setError(b.document.SetAppendix(appendixesInterfaces))
	}

	return b
}

// AddRelatedDocuments adds valid default related documents
func (b *DTEBuilder) AddRelatedDocuments() *DTEBuilder {
	if b.err != nil {
		return b
	}

	relatedDoc := &models.RelatedDocument{}

	b.setError(relatedDoc.SetDocumentType(constants.NotaRemisionElectronica))
	b.setError(relatedDoc.SetGenerationType(constants.ElectronicDocument))
	b.setError(relatedDoc.SetDocumentNumber("DA1E261A-BAD7-460F-AD15-04F2E281FC6A"))
	b.setError(relatedDoc.SetEmissionDate(utils.TimeNow().Add(-24 * time.Hour)))

	if b.err == nil {
		relatedDocs := make([]interfaces.RelatedDocument, 0, 1)
		relatedDocs = append(relatedDocs, relatedDoc)
		b.setError(b.document.SetRelatedDocuments(relatedDocs))
	}

	return b
}

// AddInvalidRelatedDocument adds invalid related documents for testing
func (b *DTEBuilder) AddInvalidRelatedDocument() *DTEBuilder {
	if b.err != nil {
		return b
	}

	relatedDoc := &models.RelatedDocument{}

	b.setError(relatedDoc.SetDocumentType(constants.NotaRemisionElectronica))
	b.setError(relatedDoc.SetGenerationType(constants.ElectronicDocument))
	b.setError(relatedDoc.SetDocumentNumber("DA1E261A-BAD7-460F-AD15-04F2E281FC6A"))
	b.setError(relatedDoc.SetEmissionDate(utils.TimeNow().Add(24 * time.Hour)))

	if b.err == nil {
		relatedDocs := make([]interfaces.RelatedDocument, 0, 1)
		relatedDocs = append(relatedDocs, relatedDoc)
		b.setError(b.document.SetRelatedDocuments(relatedDocs))
	}

	return b
}

// AddOtherDocuments adds valid default other documents
func (b *DTEBuilder) AddOtherDocuments() *DTEBuilder {
	if b.err != nil {
		return b
	}

	description := "Reference document"
	detail := "Reference document detail"

	otherDoc := &models.OtherDocument{}

	b.setError(otherDoc.SetAssociatedDocument(constants.DocumentoEmisor))
	b.setError(otherDoc.SetDescription(description))
	b.setError(otherDoc.SetDetail(detail))

	if b.err == nil {
		otherDocs := make([]interfaces.OtherDocuments, 0, 1)
		otherDocs = append(otherDocs, otherDoc)
		b.setError(b.document.SetOtherDocuments(otherDocs))
	}

	return b
}

// AddMedicalDocument adds a valid default medical document
func (b *DTEBuilder) AddMedicalDocument() *DTEBuilder {
	if b.err != nil {
		return b
	}

	doctor := &models.DoctorInfo{}

	b.setError(doctor.SetName("Dr. John Smith"))
	b.setError(doctor.SetServiceType(1))

	nit := "12345678901234"
	b.setError(doctor.SetNIT(nit))

	medicalDoc := &models.OtherDocument{}

	b.setError(medicalDoc.SetAssociatedDocument(constants.DocumentoMedico))
	b.setError(medicalDoc.SetDoctor(doctor))

	var otherDocs []interfaces.OtherDocuments

	if existingDocs := b.document.GetOtherDocuments(); existingDocs != nil && len(existingDocs) > 0 {
		otherDocs = append(existingDocs, medicalDoc)
	} else {
		otherDocs = []interfaces.OtherDocuments{medicalDoc}
	}

	if b.err == nil {
		b.setError(b.document.SetOtherDocuments(otherDocs))
	}

	return b
}

// AddInvalidMedicalDocument adds an invalid medical document for testing
func (b *DTEBuilder) AddInvalidMedicalDocument() *DTEBuilder {
	if b.err != nil {
		return b
	}

	doctor := &models.DoctorInfo{}

	b.setError(doctor.SetName("Dr. John Smith"))
	b.setError(doctor.SetServiceType(1))

	nit := "12345678901234"
	b.setError(doctor.SetNIT(nit))

	description := "Invalid description for medical document"
	detail := "Invalid detail for medical document"

	medicalDoc := &models.OtherDocument{}

	b.setError(medicalDoc.SetAssociatedDocument(constants.DocumentoMedico))
	b.setError(medicalDoc.SetDoctor(doctor))
	b.setError(medicalDoc.SetDescription(description))
	b.setError(medicalDoc.SetDetail(detail))

	if b.err == nil {
		otherDocs := make([]interfaces.OtherDocuments, 0, 1)
		otherDocs = append(otherDocs, medicalDoc)
		b.setError(b.document.SetOtherDocuments(otherDocs))
	}

	return b
}

// AddThirdPartySale adds a valid default third-party sale
func (b *DTEBuilder) AddThirdPartySale() *DTEBuilder {
	if b.err != nil {
		return b
	}

	thirdPartySale := &models.ThirdPartySale{}

	b.setError(thirdPartySale.SetNIT("98765432101234"))
	b.setError(thirdPartySale.SetName("Third Party Company, Inc."))

	if b.err == nil {
		b.setError(b.document.SetThirdPartySale(thirdPartySale))
	}

	if b.err == nil {
		for i, itemInterface := range b.document.GetItems() {
			item, ok := itemInterface.(*models.Item)
			if ok {
				relatedDoc := fmt.Sprintf("THIRD-PARTY-DOC-%d", i+1)
				b.setError(item.SetRelatedDoc(&relatedDoc))
			}
		}
	}

	return b
}

// AddInvalidThirdPartySale adds an invalid third-party sale for testing
func (b *DTEBuilder) AddInvalidThirdPartySale() *DTEBuilder {
	if b.err != nil {
		return b
	}

	thirdPartySale := &models.ThirdPartySale{}

	b.setError(thirdPartySale.SetNIT("98765432101234"))
	b.setError(thirdPartySale.SetName("Third Party Company, Inc."))

	if b.err == nil {
		b.setError(b.document.SetThirdPartySale(thirdPartySale))
	}

	if b.err == nil && len(b.document.GetItems()) > 0 {
		item, ok := b.document.GetItems()[0].(*models.Item)
		if ok {
			relatedDoc := "THIRD-PARTY-DOC-1"
			b.setError(item.SetRelatedDoc(&relatedDoc))
		}
	}

	return b
}

// BuildElectronicInvoice builds a valid electronic invoice
func (b *DTEBuilder) BuildElectronicInvoice() (*invoice_models.ElectronicInvoice, error) {
	b.AddIdentification().
		AddIssuer().
		AddReceiver().
		AddItems().
		AddSummary()

	if b.err != nil {
		return nil, b.err
	}

	baseDoc, err := b.BuildWithoutValidation()
	if err != nil {
		return nil, err
	}

	invoice := &invoice_models.ElectronicInvoice{
		DTEDocument:  baseDoc,
		InvoiceItems: make([]invoice_models.InvoiceItem, 0),
		InvoiceSummary: invoice_models.InvoiceSummary{
			Summary: baseDoc.GetSummary().(*models.Summary),
		},
	}

	for _, item := range baseDoc.GetItems() {
		baseItem, ok := item.(*models.Item)
		if ok {
			invoiceItem := invoice_models.InvoiceItem{
				Item: baseItem,
			}

			taxedAmount := baseItem.GetQuantity() * baseItem.GetUnitPrice() * (1 - baseItem.GetDiscount()/100)
			ivaAmount := taxedAmount * 0.13

			amountObj, err := financial.NewAmount(taxedAmount)
			if err != nil {
				return nil, err
			}
			invoiceItem.TaxedSale = *amountObj

			zeroAmountObj, err := financial.NewAmount(0)
			if err != nil {
				return nil, err
			}
			invoiceItem.NonSubjectSale = *zeroAmountObj
			invoiceItem.ExemptSale = *zeroAmountObj
			invoiceItem.SuggestedPrice = *zeroAmountObj
			invoiceItem.NonTaxed = *zeroAmountObj

			ivaObj, err := financial.NewAmount(ivaAmount)
			if err != nil {
				return nil, err
			}
			invoiceItem.IVAItem = *ivaObj

			invoice.InvoiceItems = append(invoice.InvoiceItems, invoiceItem)
		}
	}

	totalTaxed := invoice.GetSummary().GetTotalTaxed()
	totalIVA := totalTaxed * 0.13

	ivaAmountObj, err := financial.NewAmount(totalIVA)
	if err != nil {
		return nil, err
	}
	invoice.InvoiceSummary.TotalIva = *ivaAmountObj

	zeroAmountObj, err := financial.NewAmount(0)
	if err != nil {
		return nil, err
	}
	invoice.InvoiceSummary.TaxedDiscount = *zeroAmountObj
	invoice.InvoiceSummary.IVARetention = *zeroAmountObj
	invoice.InvoiceSummary.IncomeRetention = *zeroAmountObj
	invoice.InvoiceSummary.BalanceInFavor = *zeroAmountObj

	baseDTE := invoice.DTEDocument
	err = baseDTE.Validate()
	if err != nil {
		return nil, err
	}

	dteErr := baseDTE.ValidateDTERules()
	if dteErr != nil {
		return nil, dteErr
	}

	return invoice, nil
}

// BuildInvalidElectronicInvoice builds an invalid electronic invoice
func (b *DTEBuilder) BuildInvalidElectronicInvoice() (*invoice_models.ElectronicInvoice, error) {
	b.AddIdentification().
		AddIssuer().
		AddReceiver().
		AddItems().
		AddSummary()

	if b.err != nil {
		return nil, b.err
	}

	baseDoc, err := b.BuildWithoutValidation()
	if err != nil {
		return nil, err
	}

	invoice := &invoice_models.ElectronicInvoice{
		DTEDocument:  baseDoc,
		InvoiceItems: make([]invoice_models.InvoiceItem, 0),
		InvoiceSummary: invoice_models.InvoiceSummary{
			Summary: baseDoc.GetSummary().(*models.Summary),
		},
	}

	for _, item := range baseDoc.GetItems() {
		baseItem, ok := item.(*models.Item)
		if ok {
			invoiceItem := invoice_models.InvoiceItem{
				Item: baseItem,
			}

			taxedAmount := baseItem.GetQuantity() * baseItem.GetUnitPrice() * (1 - baseItem.GetDiscount()/100)

			ivaIncorrecto := taxedAmount * 0.20

			amountObj, err := financial.NewAmount(taxedAmount)
			if err != nil {
				return nil, err
			}
			invoiceItem.TaxedSale = *amountObj

			zeroAmountObj, err := financial.NewAmount(0)
			if err != nil {
				return nil, err
			}
			invoiceItem.NonSubjectSale = *zeroAmountObj
			invoiceItem.ExemptSale = *zeroAmountObj
			invoiceItem.SuggestedPrice = *zeroAmountObj
			invoiceItem.NonTaxed = *zeroAmountObj

			ivaObj, err := financial.NewAmount(ivaIncorrecto)
			if err != nil {
				return nil, err
			}
			invoiceItem.IVAItem = *ivaObj

			invoice.InvoiceItems = append(invoice.InvoiceItems, invoiceItem)
		}
	}

	totalTaxed := invoice.GetSummary().GetTotalTaxed()

	totalIVAIncorrecto := totalTaxed * 0.10

	ivaAmountObj, err := financial.NewAmount(totalIVAIncorrecto)
	if err != nil {
		return nil, err
	}
	invoice.InvoiceSummary.TotalIva = *ivaAmountObj

	zeroAmountObj, err := financial.NewAmount(0)
	if err != nil {
		return nil, err
	}
	invoice.InvoiceSummary.TaxedDiscount = *zeroAmountObj
	invoice.InvoiceSummary.IVARetention = *zeroAmountObj
	invoice.InvoiceSummary.IncomeRetention = *zeroAmountObj
	invoice.InvoiceSummary.BalanceInFavor = *zeroAmountObj

	return invoice, nil
}

// BuildCreditFiscalDocument builds a valid CCF
func (b *DTEBuilder) BuildCreditFiscalDocument() (*ccf_models.CreditFiscalDocument, error) {
	b.AddIdentification()

	identification, ok := b.document.GetIdentification().(*models.Identification)
	if ok && identification != nil {
		b.setError(identification.SetDTEType(constants.CCFElectronico))
	}

	b.AddIssuer().
		AddReceiverForCCF().
		AddItems().
		AddSummary()

	if b.err != nil {
		return nil, b.err
	}

	baseDoc, err := b.BuildWithoutValidation()
	if err != nil {
		return nil, err
	}

	ccf := &ccf_models.CreditFiscalDocument{
		DTEDocument: baseDoc,
		CreditItems: make([]ccf_models.CreditItem, 0),
		CreditSummary: ccf_models.CreditSummary{
			Summary: baseDoc.GetSummary().(*models.Summary),
		},
	}

	for _, item := range baseDoc.GetItems() {
		baseItem, ok := item.(*models.Item)
		if ok {
			creditItem := ccf_models.CreditItem{
				Item: baseItem,
			}

			taxedAmount := baseItem.GetQuantity() * baseItem.GetUnitPrice() * (1 - baseItem.GetDiscount()/100)

			amountObj, err := financial.NewAmount(taxedAmount)
			if err != nil {
				return nil, err
			}
			creditItem.TaxedSale = *amountObj

			zeroAmountObj, err := financial.NewAmount(0)
			if err != nil {
				return nil, err
			}
			creditItem.NonSubjectSale = *zeroAmountObj
			creditItem.ExemptSale = *zeroAmountObj
			creditItem.SuggestedPrice = *zeroAmountObj
			creditItem.NonTaxed = *zeroAmountObj

			ccf.CreditItems = append(ccf.CreditItems, creditItem)
		}
	}

	zeroAmountObj, err := financial.NewAmount(0)
	if err != nil {
		return nil, err
	}
	ccf.CreditSummary.TaxedDiscount = *zeroAmountObj
	ccf.CreditSummary.IVAPerception = *zeroAmountObj
	ccf.CreditSummary.IVARetention = *zeroAmountObj
	ccf.CreditSummary.IncomeRetention = *zeroAmountObj
	ccf.CreditSummary.BalanceInFavor = *zeroAmountObj
	ccf.CreditSummary.ElectronicPaymentNumber = nil

	baseDTE := ccf.DTEDocument
	err = baseDTE.Validate()
	if err != nil {
		return nil, err
	}

	dteErr := baseDTE.ValidateDTERules()
	if dteErr != nil {
		return nil, dteErr
	}

	return ccf, nil
}

// BuildInvalidCreditFiscalDocument builds an invalid CCF (receiver without NRC)
func (b *DTEBuilder) BuildInvalidCreditFiscalDocument() (*ccf_models.CreditFiscalDocument, error) {
	b.AddIdentification()

	identification, ok := b.document.GetIdentification().(*models.Identification)
	if ok && identification != nil {
		b.setError(identification.SetDTEType(constants.CCFElectronico))
	}

	b.AddIssuer().
		AddReceiverWithNoNRC().
		AddItems().
		AddSummary()

	if b.err != nil {
		return nil, b.err
	}

	baseDoc, err := b.BuildWithoutValidation()
	if err != nil {
		return nil, err
	}

	ccf := &ccf_models.CreditFiscalDocument{
		DTEDocument: baseDoc,
		CreditItems: make([]ccf_models.CreditItem, 0),
		CreditSummary: ccf_models.CreditSummary{
			Summary: baseDoc.GetSummary().(*models.Summary),
		},
	}

	for _, item := range baseDoc.GetItems() {
		baseItem, ok := item.(*models.Item)
		if ok {
			creditItem := ccf_models.CreditItem{
				Item: baseItem,
			}

			taxedAmount := baseItem.GetQuantity() * baseItem.GetUnitPrice() * (1 - baseItem.GetDiscount()/100)

			amountObj, err := financial.NewAmount(taxedAmount)
			if err != nil {
				return nil, err
			}
			creditItem.TaxedSale = *amountObj

			zeroAmountObj, err := financial.NewAmount(0)
			if err != nil {
				return nil, err
			}
			creditItem.NonSubjectSale = *zeroAmountObj
			creditItem.ExemptSale = *zeroAmountObj
			creditItem.SuggestedPrice = *zeroAmountObj
			creditItem.NonTaxed = *zeroAmountObj

			ccf.CreditItems = append(ccf.CreditItems, creditItem)
		}
	}

	zeroAmountObj, err := financial.NewAmount(0)
	if err != nil {
		return nil, err
	}
	ccf.CreditSummary.TaxedDiscount = *zeroAmountObj
	ccf.CreditSummary.IVAPerception = *zeroAmountObj
	ccf.CreditSummary.IVARetention = *zeroAmountObj
	ccf.CreditSummary.IncomeRetention = *zeroAmountObj
	ccf.CreditSummary.BalanceInFavor = *zeroAmountObj
	ccf.CreditSummary.ElectronicPaymentNumber = nil

	return ccf, nil
}

// BuildCreditNote builds a valid credit note
func (b *DTEBuilder) BuildCreditNote() (*credit_note_models.CreditNoteModel, error) {
	b.AddIdentification()

	identification, ok := b.document.GetIdentification().(*models.Identification)
	if ok && identification != nil {
		b.setError(identification.SetDTEType(constants.NotaCreditoElectronica))
	}

	b.AddIssuer().
		AddReceiver().
		AddItems().
		AddSummary().
		AddRelatedDocuments()

	if b.err != nil {
		return nil, b.err
	}

	baseDoc, err := b.BuildWithoutValidation()
	if err != nil {
		return nil, err
	}

	creditNote := &credit_note_models.CreditNoteModel{
		DTEDocument: baseDoc,
		CreditItems: make([]credit_note_models.CreditNoteItem, 0),
		CreditSummary: credit_note_models.CreditNoteSummary{
			Summary: baseDoc.GetSummary().(*models.Summary),
		},
	}

	for _, item := range baseDoc.GetItems() {
		baseItem, ok := item.(*models.Item)

		if ok {
			creditNoteItem := credit_note_models.CreditNoteItem{
				Item: baseItem,
			}

			taxedAmount := baseItem.GetQuantity() * baseItem.GetUnitPrice() * (1 - baseItem.GetDiscount()/100)

			amountObj, err := financial.NewAmount(taxedAmount)
			if err != nil {
				return nil, err
			}
			creditNoteItem.TaxedSale = *amountObj

			zeroAmountObj, err := financial.NewAmount(0)
			if err != nil {
				return nil, err
			}
			creditNoteItem.NonSubjectSale = *zeroAmountObj
			creditNoteItem.ExemptSale = *zeroAmountObj
			creditNote.CreditItems = append(creditNote.CreditItems, creditNoteItem)
		}
	}

	zeroAmountObj, err := financial.NewAmount(0)
	if err != nil {
		return nil, err
	}
	creditNote.CreditSummary.TaxedDiscount = *zeroAmountObj
	creditNote.CreditSummary.IVAPerception = *zeroAmountObj
	creditNote.CreditSummary.IVARetention = *zeroAmountObj
	creditNote.CreditSummary.IncomeRetention = *zeroAmountObj

	baseDTE := creditNote.DTEDocument
	err = baseDTE.Validate()
	if err != nil {
		return nil, err
	}

	dteErr := baseDTE.ValidateDTERules()
	if dteErr != nil {
		return nil, dteErr
	}

	return creditNote, nil
}

// BuildInvalidCreditNote builds an invalid credit note (without related documents)
func (b *DTEBuilder) BuildInvalidCreditNote() (*credit_note_models.CreditNoteModel, error) {
	b.AddIdentification()

	identification, ok := b.document.GetIdentification().(*models.Identification)
	if ok && identification != nil {
		b.setError(identification.SetDTEType(constants.NotaCreditoElectronica))
	}

	b.AddIssuer().
		AddReceiver().
		AddItems().
		AddSummary()

	if b.err != nil {
		return nil, b.err
	}

	baseDoc, err := b.BuildWithoutValidation()
	if err != nil {
		return nil, err
	}

	creditNote := &credit_note_models.CreditNoteModel{
		DTEDocument: baseDoc,
		CreditItems: make([]credit_note_models.CreditNoteItem, 0),
		CreditSummary: credit_note_models.CreditNoteSummary{
			Summary: baseDoc.GetSummary().(*models.Summary),
		},
	}

	for _, item := range baseDoc.GetItems() {
		baseItem, ok := item.(*models.Item)
		if ok {
			creditNoteItem := credit_note_models.CreditNoteItem{
				Item: baseItem,
			}

			taxedAmount := baseItem.GetQuantity() * baseItem.GetUnitPrice() * (1 - baseItem.GetDiscount()/100)

			amountObj, err := financial.NewAmount(taxedAmount)
			if err != nil {
				return nil, err
			}
			creditNoteItem.TaxedSale = *amountObj

			zeroAmountObj, err := financial.NewAmount(0)
			if err != nil {
				return nil, err
			}
			creditNoteItem.NonSubjectSale = *zeroAmountObj
			creditNoteItem.ExemptSale = *zeroAmountObj

			creditNote.CreditItems = append(creditNote.CreditItems, creditNoteItem)
		}
	}

	zeroAmountObj, err := financial.NewAmount(0)
	if err != nil {
		return nil, err
	}
	creditNote.CreditSummary.TaxedDiscount = *zeroAmountObj
	creditNote.CreditSummary.IVAPerception = *zeroAmountObj
	creditNote.CreditSummary.IVARetention = *zeroAmountObj
	creditNote.CreditSummary.IncomeRetention = *zeroAmountObj

	return creditNote, nil
}

// BuildRetentionDocumentWithPhysicalItems builds a valid retention document with physical items
func (b *DTEBuilder) BuildRetentionDocumentWithPhysicalItems() (*retention_models.RetentionModel, error) {
	b.AddIdentification()

	identification, ok := b.document.GetIdentification().(*models.Identification)
	if ok && identification != nil {
		b.setError(identification.SetDTEType(constants.ComprobanteRetencionElectronico))
	}

	b.AddIssuer().
		AddReceiver()

	if b.err != nil {
		return nil, b.err
	}

	baseDoc, err := b.BuildWithoutValidation()
	if err != nil {
		return nil, err
	}

	retentionDoc := &retention_models.RetentionModel{
		DTEDocument:      baseDoc,
		RetentionItems:   make([]retention_models.RetentionItem, 0),
		RetentionSummary: &retention_models.RetentionSummary{},
	}

	err = b.addPhysicalRetentionItems(retentionDoc)
	if err != nil {
		return nil, err
	}

	err = b.createRetentionSummary(retentionDoc)
	if err != nil {
		return nil, err
	}

	baseDTE := retentionDoc.DTEDocument
	err = baseDTE.Validate()
	if err != nil {
		return nil, err
	}

	dteErr := baseDTE.ValidateDTERules()
	if dteErr != nil {
		return nil, dteErr
	}

	return retentionDoc, nil
}

// BuildRetentionDocumentWithElectronicItems builds a retention document with electronic items
func (b *DTEBuilder) BuildRetentionDocumentWithElectronicItems() (*retention_models.RetentionModel, error) {
	b.AddIdentification()

	identification, ok := b.document.GetIdentification().(*models.Identification)
	if ok && identification != nil {
		b.setError(identification.SetDTEType(constants.ComprobanteRetencionElectronico))
	}

	b.AddIssuer().
		AddReceiver()

	if b.err != nil {
		return nil, b.err
	}

	baseDoc, err := b.BuildWithoutValidation()
	if err != nil {
		return nil, err
	}

	retentionDoc := &retention_models.RetentionModel{
		DTEDocument:      baseDoc,
		RetentionItems:   make([]retention_models.RetentionItem, 0),
		RetentionSummary: &retention_models.RetentionSummary{},
	}

	err = b.addElectronicRetentionItems(retentionDoc)
	if err != nil {
		return nil, err
	}

	zeroAmount, err := financial.NewAmount(0)
	if err != nil {
		return nil, err
	}
	retentionDoc.RetentionSummary.TotalIVARetention = *zeroAmount
	retentionDoc.RetentionSummary.TotalSubjectRetention = *zeroAmount

	baseDTE := retentionDoc.DTEDocument
	err = baseDTE.Validate()
	if err != nil {
		return nil, err
	}

	dteErr := baseDTE.ValidateDTERules()
	if dteErr != nil {
		return nil, dteErr
	}

	return retentionDoc, nil
}

// BuildRetentionDocumentWithMixedItems builds a retention document with mixed items (physical and electronic)
func (b *DTEBuilder) BuildRetentionDocumentWithMixedItems() (*retention_models.RetentionModel, error) {
	b.AddIdentification()

	identification, ok := b.document.GetIdentification().(*models.Identification)
	if ok && identification != nil {
		b.setError(identification.SetDTEType(constants.ComprobanteRetencionElectronico))
	}

	b.AddIssuer().
		AddReceiver()

	if b.err != nil {
		return nil, b.err
	}

	baseDoc, err := b.BuildWithoutValidation()
	if err != nil {
		return nil, err
	}

	retentionDoc := &retention_models.RetentionModel{
		DTEDocument:      baseDoc,
		RetentionItems:   make([]retention_models.RetentionItem, 0),
		RetentionSummary: &retention_models.RetentionSummary{},
	}

	err = b.addPhysicalRetentionItems(retentionDoc)
	if err != nil {
		return nil, err
	}
	err = b.addElectronicRetentionItems(retentionDoc)
	if err != nil {
		return nil, err
	}

	err = b.createRetentionSummary(retentionDoc)
	if err != nil {
		return nil, err
	}

	return retentionDoc, nil
}

// BuildInvalidRetentionDocument builds an invalid retention document with physical items without a summary
func (b *DTEBuilder) BuildInvalidRetentionDocument() (*retention_models.RetentionModel, error) {
	b.AddIdentification()

	identification, ok := b.document.GetIdentification().(*models.Identification)
	if ok && identification != nil {
		b.setError(identification.SetDTEType(constants.ComprobanteRetencionElectronico))
	}

	b.AddIssuer().
		AddReceiver()

	if b.err != nil {
		return nil, b.err
	}

	baseDoc, err := b.BuildWithoutValidation()
	if err != nil {
		return nil, err
	}

	retentionDoc := &retention_models.RetentionModel{
		DTEDocument:      baseDoc,
		RetentionItems:   make([]retention_models.RetentionItem, 0),
		RetentionSummary: &retention_models.RetentionSummary{},
	}

	err = b.addPhysicalRetentionItems(retentionDoc)
	if err != nil {
		return nil, err
	}

	zeroAmount, err := financial.NewAmount(0)
	if err != nil {
		return nil, err
	}
	retentionDoc.RetentionSummary.TotalIVARetention = *zeroAmount
	retentionDoc.RetentionSummary.TotalSubjectRetention = *zeroAmount

	return retentionDoc, nil
}

// addPhysicalRetentionItems adds physical items to the retention document
func (b *DTEBuilder) addPhysicalRetentionItems(retentionDoc *retention_models.RetentionModel) error {
	for i := 1; i <= 2; i++ {
		taxedAmount, err := financial.NewAmount(115.25 * float64(i))
		if err != nil {
			return err
		}

		ivaAmount, err := financial.NewAmount(15.00 * float64(i))
		if err != nil {
			return err
		}

		emissionDate, err := temporal.NewEmissionDate(utils.TimeNow().Add(-time.Hour * 24 * 30 * time.Duration(i)))
		if err != nil {
			return err
		}

		dteType, err := document.NewDTEType(constants.CCFElectronico)
		if err != nil {
			return err
		}

		docType, err := document.NewOperationType(constants.PhysicalDocument)
		if err != nil {
			return err
		}

		docNumber, err := document.NewDocumentNumber(fmt.Sprintf("S221001%d", 340+i), constants.PhysicalDocument)
		if err != nil {
			return err
		}

		retentionCode, err := document.NewRetentionCode(constants.RetentionOnePercent)
		if err != nil {
			return err
		}

		retentionItem := retention_models.RetentionItem{
			Number:          *item.NewValidatedItemNumber(i),
			DocumentType:    *docType,
			DocumentNumber:  docNumber,
			Description:     fmt.Sprintf("Compra de suministros de oficina %d", i),
			RetentionAmount: *taxedAmount,
			RetentionIVA:    *ivaAmount,
			EmissionDate:    *emissionDate,
			DTEType:         *dteType,
			ReceptionCodeMH: *retentionCode,
		}

		retentionDoc.RetentionItems = append(retentionDoc.RetentionItems, retentionItem)
	}

	return nil
}

// addElectronicRetentionItems adds electronic items to the retention document
func (b *DTEBuilder) addElectronicRetentionItems(retentionDoc *retention_models.RetentionModel) error {
	startIdx := len(retentionDoc.RetentionItems) + 1

	for i := 0; i < 2; i++ {
		idx := startIdx + i

		docType, err := document.NewOperationType(constants.ElectronicDocument)
		if err != nil {
			return err
		}

		docNumber, err := document.NewDocumentNumber(fmt.Sprintf("FF54E9DB-79C3-42CE-B432-EC522C97EFB%d", i), constants.ElectronicDocument)
		if err != nil {
			return err
		}

		dteType, err := document.NewDTEType(constants.CCFElectronico)
		if err != nil {
			return err
		}

		retentionCode, err := document.NewRetentionCode(constants.RetentionThirteenPercent)
		if err != nil {
			return err
		}

		retentionItem := retention_models.RetentionItem{
			Number:          *item.NewValidatedItemNumber(idx),
			DTEType:         *dteType,
			DocumentType:    *docType,
			DocumentNumber:  docNumber,
			Description:     fmt.Sprintf("Servicio de consultoria electrónica %d", i+1),
			ReceptionCodeMH: *retentionCode,
		}

		retentionDoc.RetentionItems = append(retentionDoc.RetentionItems, retentionItem)
	}

	return nil
}

// createRetentionSummary creates and assigns a summary for the retention document
func (b *DTEBuilder) createRetentionSummary(retentionDoc *retention_models.RetentionModel) error {
	var totalSubjectRetention float64
	var totalIVARetention float64

	for _, item := range retentionDoc.RetentionItems {
		if item.DocumentType.GetValue() == constants.PhysicalDocument {
			totalSubjectRetention += item.RetentionAmount.GetValue()
			totalIVARetention += item.RetentionIVA.GetValue()
		}
	}

	totalSubject, err := financial.NewAmount(totalSubjectRetention)
	if err != nil {
		return err
	}

	totalIVA, err := financial.NewAmount(totalIVARetention)
	if err != nil {
		return err
	}

	retentionDoc.RetentionSummary.TotalSubjectRetention = *totalSubject
	retentionDoc.RetentionSummary.TotalIVARetention = *totalIVA

	return nil
}

// BuildInvalidationDocumentWithReplacement builds a type 1 invalidation document (with replacement)
func (b *DTEBuilder) BuildInvalidationDocumentWithReplacement() (*invalidation_models.InvalidationDocument, error) {
	if b.err != nil {
		return nil, b.err
	}

	b.AddIdentification()

	identificationModel, ok := b.document.GetIdentification().(*models.Identification)
	if ok && identificationModel != nil {
		b.setError(identificationModel.SetDTEType(constants.FacturaElectronica))
	}

	b.AddIssuer()

	invalidationDoc := &invalidation_models.InvalidationDocument{}

	invalidationDoc.Identification = identificationModel
	invalidationDoc.Issuer = b.document.GetIssuer().(*models.Issuer)

	invalidatedDoc, err := b.createInvalidatedDocument()
	if err != nil {
		return nil, err
	}

	replacementCode, err := identification.NewGenerationCode()
	if err != nil {
		return nil, err
	}
	invalidatedDoc.ReplacementCode = replacementCode

	invalidationDoc.Document = invalidatedDoc

	reason, err := b.createInvalidationReason(1)
	if err != nil {
		return nil, err
	}
	invalidationDoc.Reason = reason

	return invalidationDoc, nil
}

// BuildInvalidationDocumentWithAnnulment builds a type 2 invalidation document (annulment)
func (b *DTEBuilder) BuildInvalidationDocumentWithAnnulment() (*invalidation_models.InvalidationDocument, error) {
	if b.err != nil {
		return nil, b.err
	}

	b.AddIdentification()

	identificationModel, ok := b.document.GetIdentification().(*models.Identification)
	if ok && identificationModel != nil {
		b.setError(identificationModel.SetDTEType(constants.FacturaElectronica))
	}

	b.AddIssuer()

	invalidationDoc := &invalidation_models.InvalidationDocument{}

	invalidationDoc.Identification = identificationModel
	invalidationDoc.Issuer = b.document.GetIssuer().(*models.Issuer)

	invalidatedDoc, err := b.createInvalidatedDocument()
	if err != nil {
		return nil, err
	}

	invalidatedDoc.ReplacementCode = nil

	invalidationDoc.Document = invalidatedDoc

	reason, err := b.createInvalidationReason(2)
	if err != nil {
		return nil, err
	}
	invalidationDoc.Reason = reason

	return invalidationDoc, nil
}

// BuildInvalidationDocumentWithDefinitive builds a type 3 invalidation document (definitive)
func (b *DTEBuilder) BuildInvalidationDocumentWithDefinitive() (*invalidation_models.InvalidationDocument, error) {
	if b.err != nil {
		return nil, b.err
	}

	b.AddIdentification()

	identificationModel, ok := b.document.GetIdentification().(*models.Identification)
	if ok && identificationModel != nil {
		b.setError(identificationModel.SetDTEType(constants.FacturaElectronica))
	}

	b.AddIssuer()

	invalidationDoc := &invalidation_models.InvalidationDocument{}

	invalidationDoc.Identification = identificationModel
	invalidationDoc.Issuer = b.document.GetIssuer().(*models.Issuer)

	invalidatedDoc, err := b.createInvalidatedDocument()
	if err != nil {
		return nil, err
	}

	replacementCode, err := identification.NewGenerationCode()
	if err != nil {
		return nil, err
	}
	invalidatedDoc.ReplacementCode = replacementCode

	invalidationDoc.Document = invalidatedDoc

	reason, err := b.createInvalidationReason(3)
	if err != nil {
		return nil, err
	}

	invalidReason := "Documento con errores graves que impiden su utilización"
	validatedReason, err := document.NewInvalidationReason(invalidReason)
	if err != nil {
		return nil, err
	}
	reason.Reason = validatedReason

	invalidationDoc.Reason = reason

	return invalidationDoc, nil
}

// BuildInvalidInvalidationDocument builds an invalid invalidation document (type 2 with replacement code)
func (b *DTEBuilder) BuildInvalidInvalidationDocument() (*invalidation_models.InvalidationDocument, error) {
	if b.err != nil {
		return nil, b.err
	}

	b.AddIdentification()

	identificationModel, ok := b.document.GetIdentification().(*models.Identification)
	if ok && identificationModel != nil {
		b.setError(identificationModel.SetDTEType(constants.FacturaElectronica))
	}

	b.AddIssuer()

	invalidationDoc := &invalidation_models.InvalidationDocument{}

	invalidationDoc.Identification = identificationModel
	invalidationDoc.Issuer = b.document.GetIssuer().(*models.Issuer)

	invalidatedDoc, err := b.createInvalidatedDocument()
	if err != nil {
		return nil, err
	}

	replacementCode, err := identification.NewGenerationCode()
	if err != nil {
		return nil, err
	}
	invalidatedDoc.ReplacementCode = replacementCode

	invalidationDoc.Document = invalidatedDoc

	reason, err := b.createInvalidationReason(2)
	if err != nil {
		return nil, err
	}
	invalidationDoc.Reason = reason

	return invalidationDoc, nil
}

// createInvalidatedDocument creates an invalidated document for use in the invalidation builders
func (b *DTEBuilder) createInvalidatedDocument() (*invalidation_models.InvalidatedDocument, error) {
	docType, err := document.NewDTEType(constants.FacturaElectronica)
	if err != nil {
		return nil, err
	}

	generationCode, err := identification.NewGenerationCode()
	if err != nil {
		return nil, err
	}

	controlNumber, err := identification.NewControlNumber("DTE-01-00000000-000000000000001")
	if err != nil {
		return nil, err
	}

	receptionStamp := "2025AAFEEE1A566A44F19A622C0C35C8A1B6FAZM"

	emissionDate, err := temporal.NewEmissionDate(time.Now().Add(-24 * time.Hour))
	if err != nil {
		return nil, err
	}

	ivaAmount, err := financial.NewAmount(13.00)
	if err != nil {
		return nil, err
	}

	documentType, err := document.NewDTEType(constants.DUI)
	if err != nil {
		return nil, err
	}

	documentNumber, err := identification.NewDocumentNumber("01234567-8", constants.DUI)
	if err != nil {
		return nil, err
	}

	email, err := base.NewEmail("cliente@example.com")
	if err != nil {
		return nil, err
	}

	phone, err := base.NewPhone("22123456")
	if err != nil {
		return nil, err
	}

	name := "Cliente Ejemplo S.A. de C.V."

	return &invalidation_models.InvalidatedDocument{
		Type:           *docType,
		GenerationCode: *generationCode,
		ControlNumber:  *controlNumber,
		ReceptionStamp: receptionStamp,
		EmissionDate:   *emissionDate,
		IVAAmount:      ivaAmount,
		DocumentType:   documentType,
		DocumentNumber: documentNumber,
		Email:          email,
		Phone:          phone,
		Name:           &name,
	}, nil
}

// createInvalidationReason creates an invalidation reason for use in the invalidation builders
func (b *DTEBuilder) createInvalidationReason(invalidationType int) (*invalidation_models.InvalidationReason, error) {
	invalidationTypeObj, err := document.NewInvalidationType(invalidationType)
	if err != nil {
		return nil, err
	}

	responsibleDocType, err := document.NewDTETypeForReceiver(constants.DUI)
	if err != nil {
		return nil, err
	}

	responsibleDocNum, err := identification.NewDocumentNumber("01234567-8", constants.DUI)
	if err != nil {
		return nil, err
	}

	requestorDocType, err := document.NewDTETypeForReceiver(constants.DUI)
	if err != nil {
		return nil, err
	}

	requestorDocNum, err := identification.NewDocumentNumber("98765432-1", constants.DUI)
	if err != nil {
		return nil, err
	}

	return &invalidation_models.InvalidationReason{
		Type:               *invalidationTypeObj,
		ResponsibleName:    "Juan Responsable",
		ResponsibleDocType: *responsibleDocType,
		ResponsibleDocNum:  *responsibleDocNum,
		RequesterName:      "Ana Solicitante",
		RequesterDocType:   *requestorDocType,
		RequesterDocNum:    *requestorDocNum,
	}, nil
}
