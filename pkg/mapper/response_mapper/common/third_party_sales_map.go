package common

import (
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/response_mapper/structs"
)

// MapCommonResponseThirdPartySale maps a third-party sale to a third-party sale model -> Source: Response
func MapCommonResponseThirdPartySale(sale interfaces.ThirdPartySale) *structs.DTEThirdPartySale {
	if sale == nil {
		return nil
	}

	fiscalDomicile := sale.GetFiscalDomicile()
	return &structs.DTEThirdPartySale{
		NIT:            sale.GetNIT(),
		Nombre:         sale.GetName(),
		CodDomiciliado: &fiscalDomicile,
	}
}
