package common

import (
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/models"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/value_objects/identification"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/api-facturacion-sv/pkg/shared/shared_error"
)

// MapCommonRequestThirdPartySale maps a third-party sale to a model
func MapCommonRequestThirdPartySale(sale *structs.ThirdPartySaleRequest) (*models.ThirdPartySale, error) {
	if sale.Name == "" || sale.NIT == "" {
		return nil, shared_error.NewFormattedGeneralServiceError(
			"CommonMapper",
			"MapCommonRequestThirdPartySale",
			"InvalidThirdParty",
		)
	}

	nit, err := identification.NewNIT(sale.NIT)
	if err != nil {
		return nil, err
	}

	thirdPartySale := &models.ThirdPartySale{
		Name: sale.Name,
		NIT:  *nit,
	}

	// Anexo IV: cuando se detalla el nombre del tercero, el Domicilio Fiscal (CAT-032) es requerido
	if err := thirdPartySale.SetFiscalDomicile(sale.FiscalDomicile); err != nil {
		return nil, err
	}

	return thirdPartySale, nil
}
