package models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/identification"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

type OtherDocument struct {
	AssociatedCode document.AssociatedDocumentCode `json:"associatedCode"`
	Description    *string                         `json:"description,omitempty"`
	Detail         *string                         `json:"detail,omitempty"`
	Doctor         interfaces.DoctorInfo           `json:"doctor,omitempty"`
}

type DoctorInfo struct {
	Name           string               `json:"name"`
	ServiceType    document.ServiceType `json:"serviceType"`
	NIT            *identification.NIT  `json:"NIT,omitempty"`
	Identification *string              `json:"identification,omitempty"`
}

func (o *OtherDocument) GetAssociatedDocument() int {
	return o.AssociatedCode.GetValue()
}

func (o *OtherDocument) GetDescription() string {
	return utils.PointerToString(o.Description)
}

func (o *OtherDocument) GetDetail() string {
	return utils.PointerToString(o.Detail)
}

func (o *OtherDocument) GetDoctor() interfaces.DoctorInfo {
	return o.Doctor
}

func (d *DoctorInfo) GetName() string {
	if d == nil {
		return ""
	}
	return d.Name
}

func (d *DoctorInfo) GetServiceType() int {
	if d == nil {
		return 0
	}
	return d.ServiceType.GetValue()
}

func (d *DoctorInfo) GetNIT() string {
	if d == nil || d.NIT == nil {
		return ""
	}
	return d.NIT.GetValue()
}

func (d *DoctorInfo) GetIdentification() string {
	if d == nil {
		return ""
	}
	return utils.PointerToString(d.Identification)
}

func (o *OtherDocument) SetAssociatedDocument(associatedDocument int) error {
	adcObj, err := document.NewAssociatedDocumentCode(associatedDocument)
	if err != nil {
		return err
	}
	o.AssociatedCode = *adcObj
	return nil
}

func (o *OtherDocument) SetDescription(description string) error {
	if description == "" {
		o.Description = nil
		return nil
	}
	o.Description = &description
	return nil
}

func (o *OtherDocument) SetDetail(detail string) error {
	if detail == "" {
		o.Detail = nil
		return nil
	}
	o.Detail = &detail
	return nil
}

func (o *OtherDocument) SetDoctor(doctor interfaces.DoctorInfo) error {
	o.Doctor = doctor
	return nil
}

func (d *DoctorInfo) SetName(name string) error {
	if name == "" {
		return dte_errors.NewValidationError("RequiredField", "Name")
	}
	d.Name = name
	return nil
}

func (d *DoctorInfo) SetServiceType(serviceType int) error {
	stObj, err := document.NewServiceType(serviceType)
	if err != nil {
		return err
	}
	d.ServiceType = *stObj
	return nil
}

func (d *DoctorInfo) SetNIT(nit string) error {
	if nit == "" {
		d.NIT = nil
		return nil
	}
	nitObj, err := identification.NewNIT(nit)
	if err != nil {
		return err
	}
	d.NIT = nitObj
	return nil
}

func (d *DoctorInfo) SetIdentification(identification string) error {
	if identification == "" {
		d.Identification = nil
		return nil
	}
	d.Identification = &identification
	return nil
}
