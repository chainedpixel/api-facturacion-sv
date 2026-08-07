package models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/base"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/identification"
)

// Issuer is a struct that represents the issuer of a DTE
type Issuer struct {
	NIT                 identification.NIT          `json:"nit"`
	NRC                 identification.NRC          `json:"nrc"`
	Name                string                      `json:"name"`
	ActivityCode        identification.ActivityCode `json:"activityCode"`
	ActivityDescription string                      `json:"activityDescription"`
	EstablishmentType   document.EstablishmentType  `json:"establishmentType"`
	Address             interfaces.Address          `json:"address"`
	Phone               base.Phone                  `json:"phone"`
	Email               base.Email                  `json:"email"`
	CommercialName      string                      `json:"commercialName"`
	EstablishmentCode   *string                     `json:"establishmentCode,omitempty"`
	EstablishmentMHCode *string                     `json:"establishmentMHCode,omitempty"`
	POSCode             *string                     `json:"POSCode,omitempty"`
	POSMHCode           *string                     `json:"POSMHCode,omitempty"`
}

func (i *Issuer) GetName() string {
	return i.Name
}
func (i *Issuer) GetActivityDescription() string {
	return i.ActivityDescription
}
func (i *Issuer) GetCommercialName() string {
	return i.CommercialName
}
func (i *Issuer) GetNIT() string {
	return i.NIT.GetValue()
}
func (i *Issuer) GetNRC() string {
	return i.NRC.GetValue()
}
func (i *Issuer) GetActivityCode() string {
	return i.ActivityCode.GetValue()
}
func (i *Issuer) GetEstablishmentType() string {
	return i.EstablishmentType.GetValue()
}
func (i *Issuer) GetAddress() interfaces.Address {
	return i.Address
}
func (i *Issuer) GetPhone() string {
	return i.Phone.GetValue()
}
func (i *Issuer) GetEmail() string {
	return i.Email.GetValue()
}
func (i *Issuer) GetEstablishmentCode() *string {
	return i.EstablishmentCode
}
func (i *Issuer) GetEstablishmentMHCode() *string {
	return i.EstablishmentMHCode
}
func (i *Issuer) GetPOSCode() *string {
	return i.POSCode
}
func (i *Issuer) GetPOSMHCode() *string {
	return i.POSMHCode
}

func (i *Issuer) SetName(name string) error {
	if name == "" {
		return dte_errors.NewValidationError("RequiredField", "Name")
	}
	i.Name = name
	return nil
}

func (i *Issuer) SetActivityDescription(description string) error {
	if description == "" {
		return dte_errors.NewValidationError("RequiredField", "ActivityDescription")
	}
	i.ActivityDescription = description
	return nil
}

func (i *Issuer) SetCommercialName(commercialName string) error {
	i.CommercialName = commercialName
	return nil
}

func (i *Issuer) SetNIT(nit string) error {
	nitObj, err := identification.NewNIT(nit)
	if err != nil {
		return err
	}
	i.NIT = *nitObj
	return nil
}

func (i *Issuer) SetNRC(nrc string) error {
	nrcObj, err := identification.NewNRC(nrc)
	if err != nil {
		return err
	}
	i.NRC = *nrcObj
	return nil
}

func (i *Issuer) SetActivityCode(activityCode string) error {
	acObj, err := identification.NewActivityCode(activityCode)
	if err != nil {
		return err
	}
	i.ActivityCode = *acObj
	return nil
}

func (i *Issuer) SetEstablishmentType(establishmentType string) error {
	etObj, err := document.NewEstablishmentType(establishmentType)
	if err != nil {
		return err
	}
	i.EstablishmentType = *etObj
	return nil
}

func (i *Issuer) SetAddress(address interfaces.Address) error {
	if address == nil {
		return dte_errors.NewValidationError("RequiredField", "Address")
	}
	i.Address = address
	return nil
}

func (i *Issuer) SetPhone(phone string) error {
	phoneObj, err := base.NewPhone(phone)
	if err != nil {
		return err
	}
	i.Phone = *phoneObj
	return nil
}

func (i *Issuer) SetEmail(email string) error {
	emailObj, err := base.NewEmail(email)
	if err != nil {
		return err
	}
	i.Email = *emailObj
	return nil
}

func (i *Issuer) SetEstablishmentCode(establishmentCode *string) error {
	i.EstablishmentCode = establishmentCode
	return nil
}

func (i *Issuer) SetEstablishmentMHCode(establishmentMHCode *string) error {
	i.EstablishmentMHCode = establishmentMHCode
	return nil
}

func (i *Issuer) SetPOSCode(posCode *string) error {
	i.POSCode = posCode
	return nil
}

func (i *Issuer) SetPOSMHCode(posMHCode *string) error {
	i.POSMHCode = posMHCode
	return nil
}
