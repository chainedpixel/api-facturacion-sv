package fixtures

import (
	"fmt"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/item"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/temporal"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/retention/retention_models"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

type RetentionBuilder struct {
	baseBuilder *DTEBuilder
	document    *retention_models.RetentionModel
	err         error
}

func NewRetentionBuilder() *RetentionBuilder {
	return &RetentionBuilder{
		baseBuilder: NewDTEBuilder(),
		document: &retention_models.RetentionModel{
			DTEDocument:      &models.DTEDocument{},
			RetentionItems:   make([]retention_models.RetentionItem, 0),
			RetentionSummary: &retention_models.RetentionSummary{},
		},
		err: nil,
	}
}

func (b *RetentionBuilder) Document() *retention_models.RetentionModel {
	return b.document
}

func (b *RetentionBuilder) Build() (*retention_models.RetentionModel, error) {
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

func (b *RetentionBuilder) BuildWithoutValidation() (*retention_models.RetentionModel, error) {
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

func (b *RetentionBuilder) setError(err error) *RetentionBuilder {
	if b.err == nil && err != nil {
		b.err = err
	}

	if err != nil {
		b.baseBuilder.setError(err)
	}

	return b
}

func (b *RetentionBuilder) AddIdentification() *RetentionBuilder {
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

	b.setError(baseIdentification.SetDTEType(constants.ComprobanteRetencionElectronico))

	controlNumber := baseIdentification.GetControlNumber()
	if len(controlNumber) > 4 {
		newControlNumber := "DTE-06" + controlNumber[6:]
		b.setError(baseIdentification.SetControlNumber(newControlNumber))
	}

	return b
}

func (b *RetentionBuilder) AddIssuer() *RetentionBuilder {
	b.baseBuilder.AddIssuer()

	if b.baseBuilder.err != nil {
		b.err = b.baseBuilder.err
	}

	return b
}

func (b *RetentionBuilder) AddReceiver() *RetentionBuilder {
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
	commercialName := "CLIENT COMPANY INC."

	b.setError(baseReceiver.SetNRC(&nrc))
	b.setError(baseReceiver.SetCommercialName(&commercialName))

	return b
}

func (b *RetentionBuilder) AddReceiverForCompany() *RetentionBuilder {
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

	name := "EMPRESA CLIENTE, S.A. DE C.V."
	nrc := "123456-7"
	nit := "0614-010190-101-1"
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

func (b *RetentionBuilder) AddPhysicalItems() *RetentionBuilder {
	if b.err != nil {
		return b
	}

	for i := 1; i <= 2; i++ {
		taxedAmount, err := financial.NewAmount(115.25 * float64(i))
		if err != nil {
			b.setError(err)
			return b
		}

		ivaAmount, err := financial.NewAmount(1.15 * float64(i))
		if err != nil {
			b.setError(err)
			return b
		}

		emissionDate, err := temporal.NewEmissionDate(utils.TimeNow().AddDate(0, 0, -i*5))
		if err != nil {
			b.setError(err)
			return b
		}

		dteType, err := document.NewDTEType(constants.CCFElectronico)
		if err != nil {
			b.setError(err)
			return b
		}

		docType, err := document.NewOperationType(constants.PhysicalDocument)
		if err != nil {
			b.setError(err)
			return b
		}

		docNumber, err := document.NewDocumentNumber(fmt.Sprintf("S221001%d", 340+i), constants.PhysicalDocument)
		if err != nil {
			b.setError(err)
			return b
		}

		retentionCode, err := document.NewRetentionCode(constants.RetentionOnePercent)
		if err != nil {
			b.setError(err)
			return b
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

		b.document.RetentionItems = append(b.document.RetentionItems, retentionItem)
	}

	return b
}

func (b *RetentionBuilder) AddElectronicItems() *RetentionBuilder {
	if b.err != nil {
		return b
	}

	startIdx := len(b.document.RetentionItems) + 1

	for i := 0; i < 2; i++ {
		idx := startIdx + i

		docType, err := document.NewOperationType(constants.ElectronicDocument)
		if err != nil {
			b.setError(err)
			return b
		}

		docNumber, err := document.NewDocumentNumber(fmt.Sprintf("FF54E9DB-79C3-42CE-B432-EC522C97EFB%d", i), constants.ElectronicDocument)
		if err != nil {
			b.setError(err)
			return b
		}

		dteType, err := document.NewDTEType(constants.FacturaElectronica)
		if err != nil {
			b.setError(err)
			return b
		}

		emissionDate, err := temporal.NewEmissionDate(utils.TimeNow().AddDate(0, 0, -i*15))
		if err != nil {
			b.setError(err)
			return b
		}

		retentionCode, err := document.NewRetentionCode(constants.RetentionThirteenPercent)
		if err != nil {
			b.setError(err)
			return b
		}

		zeroAmount, err := financial.NewAmount(0.0)
		if err != nil {
			b.setError(err)
			return b
		}

		retentionItem := retention_models.RetentionItem{
			Number:          *item.NewValidatedItemNumber(idx),
			DTEType:         *dteType,
			DocumentType:    *docType,
			DocumentNumber:  docNumber,
			EmissionDate:    *emissionDate,
			Description:     fmt.Sprintf("Servicio de consultoría electrónica %d", i+1),
			RetentionAmount: *zeroAmount,
			RetentionIVA:    *zeroAmount,
			ReceptionCodeMH: *retentionCode,
		}

		b.document.RetentionItems = append(b.document.RetentionItems, retentionItem)
	}

	return b
}

func (b *RetentionBuilder) AddRetentionSummary() *RetentionBuilder {
	if b.err != nil {
		return b
	}

	var totalSubjectRetention, totalIVARetention float64

	for _, item := range b.document.RetentionItems {
		if item.DocumentType.GetValue() == constants.PhysicalDocument {
			totalSubjectRetention += item.RetentionAmount.GetValue()
			totalIVARetention += item.RetentionIVA.GetValue()
		}
	}

	totalSubject, err := financial.NewAmount(totalSubjectRetention)
	if err != nil {
		b.setError(err)
		return b
	}

	totalIVA, err := financial.NewAmount(totalIVARetention)
	if err != nil {
		b.setError(err)
		return b
	}

	b.document.RetentionSummary.TotalSubjectRetention = *totalSubject
	b.document.RetentionSummary.TotalIVARetention = *totalIVA
	b.document.RetentionSummary.TotalIVARetentionLetters = "QUINCE DOLARES CON 00/100"

	return b
}

func (b *RetentionBuilder) AddInvalidRetentionSummary() *RetentionBuilder {
	if b.err != nil {
		return b
	}

	incorrectTotal, err := financial.NewAmount(999.99)
	if err != nil {
		b.setError(err)
		return b
	}

	b.document.RetentionSummary.TotalSubjectRetention = *incorrectTotal
	b.document.RetentionSummary.TotalIVARetention = *incorrectTotal
	b.document.RetentionSummary.TotalIVARetentionLetters = "NOVECIENTOS NOVENTA Y NUEVE DOLARES CON 99/100"

	return b
}

func (b *RetentionBuilder) AddMixedItems() *RetentionBuilder {
	b.AddPhysicalItems()
	b.AddElectronicItems()
	return b
}

func (b *RetentionBuilder) BuildAsInputRetentionData() (*retention_models.InputRetentionData, error) {
	if b == nil {
		return nil, fmt.Errorf("RetentionBuilder is nil")
	}

	if b.err != nil {
		return nil, b.err
	}

	retention := b.document
	var appendixes []models.Appendix
	for _, app := range retention.Appendix {
		if a, ok := app.(*models.Appendix); ok {
			appendixes = append(appendixes, *a)
		}
	}

	var relatedDocs []models.RelatedDocument
	for _, rd := range retention.RelatedDocuments {
		if r, ok := rd.(*models.RelatedDocument); ok {
			relatedDocs = append(relatedDocs, *r)
		}
	}

	var otherDocs []models.OtherDocument
	for _, od := range retention.OtherDocuments {
		if o, ok := od.(*models.OtherDocument); ok {
			otherDocs = append(otherDocs, *o)
		}
	}

	var thirdPartySale *models.ThirdPartySale
	if retention.ThirdPartySale != nil {
		if t, ok := retention.ThirdPartySale.(*models.ThirdPartySale); ok {
			thirdPartySale = t
		}
	}

	var extension *models.Extension
	if retention.Extension != nil {
		if e, ok := retention.Extension.(*models.Extension); ok {
			extension = e
		}
	}

	return &retention_models.InputRetentionData{
		InputDataCommon: &models.InputDataCommon{
			Identification: retention.GetIdentification().(*models.Identification),
			Issuer:         retention.GetIssuer().(*models.Issuer),
			Receiver:       retention.GetReceiver().(*models.Receiver),
			Extension:      extension,
			RelatedDocs:    relatedDocs,
			OtherDocs:      otherDocs,
			ThirdPartySale: thirdPartySale,
			Appendixes:     appendixes,
		},
		RetentionItems:   retention.RetentionItems,
		RetentionSummary: retention.RetentionSummary,
	}, nil
}

func BuildAsInputRetentionData(retention *retention_models.RetentionModel) *retention_models.InputRetentionData {
	var appendixes []models.Appendix
	for _, app := range retention.Appendix {
		if a, ok := app.(*models.Appendix); ok {
			appendixes = append(appendixes, *a)
		}
	}

	var relatedDocs []models.RelatedDocument
	for _, rd := range retention.RelatedDocuments {
		if r, ok := rd.(*models.RelatedDocument); ok {
			relatedDocs = append(relatedDocs, *r)
		}
	}

	var otherDocs []models.OtherDocument
	for _, od := range retention.OtherDocuments {
		if o, ok := od.(*models.OtherDocument); ok {
			otherDocs = append(otherDocs, *o)
		}
	}

	var thirdPartySale *models.ThirdPartySale
	if retention.ThirdPartySale != nil {
		if t, ok := retention.ThirdPartySale.(*models.ThirdPartySale); ok {
			thirdPartySale = t
		}
	}

	var extension *models.Extension
	if retention.Extension != nil {
		if e, ok := retention.Extension.(*models.Extension); ok {
			extension = e
		}
	}

	return &retention_models.InputRetentionData{
		InputDataCommon: &models.InputDataCommon{
			Identification: retention.GetIdentification().(*models.Identification),
			Issuer:         retention.GetIssuer().(*models.Issuer),
			Receiver:       retention.GetReceiver().(*models.Receiver),
			Extension:      extension,
			RelatedDocs:    relatedDocs,
			OtherDocs:      otherDocs,
			ThirdPartySale: thirdPartySale,
			Appendixes:     appendixes,
		},
		RetentionItems:   retention.RetentionItems,
		RetentionSummary: retention.RetentionSummary,
	}
}

func BuildValidRetention() (*retention_models.RetentionModel, error) {
	builder := NewRetentionBuilder()

	builder.AddIdentification().
		AddIssuer().
		AddReceiver().
		AddPhysicalItems().
		AddRetentionSummary()

	return builder.Build()
}

func BuildRetentionWithElectronicItems() (*retention_models.RetentionModel, error) {
	builder := NewRetentionBuilder()

	builder.AddIdentification().
		AddIssuer().
		AddReceiver().
		AddElectronicItems()

	return builder.Build()
}

func BuildRetentionWithMixedItems() (*retention_models.RetentionModel, error) {
	builder := NewRetentionBuilder()

	builder.AddIdentification().
		AddIssuer().
		AddReceiver().
		AddMixedItems().
		AddRetentionSummary()

	return builder.Build()
}

func BuildInvalidRetentionWithInconsistentSummary() (*retention_models.RetentionModel, error) {
	builder := NewRetentionBuilder()

	builder.AddIdentification().
		AddIssuer().
		AddReceiver().
		AddPhysicalItems().
		AddInvalidRetentionSummary()

	return builder.BuildWithoutValidation()
}

func BuildRetentionWithoutSummary() (*retention_models.RetentionModel, error) {
	builder := NewRetentionBuilder()

	builder.AddIdentification().
		AddIssuer().
		AddReceiver().
		AddPhysicalItems()

	return builder.BuildWithoutValidation()
}

func BuildRetention() (*retention_models.RetentionModel, error) {
	return BuildValidRetention()
}
