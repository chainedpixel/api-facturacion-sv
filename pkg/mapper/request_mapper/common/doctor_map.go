package common

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/identification"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
)

// MapCommonRequestDoctorInfo maps doctor information to a doctor model -> Source: Request
func MapCommonRequestDoctorInfo(docInfo structs.DoctorRequest) (*models.DoctorInfo, error) {
	var docNIT *identification.NIT
	var err error
	if docInfo.Name == "" || docInfo.ServiceType == 0 {
		return nil, shared_error.NewFormattedGeneralServiceError("CommonMapper", "MapCommonRequestDoctorInfo", "InvalidDoctorInfo")
	}

	if docInfo.IdentificationDoc == nil && docInfo.NIT == nil {
		return nil, shared_error.NewFormattedGeneralServiceError("CommonMapper", "MapCommonRequestDoctorInfo", "InvalidDoctorDocuments")
	}

	if docInfo.NIT != nil {
		docNIT, err = identification.NewNIT(*docInfo.NIT)
		if err != nil {
			return nil, err
		}
	}

	serviceType, err := document.NewServiceType(docInfo.ServiceType)
	if err != nil {
		return nil, err
	}

	return &models.DoctorInfo{
		Name:           docInfo.Name,
		ServiceType:    *serviceType,
		NIT:            docNIT,
		Identification: docInfo.IdentificationDoc,
	}, nil
}
