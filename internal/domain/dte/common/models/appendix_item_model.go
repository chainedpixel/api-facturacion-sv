package models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/document"
)

// Appendix is a structure that represents an appendix, contains Field, Label and Value of a DTE
type Appendix struct {
	Field document.AppendixField `json:"field"`
	Label document.AppendixLabel `json:"label"`
	Value document.AppendixValue `json:"value"`
}

func (a *Appendix) GetField() string {
	return a.Field.GetValue()
}

func (a *Appendix) GetLabel() string {
	return a.Label.GetValue()
}

func (a *Appendix) GetValue() string {
	return a.Value.GetValue()
}

func (a *Appendix) SetField(field string) error {
	fieldObj, err := document.NewAppendixField(field)
	if err != nil {
		return err
	}
	a.Field = *fieldObj
	return nil
}

func (a *Appendix) SetLabel(label string) error {
	labelObj, err := document.NewAppendixLabel(label)
	if err != nil {
		return err
	}
	a.Label = *labelObj
	return nil
}

func (a *Appendix) SetValue(value string) error {
	valueObj, err := document.NewAppendixValue(value)
	if err != nil {
		return err
	}
	a.Value = *valueObj
	return nil
}
