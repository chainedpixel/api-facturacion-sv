package fse_models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
)

type FSESummary struct {
	*models.Summary
	TotalPurchase   financial.Amount
	IVARetention    financial.Amount
	IncomeRetention financial.Amount
	Observations    *string
}

func NewFSESummary(
	baseSummary *models.Summary,
	totalPurchase financial.Amount,
	ivaRetention financial.Amount,
	incomeRetention financial.Amount,
	observations *string,
) *FSESummary {
	return &FSESummary{
		Summary:         baseSummary,
		TotalPurchase:   totalPurchase,
		IVARetention:    ivaRetention,
		IncomeRetention: incomeRetention,
		Observations:    observations,
	}
}

func (s *FSESummary) GetTotalPurchase() financial.Amount {
	return s.TotalPurchase
}

func (s *FSESummary) GetIVARetention() financial.Amount {
	return s.IVARetention
}

func (s *FSESummary) GetIncomeRetention() financial.Amount {
	return s.IncomeRetention
}

func (s *FSESummary) SetTotalPurchase(amount financial.Amount) {
	s.TotalPurchase = amount
}

func (s *FSESummary) SetIVARetention(amount financial.Amount) {
	s.IVARetention = amount
}

func (s *FSESummary) SetIncomeRetention(amount financial.Amount) {
	s.IncomeRetention = amount
}

func (s *FSESummary) GetObservations() *string {
	return s.Observations
}

func (s *FSESummary) SetObservations(observations *string) {
	s.Observations = observations
}
