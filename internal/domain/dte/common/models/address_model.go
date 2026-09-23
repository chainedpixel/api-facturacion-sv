package models

import "github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/location"

// Address is a structure that represents a Department, Municipality, District and Complement of a DTE
type Address struct {
	Department   location.Department   `json:"department"`
	Municipality location.Municipality `json:"municipality"`
	District     location.District     `json:"district"`
	Complement   location.Address      `json:"complement"`
}

func (a *Address) GetDepartment() string {
	return a.Department.GetValue()
}

func (a *Address) GetMunicipality() string {
	return a.Municipality.GetValue()
}

func (a *Address) GetDistrict() string {
	return a.District.GetValue()
}

func (a *Address) GetComplement() string {
	return a.Complement.GetValue()
}

func (a *Address) SetDepartment(department string) error {
	deptObj, err := location.NewDepartment(department)
	if err != nil {
		return err
	}
	a.Department = *deptObj
	return nil
}

func (a *Address) SetMunicipality(municipality string) error {
	munObj, err := location.NewMunicipality(municipality, a.Department)
	if err != nil {
		return err
	}
	a.Municipality = *munObj
	return nil
}

func (a *Address) SetDistrict(district string) error {
	distObj, err := location.NewDistrict(district)
	if err != nil {
		return err
	}
	a.District = *distObj
	return nil
}

func (a *Address) SetComplement(complement string) error {
	compObj, err := location.NewAddress(complement)
	if err != nil {
		return err
	}
	a.Complement = *compObj
	return nil
}
