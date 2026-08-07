package fixtures

import (
	"fmt"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/ccf/ccf_models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
)

type CCFBuilder struct {
	baseBuilder *DTEBuilder
	document    *ccf_models.CreditFiscalDocument
	err         error
}

func NewCCFBuilder() *CCFBuilder {
	return &CCFBuilder{
		baseBuilder: NewDTEBuilder(),
		document: &ccf_models.CreditFiscalDocument{
			DTEDocument: &models.DTEDocument{},
			CreditItems: make([]ccf_models.CreditItem, 0),
		},
		err: nil,
	}
}

func (b *CCFBuilder) Document() *ccf_models.CreditFiscalDocument {
	return b.document
}

func (b *CCFBuilder) Build() (*ccf_models.CreditFiscalDocument, error) {
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

func (b *CCFBuilder) BuildWithoutValidation() (*ccf_models.CreditFiscalDocument, error) {
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

func (b *CCFBuilder) setError(err error) *CCFBuilder {
	if b.err == nil && err != nil {
		b.err = err
	}

	if err != nil {
		b.baseBuilder.setError(err)
	}

	return b
}

func (b *CCFBuilder) AddIdentification() *CCFBuilder {
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

	b.setError(baseIdentification.SetDTEType(constants.CCFElectronico))
	b.document.Identification = baseIdentification

	controlNumber := baseIdentification.GetControlNumber()
	if len(controlNumber) > 4 {
		newControlNumber := "DTE-03" + controlNumber[6:]
		b.setError(b.document.Identification.SetControlNumber(newControlNumber))
	}

	return b
}

func (b *CCFBuilder) AddIssuer() *CCFBuilder {
	b.baseBuilder.AddIssuer()

	if b.baseBuilder.err != nil {
		b.err = b.baseBuilder.err
	}

	return b
}

// AddReceiverForCCF adds valid receiver data for CCF
func (b *CCFBuilder) AddReceiverForCCF() *CCFBuilder {
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

	nrc := "987654"
	activityDescription := "Purchase of goods and services"
	activityCode := "56789"
	commercialName := "CLIENT COMPANY INC."
	nit := "98765432101234"

	b.setError(baseReceiver.SetNRC(&nrc))
	b.setError(baseReceiver.SetActivityDescription(&activityDescription))
	b.setError(baseReceiver.SetActivityCode(&activityCode))
	b.setError(baseReceiver.SetCommercialName(&commercialName))
	b.setError(baseReceiver.SetNIT(&nit))

	return b
}

// AddReceiverWithNoNRC adds receiver data without NRC (invalid for CCF)
func (b *CCFBuilder) AddReceiverWithNoNRC() *CCFBuilder {
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

	activityDescription := "Purchase of goods and services"
	activityCode := "56789"
	commercialName := "CLIENT COMPANY INC."
	nit := "98765432101234"

	b.setError(baseReceiver.SetActivityDescription(&activityDescription))
	b.setError(baseReceiver.SetActivityCode(&activityCode))
	b.setError(baseReceiver.SetCommercialName(&commercialName))
	b.setError(baseReceiver.SetNIT(&nit))

	return b
}

func (b *CCFBuilder) AddItems() *CCFBuilder {
	b.baseBuilder.AddItems()

	if b.baseBuilder.err != nil {
		b.err = b.baseBuilder.err
		return b
	}

	baseItems := b.baseBuilder.document.GetItems()
	baseItems[0].SetTaxes([]string{constants.TaxIVA})
	creditItems := make([]ccf_models.CreditItem, 0, len(baseItems))

	for _, baseItem := range baseItems {
		item, ok := baseItem.(*models.Item)
		if !ok {
			b.setError(fmt.Errorf("failed to convert item to proper type"))
			return b
		}

		creditItem := ccf_models.CreditItem{
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
		creditItem.SuggestedPrice = *zeroAmount
		creditItem.NonTaxed = *zeroAmount

		creditItems = append(creditItems, creditItem)
	}

	b.document.CreditItems = creditItems

	return b
}

func (b *CCFBuilder) AddSummary() *CCFBuilder {
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

	creditSummary := ccf_models.CreditSummary{
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
	creditSummary.BalanceInFavor = *zeroAmount
	creditSummary.IncomeRetention = *zeroAmount

	b.document.CreditSummary = creditSummary

	return b
}

func (b *CCFBuilder) AddSummaryWithCreditOperation() *CCFBuilder {
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

func (b *CCFBuilder) AddSummaryWithCashOperation() *CCFBuilder {
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

	b.setError(baseSummary.SetOperationCondition(constants.Cash))

	return b
}

// AddSummaryWithZeroTaxes adds a summary with zeros for NRC validation testing
func (b *CCFBuilder) AddSummaryWithZeroTaxes() *CCFBuilder {
	baseSummary := &models.Summary{}

	b.setError(baseSummary.SetTotalNonSubject(0.00))
	b.setError(baseSummary.SetTotalExempt(0.00))
	b.setError(baseSummary.SetTotalTaxed(0.00))
	b.setError(baseSummary.SetSubTotal(0.00))
	b.setError(baseSummary.SetSubtotalSales(0.00))
	b.setError(baseSummary.SetNonSubjectDiscount(0.00))
	b.setError(baseSummary.SetExemptDiscount(0.00))
	b.setError(baseSummary.SetDiscountPercentage(0.00))
	b.setError(baseSummary.SetTotalDiscount(0.00))
	b.setError(baseSummary.SetTotalOperation(0.00))
	b.setError(baseSummary.SetTotalNotTaxed(0.00))
	b.setError(baseSummary.SetOperationCondition(constants.Cash))
	b.setError(baseSummary.SetTotalToPay(0.00))
	b.setError(baseSummary.SetTotalInWords("CERO DOLARES CON 00/100 CENTAVOS"))

	payment := &models.PaymentType{}
	b.setError(payment.SetCode(constants.BilletesMonedas))
	b.setError(payment.SetAmount(0.00))
	b.setError(payment.SetReference("Cash payment"))

	var payments []interfaces.PaymentType
	if b.err == nil {
		payments = append(payments, payment)
		b.setError(baseSummary.SetPaymentTypes(payments))
	}

	if b.err == nil {
		b.setError(b.baseBuilder.document.SetSummary(baseSummary))
	}

	creditSummary := ccf_models.CreditSummary{
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
	creditSummary.BalanceInFavor = *zeroAmount
	creditSummary.IncomeRetention = *zeroAmount

	b.document.CreditSummary = creditSummary

	return b
}

func BuildValidCCF() (*ccf_models.CreditFiscalDocument, error) {
	builder := NewCCFBuilder()

	builder.AddIdentification().
		AddIssuer().
		AddReceiverForCCF().
		AddItems().
		AddSummary()

	return builder.Build()
}

func BuildInvalidCCFWithoutNRC() (*ccf_models.CreditFiscalDocument, error) {
	builder := NewCCFBuilder()

	builder.AddIdentification().
		AddIssuer().
		AddReceiverWithNoNRC().
		AddItems().
		AddSummaryWithZeroTaxes()

	return builder.BuildWithoutValidation()
}

func (b *CCFBuilder) BuildAsCCFData() (*ccf_models.CCFData, error) {
	if b == nil {
		return nil, fmt.Errorf("CCFBuilder is nil")
	}

	if b.err != nil {
		return nil, b.err
	}

	ccf := b.document
	var appendixes []models.Appendix
	for _, app := range ccf.Appendix {
		if a, ok := app.(*models.Appendix); ok {
			appendixes = append(appendixes, *a)
		}
	}

	var relatedDocs []models.RelatedDocument
	for _, rd := range ccf.RelatedDocuments {
		if r, ok := rd.(*models.RelatedDocument); ok {
			relatedDocs = append(relatedDocs, *r)
		}
	}

	var otherDocs []models.OtherDocument
	for _, od := range ccf.OtherDocuments {
		if o, ok := od.(*models.OtherDocument); ok {
			otherDocs = append(otherDocs, *o)
		}
	}

	var thirdPartySale *models.ThirdPartySale
	if ccf.ThirdPartySale != nil {
		if t, ok := ccf.ThirdPartySale.(*models.ThirdPartySale); ok {
			thirdPartySale = t
		}
	}

	var extension *models.Extension
	if ccf.Extension != nil {
		if e, ok := ccf.Extension.(*models.Extension); ok {
			extension = e
		}
	}

	return &ccf_models.CCFData{
		InputDataCommon: &models.InputDataCommon{
			Identification: ccf.GetIdentification().(*models.Identification),
			Issuer:         ccf.GetIssuer().(*models.Issuer),
			Receiver:       ccf.GetReceiver().(*models.Receiver),
			Extension:      extension,
			RelatedDocs:    relatedDocs,
			OtherDocs:      otherDocs,
			ThirdPartySale: thirdPartySale,
			Appendixes:     appendixes,
		},
		Items:         ccf.CreditItems,
		CreditSummary: &ccf.CreditSummary,
	}, nil
}

func BuildAsCCFData(ccf *ccf_models.CreditFiscalDocument) *ccf_models.CCFData {

	var appendixes []models.Appendix
	for _, app := range ccf.Appendix {
		if a, ok := app.(*models.Appendix); ok {
			appendixes = append(appendixes, *a)
		}
	}

	var relatedDocs []models.RelatedDocument
	for _, rd := range ccf.RelatedDocuments {
		if r, ok := rd.(*models.RelatedDocument); ok {
			relatedDocs = append(relatedDocs, *r)
		}
	}

	var otherDocs []models.OtherDocument
	for _, od := range ccf.OtherDocuments {
		if o, ok := od.(*models.OtherDocument); ok {
			otherDocs = append(otherDocs, *o)
		}
	}

	var thirdPartySale *models.ThirdPartySale
	if ccf.ThirdPartySale != nil {
		if t, ok := ccf.ThirdPartySale.(*models.ThirdPartySale); ok {
			thirdPartySale = t
		}
	}

	var extension *models.Extension
	if ccf.Extension != nil {
		if e, ok := ccf.Extension.(*models.Extension); ok {
			extension = e
		}
	}

	return &ccf_models.CCFData{
		InputDataCommon: &models.InputDataCommon{
			Identification: ccf.GetIdentification().(*models.Identification),
			Issuer:         ccf.GetIssuer().(*models.Issuer),
			Receiver:       ccf.GetReceiver().(*models.Receiver),
			Extension:      extension,
			RelatedDocs:    relatedDocs,
			OtherDocs:      otherDocs,
			ThirdPartySale: thirdPartySale,
			Appendixes:     appendixes,
		},
		Items:         ccf.CreditItems,
		CreditSummary: &ccf.CreditSummary,
	}
}

func BuildCCF() (*ccf_models.CreditFiscalDocument, error) {
	builder := NewCCFBuilder()

	builder.AddIdentification().
		AddIssuer().
		AddReceiverForCCF().
		AddItems().
		AddSummary()

	return builder.Build()
}

func BuildCCFWithMixedItemsType() (*ccf_models.CreditFiscalDocument, error) {
	builder := NewCCFBuilder()

	builder.AddIdentification().
		AddIssuer().
		AddReceiverForCCF().
		AddItems().
		AddSummary()

	ccf, err := builder.Build()
	if err != nil {
		return nil, err
	}

	var creditItems []ccf_models.CreditItem
	for i, baseItem := range ccf.GetItems() {
		if item, ok := baseItem.(*models.Item); ok {
			if i%2 == 0 {
				item.SetType(constants.Producto)
			} else {
				item.SetType(constants.Servicio)
			}

			creditItem := ccf_models.CreditItem{
				Item: item,
			}

			taxedAmount, _ := financial.NewAmount(item.GetQuantity() * item.GetUnitPrice() * (1 - item.GetDiscount()/100))
			creditItem.TaxedSale = *taxedAmount

			zeroAmount, _ := financial.NewAmount(0)
			creditItem.NonSubjectSale = *zeroAmount
			creditItem.ExemptSale = *zeroAmount
			creditItem.SuggestedPrice = *zeroAmount
			creditItem.NonTaxed = *zeroAmount

			creditItems = append(creditItems, creditItem)
		}
	}
	ccf.CreditItems = creditItems

	return ccf, nil
}
