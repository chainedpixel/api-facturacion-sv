package remission_note

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/remission_note/remission_note_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

// MapRemissionNoteIssuer converts the domain issuer to the Ministry of Finance format
func MapRemissionNoteIssuer(model *remission_note_models.RemissionNoteModel) *structs.DTEIssuer {
	if model.Issuer == nil {
		return nil
	}

	result := structs.DTEIssuer{
		NIT:                 model.Issuer.GetNIT(),
		NRC:                 model.Issuer.GetNRC(),
		Nombre:              model.Issuer.GetName(),
		CodActividad:        model.Issuer.GetActivityCode(),
		DescActividad:       model.Issuer.GetActivityDescription(),
		TipoEstablecimiento: model.Issuer.GetEstablishmentType(),
		Direccion:           common.MapCommonResponseAddress(model.Issuer.GetAddress()),
		Telefono:            model.Issuer.GetPhone(),
		Correo:              model.Issuer.GetEmail(),
	}

	if name := model.Issuer.GetCommercialName(); name != "" {
		result.NombreComercial = &name
	}
	if code := model.Issuer.GetEstablishmentCode(); code != nil {
		result.CodEstable = code
	}
	if code := model.Issuer.GetEstablishmentMHCode(); code != nil {
		result.CodEstableMH = code
	}
	if code := model.Issuer.GetPOSCode(); code != nil {
		result.CodPuntoVenta = code
	}
	if code := model.Issuer.GetPOSMHCode(); code != nil {
		result.CodPuntoVentaMH = code
	}

	return &result
}

// MapRemissionNoteIdentification converts the domain identification to the Ministry of Finance format
func MapRemissionNoteIdentification(model *remission_note_models.RemissionNoteModel) *structs.DTEIdentification {
	if model.Identification == nil {
		return nil
	}

	return &structs.DTEIdentification{
		Version:          model.Identification.GetVersion(),
		Ambiente:         model.Identification.GetAmbient(),
		TipoDte:          model.Identification.GetDTEType(),
		NumeroControl:    model.Identification.GetControlNumber(),
		CodigoGeneracion: model.Identification.GetGenerationCode(),
		TipoModelo:       model.Identification.GetModelType(),
		TipoOperacion:    model.Identification.GetOperationType(),
		FecEmi:           model.Identification.GetEmissionDate().Format("2006-01-02"),
		HorEmi:           model.Identification.GetEmissionTime().Format("15:04:05"),
		TipoMoneda:       model.Identification.GetCurrency(),
	}
}

// MapRemissionNoteRelatedDocuments converts the domain related documents to the Ministry of Finance format
func MapRemissionNoteRelatedDocuments(model *remission_note_models.RemissionNoteModel) []structs.DTERelatedDocument {
	if model.RelatedDocuments == nil || len(model.RelatedDocuments) == 0 {
		return nil
	}

	return common.MapCommonResponseRelatedDocuments(model.RelatedDocuments)
}

// MapRemissionNoteThirdPartySale converts the domain third-party sales to the Ministry of Finance format
func MapRemissionNoteThirdPartySale(model *remission_note_models.RemissionNoteModel) *structs.DTEThirdPartySale {
	if model.ThirdPartySale == nil {
		return nil
	}

	return common.MapCommonResponseThirdPartySale(model.ThirdPartySale)
}

// MapRemissionNoteAppendix converts the domain appendixes to the Ministry of Finance format
func MapRemissionNoteAppendix(model *remission_note_models.RemissionNoteModel) []structs.DTEApendice {
	if model.Appendix == nil || len(model.Appendix) == 0 {
		return nil
	}

	return common.MapCommonResponseAppendix(model.Appendix)
}
