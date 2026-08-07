package models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
)

// Summary is a structure that represents a summary of a DTE, contains TotalNonSubject, TotalExempt, TotalTaxed, SubTotal, NonSubjectDiscount,
// ExemptDiscount, DiscountPercentage, TotalDiscount, TotalTaxes, SubTotalOperation, TotalOperation, TotalNonTaxed, PaymentTypes, OperationCondition and ElectronicPayment
type Summary struct {
	TotalNonSubject    financial.Amount           `json:"totalNonSubject"`
	TotalExempt        financial.Amount           `json:"totalExempt"`
	TotalTaxed         financial.Amount           `json:"totalTaxed"`
	SubTotal           financial.Amount           `json:"subTotal"`
	SubTotalSales      financial.Amount           `json:"subTotalSales"`
	NonSubjectDiscount financial.Amount           `json:"nonSubjectDiscount"`
	ExemptDiscount     financial.Amount           `json:"exemptDiscount"`
	DiscountPercentage financial.Discount         `json:"discountPercentage"`
	TotalDiscount      financial.Amount           `json:"totalDiscount"`
	TotalOperation     financial.Amount           `json:"totalOperation"`
	TotalNonTaxed      financial.Amount           `json:"totalNotTaxed"`
	OperationCondition financial.PaymentCondition `json:"operation_condition"`
	TotalToPay         financial.Amount           `json:"totalToPay"`
	TotalTaxes         []interfaces.Tax           `json:"taxes,omitempty"`
	TotalInWords       string                     `json:"totalInWords"`
	ElectronicPayment  *string                    `json:"electronicPayment,omitempty"`
	PaymentTypes       []interfaces.PaymentType   `json:"payments,omitempty"`
}

func (s *Summary) GetTotalNonSubject() float64 {
	return s.TotalNonSubject.GetValue()
}
func (s *Summary) GetTotalExempt() float64 {
	return s.TotalExempt.GetValue()
}
func (s *Summary) GetTotalTaxed() float64 {
	return s.TotalTaxed.GetValue()
}
func (s *Summary) GetSubTotal() float64 {
	return s.SubTotal.GetValue()
}
func (s *Summary) GetNonSubjectDiscount() float64 {
	return s.NonSubjectDiscount.GetValue()
}
func (s *Summary) GetExemptDiscount() float64 {
	return s.ExemptDiscount.GetValue()
}
func (s *Summary) GetDiscountPercentage() float64 {
	return s.DiscountPercentage.GetValue()
}
func (s *Summary) GetTotalDiscount() float64 {
	return s.TotalDiscount.GetValue()
}
func (s *Summary) GetTotalTaxes() []interfaces.Tax {
	return s.TotalTaxes
}
func (s *Summary) GetTotalOperation() float64 {
	return s.TotalOperation.GetValue()
}
func (s *Summary) GetTotalNotTaxed() float64 {
	return s.TotalNonTaxed.GetValue()
}
func (s *Summary) GetPaymentTypes() []interfaces.PaymentType {
	return s.PaymentTypes
}
func (s *Summary) GetOperationCondition() int {
	return s.OperationCondition.GetValue()
}
func (s *Summary) GetSubtotalSales() float64 {
	return s.SubTotalSales.GetValue()
}
func (s *Summary) GetTotalToPay() float64 {
	return s.TotalToPay.GetValue()
}
func (s *Summary) GetTotalInWords() string {
	return s.TotalInWords
}
func (s *Summary) GetElectronicPayment() *string {
	return s.ElectronicPayment
}

func (s *Summary) SetTotalNonSubject(totalNonSubject float64) error {
	tnsObj, err := financial.NewAmount(totalNonSubject)
	if err != nil {
		return err
	}
	s.TotalNonSubject = *tnsObj
	return nil
}

func (s *Summary) SetTotalExempt(totalExempt float64) error {
	teObj, err := financial.NewAmount(totalExempt)
	if err != nil {
		return err
	}
	s.TotalExempt = *teObj
	return nil
}

func (s *Summary) SetTotalTaxed(totalTaxed float64) error {
	ttObj, err := financial.NewAmount(totalTaxed)
	if err != nil {
		return err
	}
	s.TotalTaxed = *ttObj
	return nil
}

func (s *Summary) SetSubTotal(subTotal float64) error {
	stObj, err := financial.NewAmount(subTotal)
	if err != nil {
		return err
	}
	s.SubTotal = *stObj
	return nil
}

func (s *Summary) SetSubtotalSales(subtotalSales float64) error {
	ssObj, err := financial.NewAmount(subtotalSales)
	if err != nil {
		return err
	}
	s.SubTotalSales = *ssObj
	return nil
}

func (s *Summary) SetNonSubjectDiscount(nonSubjectDiscount float64) error {
	nsdObj, err := financial.NewAmount(nonSubjectDiscount)
	if err != nil {
		return err
	}
	s.NonSubjectDiscount = *nsdObj
	return nil
}

func (s *Summary) SetExemptDiscount(exemptDiscount float64) error {
	edObj, err := financial.NewAmount(exemptDiscount)
	if err != nil {
		return err
	}
	s.ExemptDiscount = *edObj
	return nil
}

func (s *Summary) SetDiscountPercentage(discountPercentage float64) error {
	dpObj, err := financial.NewDiscount(discountPercentage)
	if err != nil {
		return err
	}
	s.DiscountPercentage = *dpObj
	return nil
}

func (s *Summary) SetTotalDiscount(totalDiscount float64) error {
	tdObj, err := financial.NewAmount(totalDiscount)
	if err != nil {
		return err
	}
	s.TotalDiscount = *tdObj
	return nil
}

func (s *Summary) SetTotalTaxes(totalTaxes []interfaces.Tax) error {
	s.TotalTaxes = totalTaxes
	return nil
}

func (s *Summary) SetTotalOperation(totalOperation float64) error {
	toObj, err := financial.NewAmount(totalOperation)
	if err != nil {
		return err
	}
	s.TotalOperation = *toObj
	return nil
}

func (s *Summary) SetTotalNotTaxed(totalNotTaxed float64) error {
	tntObj, err := financial.NewAmount(totalNotTaxed)
	if err != nil {
		return err
	}
	s.TotalNonTaxed = *tntObj
	return nil
}

func (s *Summary) SetPaymentTypes(paymentTypes []interfaces.PaymentType) error {
	s.PaymentTypes = paymentTypes
	return nil
}

func (s *Summary) SetOperationCondition(operationCondition int) error {
	ocObj, err := financial.NewPaymentCondition(operationCondition)
	if err != nil {
		return err
	}
	s.OperationCondition = *ocObj
	return nil
}

func (s *Summary) SetElectronicPayment(electronicPayment *string) error {
	s.ElectronicPayment = electronicPayment
	return nil
}

func (s *Summary) SetTotalInWords(totalInWords string) error {
	if totalInWords == "" {
		return dte_errors.NewValidationError("RequiredField", "TotalInWords")
	}
	s.TotalInWords = totalInWords
	return nil
}

func (s *Summary) SetTotalToPay(totalToPay float64) error {
	ttpObj, err := financial.NewAmount(totalToPay)
	if err != nil {
		return err
	}
	s.TotalToPay = *ttpObj
	return nil
}

func (s *Summary) SetForceTotalToPay(totalToPay float64) {
	ttpObj := financial.NewValidatedAmount(totalToPay)
	s.TotalToPay = *ttpObj
}
