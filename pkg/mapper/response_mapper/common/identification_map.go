package common

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

// MapCommonResponseIdentification maps the identification of an electronic invoice to an identification model -> Source: Response
func MapCommonResponseIdentification(identification interfaces.Identification) *structs.DTEIdentification {
	return &structs.DTEIdentification{
		Version:          identification.GetVersion(),
		Ambiente:         identification.GetAmbient(),
		TipoDte:          identification.GetDTEType(),
		NumeroControl:    identification.GetControlNumber(),
		CodigoGeneracion: identification.GetGenerationCode(),
		TipoModelo:       identification.GetModelType(),
		TipoOperacion:    identification.GetOperationType(),
		FecEmi:           identification.GetEmissionDate().Format("2006-01-02"),
		HorEmi:           identification.GetEmissionTime().Format("15:04:05"),
		TipoMoneda:       identification.GetCurrency(),
	}
}
