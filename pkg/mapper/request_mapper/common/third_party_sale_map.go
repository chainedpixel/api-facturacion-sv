package common

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/identification"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
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

	return &models.ThirdPartySale{
		Name: sale.Name,
		NIT:  *nit,
	}, nil
}
