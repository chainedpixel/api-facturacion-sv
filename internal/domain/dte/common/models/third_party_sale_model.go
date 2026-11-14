package models

import (
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/constants"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/value_objects/identification"
)

type ThirdPartySale struct {
	NIT            identification.NIT `json:"nit"`
	Name           string             `json:"name"`
	FiscalDomicile int                `json:"fiscal_domicile"` // CAT-032 Domicilio Fiscal
}

func (t *ThirdPartySale) GetNIT() string {
	return t.NIT.GetValue()
}

func (t *ThirdPartySale) GetName() string {
	return t.Name
}

func (t *ThirdPartySale) GetFiscalDomicile() int {
	return t.FiscalDomicile
}

func (t *ThirdPartySale) SetFiscalDomicile(fiscalDomicile int) error {
	if !constants.ValidFiscalDomiciles[fiscalDomicile] {
		return dte_errors.NewValidationError("InvalidField", "FiscalDomicile")
	}
	t.FiscalDomicile = fiscalDomicile
	return nil
}

func (t *ThirdPartySale) SetNIT(nit string) error {
	nitObj, err := identification.NewNIT(nit)
	if err != nil {
		return err
	}
	t.NIT = *nitObj
	return nil
}

func (t *ThirdPartySale) SetName(name string) error {
	if name == "" {
		return dte_errors.NewValidationError("RequiredField", "Name")
	}
	t.Name = name
	return nil
}
