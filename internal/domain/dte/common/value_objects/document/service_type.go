package document

import (
	"fmt"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
)

type ServiceType struct {
	value int
}

func NewServiceType(value int) (*ServiceType, error) {
	service := ServiceType{value: value}
	if service.IsValid() {
		return &service, nil
	}
	return nil, dte_errors.NewValidationError("InvalidServiceType", value)
}

func NewValidatedServiceType(value int) *ServiceType {
	return &ServiceType{value: value}
}

func (s *ServiceType) IsValid() bool {
	return s.value > 0 && s.value <= 6
}

func (s *ServiceType) GetValue() int {
	return s.value
}

func (s *ServiceType) Equals(other interfaces.ValueObject[int]) bool {
	return s.value == other.GetValue()
}

func (s *ServiceType) ToString() string {
	return fmt.Sprintf("%d", s.value)
}
