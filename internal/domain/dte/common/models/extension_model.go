package models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
)

// Extension is a struct that represents an extension of a DTE, containing DeliveryName, DeliveryDocument, ReceiverName,
// ReceiverDocument and Observation
type Extension struct {
	DeliveryName     document.DeliveryName     `json:"deliveryName"`
	DeliveryDocument document.DeliveryDocument `json:"deliveryDocument"`
	ReceiverName     document.DeliveryName     `json:"receiverName"`
	ReceiverDocument document.DeliveryDocument `json:"receiverDocument"`
	Observation      *document.Observation     `json:"observation,omitempty"`
	VehiculePlate    *string                   `json:"vehiculePlate,omitempty"`
}

func (e *Extension) GetDeliveryName() string {
	return e.DeliveryName.GetValue()
}
func (e *Extension) GetDeliveryDocument() string {
	return e.DeliveryDocument.GetValue()
}
func (e *Extension) GetReceiverName() string {
	return e.ReceiverName.GetValue()
}
func (e *Extension) GetReceiverDocument() string {
	return e.ReceiverDocument.GetValue()
}
func (e *Extension) GetVehiculePlate() *string {
	return e.VehiculePlate
}
func (e *Extension) GetObservation() *string {
	if e.Observation != nil {
		value := e.Observation.GetValue()
		return &value
	}
	return nil
}

func (e *Extension) SetDeliveryName(deliveryName string) error {
	dnObj, err := document.NewDeliveryName(deliveryName)
	if err != nil {
		return err
	}
	e.DeliveryName = *dnObj
	return nil
}

func (e *Extension) SetDeliveryDocument(deliveryDocument string) error {
	ddObj, err := document.NewDeliveryDocument(deliveryDocument)
	if err != nil {
		return err
	}
	e.DeliveryDocument = *ddObj
	return nil
}

func (e *Extension) SetReceiverName(receiverName string) error {
	rnObj, err := document.NewDeliveryName(receiverName)
	if err != nil {
		return err
	}
	e.ReceiverName = *rnObj
	return nil
}

func (e *Extension) SetReceiverDocument(receiverDocument string) error {
	rdObj, err := document.NewDeliveryDocument(receiverDocument)
	if err != nil {
		return err
	}
	e.ReceiverDocument = *rdObj
	return nil
}

func (e *Extension) SetObservation(observation *string) error {
	if observation == nil {
		e.Observation = nil
		return nil
	}

	obsObj, err := document.NewObservation(*observation)
	if err != nil {
		return err
	}
	e.Observation = obsObj
	return nil
}

func (e *Extension) SetVehiculePlate(vehiculePlate *string) error {
	e.VehiculePlate = vehiculePlate
	return nil
}
