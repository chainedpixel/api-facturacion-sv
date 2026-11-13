package fixtures

import (
	"fmt"
	"math"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/identification"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/fse/fse_models"
)

type FSEBuilder struct {
	baseBuilder *DTEBuilder
	document    *fse_models.FSEModel
	err         error
}

func NewFSEBuilder() *FSEBuilder {
	return &FSEBuilder{
		baseBuilder: NewDTEBuilder(),
		document:    &fse_models.FSEModel{DTEDocument: &models.DTEDocument{}},
		err:         nil,
	}
}

func (b *FSEBuilder) Document() *fse_models.FSEModel { return b.document }

func (b *FSEBuilder) Build() (*fse_models.FSEModel, error) {
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

func (b *FSEBuilder) BuildWithoutValidation() (*fse_models.FSEModel, error) {
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

func (b *FSEBuilder) setError(err error) *FSEBuilder {
	if b.err == nil && err != nil {
		b.err = err
	}
	if err != nil {
		b.baseBuilder.setError(err)
	}
	return b
}

func (b *FSEBuilder) AddIdentification() *FSEBuilder {
	b.baseBuilder.AddIdentification()
	if b.baseBuilder.err != nil {
		b.err = b.baseBuilder.err
		return b
	}
	baseIdentification, ok := b.baseBuilder.document.GetIdentification().(*models.Identification)
	if !ok || baseIdentification == nil {
		b.setError(fmt.Errorf("failed to get identification"))
		return b
	}
	b.setError(baseIdentification.SetDTEType(constants.FacturaSujetoExcluidoElectronica))
	controlNumber := baseIdentification.GetControlNumber()
	if len(controlNumber) > 4 {
		b.setError(baseIdentification.SetControlNumber("DTE-14" + controlNumber[6:]))
	}
	return b
}

func (b *FSEBuilder) AddIssuer() *FSEBuilder {
	b.baseBuilder.AddIssuer()
	if b.baseBuilder.err != nil {
		b.err = b.baseBuilder.err
	}
	return b
}

func (b *FSEBuilder) AddItems() *FSEBuilder {
	b.baseBuilder.AddItems()
	if b.baseBuilder.err != nil {
		b.err = b.baseBuilder.err
		return b
	}
	baseItems := b.baseBuilder.document.GetItems()
	fseItems := make([]fse_models.FSEItem, 0, len(baseItems))
	for _, baseItem := range baseItems {
		item, ok := baseItem.(*models.Item)
		if !ok {
			b.setError(fmt.Errorf("failed to convert item"))
			return b
		}
		// compra = (precioUni x cantidad) - montoDescu (Anexo IV)
		purchaseAmount, err := financial.NewAmount(item.GetQuantity()*item.GetUnitPrice() - item.GetDiscount())
		if err != nil {
			b.setError(err)
			return b
		}
		fseItems = append(fseItems, fse_models.FSEItem{Item: item, Purchase: *purchaseAmount})
	}
	b.document.FSEItems = fseItems
	return b
}

func (b *FSEBuilder) AddFSEReceiver() *FSEBuilder {
	actCode, err := identification.NewActivityCode("46900")
	if err != nil {
		b.setError(err)
		return b
	}
	docType, err := document.NewDTETypeForReceiver("13")
	if err != nil {
		b.setError(err)
		return b
	}
	docNumber, err := identification.NewDocumentNumber("00000000-0", "13")
	if err != nil {
		b.setError(err)
		return b
	}
	baseReceiver := &models.Receiver{}
	name := "Juan Carlos Pérez"
	phone := "22345678"
	email := "juan@example.com"
	b.setError(baseReceiver.SetName(&name))
	b.setError(baseReceiver.SetPhone(&phone))
	b.setError(baseReceiver.SetEmail(&email))

	// El schema fe-fse-v2 exige la direccion del sujeto excluido
	address := &models.Address{}
	b.setError(address.SetDepartment("06"))
	b.setError(address.SetMunicipality("22"))
	b.setError(address.SetDistrict("01"))
	b.setError(address.SetComplement("Colonia Escalon, Calle El Mirador #123"))
	b.setError(baseReceiver.SetAddress(address))

	actDesc := "Servicios profesionales"
	b.document.FSEReceiver = fse_models.FSEReceiver{
		Receiver:            baseReceiver,
		DocumentType:        *docType,
		DocumentNumber:      *docNumber,
		ActivityCode:        actCode,
		ActivityDescription: &actDesc,
	}
	return b
}

func (b *FSEBuilder) AddSummary() *FSEBuilder {
	b.baseBuilder.AddSummary()
	if b.baseBuilder.err != nil {
		b.err = b.baseBuilder.err
		return b
	}
	baseSummary, ok := b.baseBuilder.document.GetSummary().(*models.Summary)
	if !ok || baseSummary == nil {
		b.setError(fmt.Errorf("failed to get summary"))
		return b
	}
	zeroAmt, err := financial.NewAmount(0.0)
	if err != nil {
		b.setError(err)
		return b
	}
	// El resumen se calcula desde los items (regla 7.3): FSE no tiene ventas gravadas ni IVA,
	// subTotal = totalCompra - descu y totalPagar = subTotal - reteRenta (sin retencion en este fixture)
	var totalPurchase float64
	for _, item := range b.document.FSEItems {
		totalPurchase += item.Purchase.GetValue()
	}
	totalPurchase = math.Round(totalPurchase*100) / 100

	b.setError(baseSummary.SetTotalTaxed(0))
	b.setError(baseSummary.SetSubtotalSales(0))
	b.setError(baseSummary.SetTotalTaxes(nil))
	b.setError(baseSummary.SetSubTotal(totalPurchase))
	b.setError(baseSummary.SetTotalOperation(totalPurchase))
	b.setError(baseSummary.SetTotalToPay(totalPurchase))
	for i, payment := range baseSummary.GetPaymentTypes() {
		if i == 0 {
			b.setError(payment.SetAmount(totalPurchase))
		}
	}

	purchaseAmt, err := financial.NewAmount(totalPurchase)
	if err != nil {
		b.setError(err)
		return b
	}
	b.document.FSESummary = fse_models.FSESummary{
		Summary:         baseSummary,
		TotalPurchase:   *purchaseAmt,
		IVARetention:    *zeroAmt,
		IncomeRetention: *zeroAmt,
	}
	return b
}

// BuildAsFSEData converts an FSEModel to FSEData for service testing.
func BuildAsFSEData(fseDoc *fse_models.FSEModel) *fse_models.FSEData {
	fseSummary := fseDoc.FSESummary
	fseReceiver := fseDoc.FSEReceiver
	return &fse_models.FSEData{
		InputDataCommon: &models.InputDataCommon{
			Identification: fseDoc.GetIdentification().(*models.Identification),
			Issuer:         fseDoc.GetIssuer().(*models.Issuer),
		},
		Items:       fseDoc.FSEItems,
		FSESummary:  &fseSummary,
		FSEReceiver: &fseReceiver,
	}
}

func BuildValidFSE() (*fse_models.FSEModel, error) {
	builder := NewFSEBuilder()
	builder.AddIdentification().
		AddIssuer().
		AddItems().
		AddFSEReceiver().
		AddSummary()
	return builder.Build()
}
