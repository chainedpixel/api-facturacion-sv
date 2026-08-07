package models

import (
	"time"

	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/identification"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/temporal"
)

// Identification is a struct that represents the identification of a DTE, containing Version, Ambient, DTEType, ControlNumber,
// GenerationCode, ModelType, OperationType, EmissionDate, EmissionTime, Currency, ContingencyType and ContingencyReason
type Identification struct {
	Version           document.Version              `json:"version"`
	Ambient           document.Ambient              `json:"ambient"`
	DTEType           document.DTEType              `json:"dteType"`
	ControlNumber     identification.ControlNumber  `json:"controlNumber"`
	GenerationCode    identification.GenerationCode `json:"generationCode"`
	ModelType         document.ModelType            `json:"modelType"`
	OperationType     document.OperationType        `json:"operationType"`
	EmissionDate      temporal.EmissionDate         `json:"emissionDate"`
	EmissionTime      temporal.EmissionTime         `json:"emissionTime"`
	Currency          financial.Currency            `json:"currency"`
	ContingencyType   *document.ContingencyType     `json:"contingencyType,omitempty"`
	ContingencyReason *document.ContingencyReason   `json:"contingencyReason,omitempty"`
}

func (i *Identification) GetVersion() int {
	return i.Version.GetValue()
}
func (i *Identification) GetAmbient() string {
	return i.Ambient.GetValue()
}
func (i *Identification) GetDTEType() string {
	return i.DTEType.GetValue()
}
func (i *Identification) GetControlNumber() string {
	return i.ControlNumber.GetValue()
}
func (i *Identification) GetGenerationCode() string {
	return i.GenerationCode.GetValue()
}
func (i *Identification) GetModelType() int {
	return i.ModelType.GetValue()
}
func (i *Identification) GetOperationType() int {
	return i.OperationType.GetValue()
}
func (i *Identification) GetEmissionDate() time.Time {
	return i.EmissionDate.GetValue()
}
func (i *Identification) GetEmissionTime() time.Time {
	return i.EmissionTime.GetValue()
}
func (i *Identification) GetCurrency() string {
	return i.Currency.GetValue()
}
func (i *Identification) GetContingencyType() *int {
	return utils.ToIntPointer(i.ContingencyType.GetValue())
}
func (i *Identification) GetContingencyReason() *string {
	return utils.ToStringPointer(i.ContingencyReason.GetValue())
}
func (i *Identification) SetControlNumber(controlNumber string) error {
	cn, err := identification.NewControlNumber(controlNumber)
	if err != nil {
		return err
	}
	i.ControlNumber = *cn
	return err
}
func (i *Identification) GenerateCode() error {
	gc, err := identification.NewGenerationCode()
	if err != nil {
		return err
	}
	i.GenerationCode = *gc
	return err
}

func (i *Identification) SetVersion(version int) error {
	versionObj, err := document.NewVersion(version)
	if err != nil {
		return err
	}
	i.Version = *versionObj
	return nil
}

func (i *Identification) SetAmbient(ambient string) error {
	ambientObj, err := document.NewAmbientCustom(ambient)
	if err != nil {
		return err
	}
	i.Ambient = *ambientObj
	return nil
}

func (i *Identification) SetDTEType(dteType string) error {
	dteTypeObj, err := document.NewDTEType(dteType)
	if err != nil {
		return err
	}
	i.DTEType = *dteTypeObj
	return nil
}

func (i *Identification) SetModelType(modelType int) error {
	modelTypeObj, err := document.NewModelType(modelType)
	if err != nil {
		return err
	}
	i.ModelType = *modelTypeObj
	return nil
}

func (i *Identification) SetOperationType(operationType int) error {
	operationTypeObj, err := document.NewOperationType(operationType)
	if err != nil {
		return err
	}
	i.OperationType = *operationTypeObj
	return nil
}

func (i *Identification) SetEmissionDate(emissionDate time.Time) error {
	emissionDateObj, err := temporal.NewEmissionDate(emissionDate)
	if err != nil {
		return err
	}
	i.EmissionDate = *emissionDateObj
	return nil
}

func (i *Identification) SetEmissionTime(emissionTime time.Time) error {
	emissionTimeObj, err := temporal.NewEmissionTime(emissionTime)
	if err != nil {
		return err
	}
	i.EmissionTime = *emissionTimeObj
	return nil
}

func (i *Identification) SetCurrency(currency string) error {
	currencyObj, err := financial.NewCurrency(currency)
	if err != nil {
		return err
	}
	i.Currency = *currencyObj
	return nil
}

func (i *Identification) SetDTETypeForce(dteType string) {
	dteTypeObj := document.NewValidatedDTEType(dteType)
	i.DTEType = *dteTypeObj
}

func (i *Identification) SetContingencyType(contingencyType *int) error {
	if contingencyType == nil {
		i.ContingencyType = nil
		return nil
	}

	ctObj, err := document.NewContingencyType(*contingencyType)
	if err != nil {
		return err
	}
	i.ContingencyType = ctObj
	return nil
}

func (i *Identification) SetContingencyReason(contingencyReason *string) error {
	if contingencyReason == nil {
		i.ContingencyReason = nil
		return nil
	}

	crObj, err := document.NewContingencyReason(*contingencyReason)
	if err != nil {
		return err
	}
	i.ContingencyReason = crObj
	return nil
}
