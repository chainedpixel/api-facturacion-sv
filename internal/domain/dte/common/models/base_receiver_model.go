package models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/base"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/identification"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

// Receiver is a struct that represents the receiver of a DTE, containing Name, DocumentType, DocumentNumber, Address,
// Email, Phone, NRC, ActivityDescription, ActivityCode and CommercialName
type Receiver struct {
	Name                *string                        `json:"name,omitempty"`
	DocumentType        *document.DTEType              `json:"documentType,omitempty"`
	DocumentNumber      *identification.DocumentNumber `json:"documentNumber,omitempty"`
	Address             interfaces.Address             `json:"address,omitempty"`
	Email               *base.Email                    `json:"email,omitempty"`
	Phone               *base.Phone                    `json:"phone,omitempty"`
	NRC                 *identification.NRC            `json:"nrc,omitempty"`
	NIT                 *identification.NIT            `json:"nit,omitempty"`
	ActivityDescription *string                        `json:"activityDescription,omitempty"`
	ActivityCode        *identification.ActivityCode   `json:"activityCode,omitempty"`
	CommercialName      *string                        `json:"commercialName,omitempty"`
}

func (r *Receiver) GetName() *string {
	return r.Name
}
func (r *Receiver) GetDocumentType() *string {
	if r.DocumentType == nil {
		return nil
	}

	return utils.ToStringPointer(r.DocumentType.GetValue())
}
func (r *Receiver) GetDocumentNumber() *string {
	if r.DocumentNumber == nil {
		return nil
	}

	return utils.ToStringPointer(r.DocumentNumber.GetValue())
}
func (r *Receiver) GetAddress() interfaces.Address {
	return r.Address
}
func (r *Receiver) GetEmail() *string {
	if r.Email == nil {
		return nil
	}

	return utils.ToStringPointer(r.Email.GetValue())
}
func (r *Receiver) GetPhone() *string {
	if r.Phone == nil {
		return nil
	}

	return utils.ToStringPointer(r.Phone.GetValue())
}
func (r *Receiver) GetNRC() *string {
	if r.NRC == nil {
		return nil
	}
	return utils.ToStringPointer(r.NRC.GetValue())
}
func (r *Receiver) GetActivityCode() *string {
	if r.ActivityCode == nil {
		return nil
	}
	return utils.ToStringPointer(r.ActivityCode.GetValue())
}
func (r *Receiver) GetNIT() *string {
	if r.NIT == nil {
		return nil
	}
	return utils.ToStringPointer(r.NIT.GetValue())
}
func (r *Receiver) GetActivityDescription() *string {
	return r.ActivityDescription
}
func (r *Receiver) GetCommercialName() *string {
	return r.CommercialName
}
func (r *Receiver) SetName(name *string) error {
	r.Name = name
	return nil
}

func (r *Receiver) SetDocumentType(documentType *string) error {
	if documentType == nil {
		r.DocumentType = nil
		return nil
	}

	dtObj, err := document.NewDTETypeForReceiver(*documentType)
	if err != nil {
		return err
	}
	r.DocumentType = dtObj
	return nil
}

func (r *Receiver) SetDocumentNumber(documentNumber *string) error {
	if documentNumber == nil {
		r.DocumentNumber = nil
		return nil
	}

	if r.DocumentType == nil {
		return dte_errors.NewValidationError("RequiredField", "DocumentType")
	}

	dnObj, err := identification.NewDocumentNumber(*documentNumber, r.DocumentType.GetValue())
	if err != nil {
		return err
	}
	r.DocumentNumber = dnObj
	return nil
}

func (r *Receiver) SetAddress(address interfaces.Address) error {
	r.Address = address
	return nil
}

func (r *Receiver) SetEmail(email *string) error {
	if email == nil {
		r.Email = nil
		return nil
	}

	emailObj, err := base.NewEmail(*email)
	if err != nil {
		return err
	}
	r.Email = emailObj
	return nil
}

func (r *Receiver) SetPhone(phone *string) error {
	if phone == nil {
		r.Phone = nil
		return nil
	}

	phoneObj, err := base.NewPhone(*phone)
	if err != nil {
		return err
	}
	r.Phone = phoneObj
	return nil
}

func (r *Receiver) SetNRC(nrc *string) error {
	if nrc == nil {
		r.NRC = nil
		return nil
	}

	nrcObj, err := identification.NewNRC(*nrc)
	if err != nil {
		return err
	}
	r.NRC = nrcObj
	return nil
}

func (r *Receiver) SetNIT(nit *string) error {
	if nit == nil {
		r.NIT = nil
		return nil
	}

	nitObj, err := identification.NewNIT(*nit)
	if err != nil {
		return err
	}
	r.NIT = nitObj
	return nil
}

func (r *Receiver) SetActivityCode(activityCode *string) error {
	if activityCode == nil {
		r.ActivityCode = nil
		return nil
	}

	acObj, err := identification.NewActivityCode(*activityCode)
	if err != nil {
		return err
	}
	r.ActivityCode = acObj
	return nil
}

func (r *Receiver) SetActivityDescription(activityDescription *string) error {
	r.ActivityDescription = activityDescription
	return nil
}

func (r *Receiver) SetCommercialName(commercialName *string) error {
	r.CommercialName = commercialName
	return nil
}
