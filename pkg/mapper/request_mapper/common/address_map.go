package common

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/core/user"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/location"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
)

// MapCommonRequestAddress maps a common address to an address model -> Source: Request
func MapCommonRequestAddress(address structs.AddressRequest) (*models.Address, error) {
	if address.Department == "" || address.Municipality == "" || address.Complement == "" {
		return nil, shared_error.NewFormattedGeneralServiceError("CommonMapper", "MapCommonRequestAddress", "AddressWithReceiver")
	}

	department, err := location.NewDepartment(address.Department)
	if err != nil {
		return nil, err
	}

	municipality, err := location.NewMunicipality(address.Municipality, *department)
	if err != nil {
		return nil, err
	}

	complement, err := location.NewAddress(address.Complement)
	if err != nil {
		return nil, err
	}

	return &models.Address{
		Department:   *department,
		Municipality: *municipality,
		Complement:   *complement,
	}, nil
}

// MapClientAddress maps a client address to an address model -> Source: Database
func MapClientAddress(address *user.Address) (*models.Address, error) {
	return &models.Address{
		Department:   *location.NewValidatedDepartment(address.Department),
		Municipality: *location.NewValidatedMunicipality(address.Municipality, address.Department),
		Complement:   *location.NewValidatedAddress(address.Complement),
	}, nil
}
