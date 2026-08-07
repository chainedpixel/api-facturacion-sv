package fse

import (
	"strings"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/fse/fse_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/common"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

func MapFSEResponseReceiver(receiver fse_models.FSEReceiver) structs.FSESubjectExcluded {
	var codActividad *string
	if receiver.ActivityCode != nil && receiver.ActivityCode.IsValid() {
		code := receiver.ActivityCode.GetValue()
		codActividad = &code
	}

	var descActividad *string
	if receiver.ActivityDescription != nil && *receiver.ActivityDescription != "" {
		descActividad = receiver.ActivityDescription
	}

	var telefono *string
	if receiver.Phone != nil && receiver.Phone.IsValid() {
		phone := receiver.Phone.GetValue()
		telefono = &phone
	}

	var correo *string
	if receiver.Email != nil && receiver.Email.IsValid() {
		email := receiver.Email.GetValue()
		correo = &email
	}

	direccion := common.MapCommonResponseAddress(receiver.Address)

	nombre := ""
	if receiver.Name != nil {
		nombre = *receiver.Name
	}

	cleanDocNumber := strings.ReplaceAll(receiver.DocumentNumber.GetValue(), "-", "")

	return structs.FSESubjectExcluded{
		TipoDocumento: receiver.DocumentType.GetValue(),
		NumDocumento:  cleanDocNumber,
		Nombre:        nombre,
		CodActividad:  codActividad,
		DescActividad: descActividad,
		Direccion:     &direccion,
		Telefono:      telefono,
		Correo:        correo,
	}
}
