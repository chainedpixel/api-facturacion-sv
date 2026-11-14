package common

import (
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/api-facturacion-sv/pkg/mapper/response_mapper/structs"
)

// MapCommonResponseAddress maps an address to a DTE address
func MapCommonResponseAddress(address interfaces.Address) structs.DTEAddress {
	return structs.DTEAddress{
		Departamento: address.GetDepartment(),
		Municipio:    address.GetMunicipality(),
		Distrito:     address.GetDistrict(),
		Complemento:  address.GetComplement(),
	}
}
