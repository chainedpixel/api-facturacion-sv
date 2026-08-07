package common

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/temporal"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

// MapCommonRequestIdentification maps a common identification to an identification model -> Source: Request
func MapCommonRequestIdentification(model, versionDoc int, typeToEmit string) (*models.Identification, error) {
	now := utils.TimeNow()

	version := document.NewValidatedVersion(versionDoc)
	dteType := document.NewValidatedDTEType(typeToEmit)
	currency := financial.NewValidatedCurrency("USD")

	ambient, err := document.NewAmbient()
	if err != nil {
		return nil, err
	}

	emissionDate, err := temporal.NewEmissionDate(now)
	if err != nil {
		return nil, err
	}

	emissionTime, err := temporal.NewEmissionTime(now)
	if err != nil {
		return nil, err
	}

	modelType, err := document.NewModelType(model)
	if err != nil {
		return nil, err
	}

	operationType, err := document.NewOperationType(constants.TransmisionNormal)
	if err != nil {
		return nil, err
	}

	return &models.Identification{
		Version:       *version,
		Ambient:       *ambient,
		DTEType:       *dteType,
		Currency:      *currency,
		OperationType: *operationType,
		ModelType:     *modelType,
		EmissionDate:  *emissionDate,
		EmissionTime:  *emissionTime,
	}, nil
}
