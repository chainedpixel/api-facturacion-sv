package remission_note

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/remission_note/remission_note_models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/response_mapper/structs"
)

// MapRemissionNoteReceiver converts the specific Remission Note receiver to the Ministry of Finance format
func MapRemissionNoteReceiver(model *remission_note_models.RemissionNoteModel) *structs.MHRemissionNoteReceiver {
	if model.Receiver == nil {
		return nil
	}

	remissionReceiver, ok := model.Receiver.(*remission_note_models.RemissionNoteReceiver)
	if !ok {
		return mapFromBaseReceiver(model.Receiver)
	}

	return mapFromRemissionReceiver(remissionReceiver)
}

func mapFromRemissionReceiver(receiver *remission_note_models.RemissionNoteReceiver) *structs.MHRemissionNoteReceiver {
	mhReceiver := &structs.MHRemissionNoteReceiver{
		DocumentType:   getStringValueOrDefault(receiver.GetDocumentType(), ""),
		DocumentNumber: getStringValueOrDefault(receiver.GetDocumentNumber(), ""),
		Name:           getStringValueOrDefault(receiver.GetName(), ""),
		Email:          getStringValueOrDefault(receiver.GetEmail(), ""),
		BienTitulo:     getStringValueOrDefault(receiver.GetBienTitulo(), ""),
	}

	if nrc := receiver.GetNRC(); nrc != nil {
		docType := getStringValueOrDefault(receiver.GetDocumentType(), "")
		if docType == "13" {
			mhReceiver.NRC = nil
		} else {
			mhReceiver.NRC = nrc
		}
	} else {
		mhReceiver.NRC = nil
	}

	if activityCode := receiver.GetActivityCode(); activityCode != nil {
		mhReceiver.ActivityCode = activityCode
	}

	if activityDesc := receiver.GetActivityDescription(); activityDesc != nil {
		mhReceiver.ActivityDesc = activityDesc
	}

	if commercialName := receiver.GetCommercialName(); commercialName != nil {
		mhReceiver.CommercialName = commercialName
	}

	if phone := receiver.GetPhone(); phone != nil {
		mhReceiver.Phone = phone
	}

	if address := receiver.GetAddress(); address != nil {
		mhReceiver.Address = &structs.DTEAddress{
			Departamento: address.GetDepartment(),
			Municipio:    address.GetMunicipality(),
			Complemento:  address.GetComplement(),
		}
	}

	return mhReceiver
}

func mapFromBaseReceiver(receiver interface{}) *structs.MHRemissionNoteReceiver {
	receiverGetter, ok := receiver.(interface {
		GetDocumentType() *string
		GetDocumentNumber() *string
		GetName() *string
		GetEmail() *string
		GetNRC() *string
		GetActivityCode() *string
		GetActivityDescription() *string
		GetCommercialName() *string
		GetPhone() *string
		GetAddress() interface{}
	})

	if !ok {
		return nil
	}

	mhReceiver := &structs.MHRemissionNoteReceiver{
		DocumentType:   getStringValueOrDefault(receiverGetter.GetDocumentType(), ""),
		DocumentNumber: getStringValueOrDefault(receiverGetter.GetDocumentNumber(), ""),
		Name:           getStringValueOrDefault(receiverGetter.GetName(), ""),
		Email:          getStringValueOrDefault(receiverGetter.GetEmail(), ""),
		BienTitulo:     "99",
	}

	if nrc := receiverGetter.GetNRC(); nrc != nil {
		docType := getStringValueOrDefault(receiverGetter.GetDocumentType(), "")
		if docType == "13" {
			mhReceiver.NRC = nil
		} else {
			mhReceiver.NRC = nrc
		}
	} else {
		mhReceiver.NRC = nil
	}

	if activityCode := receiverGetter.GetActivityCode(); activityCode != nil {
		mhReceiver.ActivityCode = activityCode
	}

	if activityDesc := receiverGetter.GetActivityDescription(); activityDesc != nil {
		mhReceiver.ActivityDesc = activityDesc
	}

	if commercialName := receiverGetter.GetCommercialName(); commercialName != nil {
		mhReceiver.CommercialName = commercialName
	}

	if phone := receiverGetter.GetPhone(); phone != nil {
		mhReceiver.Phone = phone
	}

	if address := receiverGetter.GetAddress(); address != nil {
		if addressInterface, ok := address.(interface {
			GetDepartment() string
			GetMunicipality() string
			GetComplement() string
		}); ok {
			mhReceiver.Address = &structs.DTEAddress{
				Departamento: addressInterface.GetDepartment(),
				Municipio:    addressInterface.GetMunicipality(),
				Complemento:  addressInterface.GetComplement(),
			}
		}
	}

	return mhReceiver
}

func getStringValueOrDefault(value *string, defaultValue string) string {
	if value == nil {
		return defaultValue
	}
	return *value
}
