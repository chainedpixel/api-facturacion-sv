package dte_documents

import "context"

// SequentialNumberManager defines the methods that a sequential number repository must implement
type SequentialNumberManager interface {
	GetNextControlNumber(ctx context.Context, dteType string, branchID uint, posCode, establishmentCode *string) (string, error)

	ReserveNextNumber(ctx context.Context, dteType string, branchID uint, posCode, establishmentCode *string, documentData string, isContingency bool) (string, uint, error)

	ConfirmReservation(ctx context.Context, controlNumber string, documentID string, branchID uint) error

	ReleaseReservation(ctx context.Context, controlNumber string, rejectionReason string, haciendaCode string, branchID uint) error

	ConfirmReservationByDocumentID(ctx context.Context, documentID string) error

	ReleaseReservationByDocumentID(ctx context.Context, documentID string, rejectionReason string, haciendaCode string) error

	MarkReservationAsContingency(ctx context.Context, reservationID uint, documentID string, contingencyDocID string) error

	MarkReservationAsContingencyByControlNumber(ctx context.Context, controlNumber string, documentID string, contingencyDocID string, branchID uint) error
}
