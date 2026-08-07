package dte

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/core/user"
)

type IssuerDTE struct {
	NIT                  string
	NRC                  string
	CommercialName       string
	BusinessName         string
	EconomicActivity     string
	EconomicActivityDesc string
	EstablishmentCode    *string
	EstablishmentCodeMH  *string
	Email                *string
	Phone                *string
	EstablishmentType    string
	POSCode              *string
	POSCodeMH            *string
	Address              *user.Address
}
