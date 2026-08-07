package dte_documents

import (
	"context"
	"time"
)

type ReservedSequenceRepositoryPort interface {
	Create(ctx context.Context, reservation *ReservedSequence) (uint, error)

	GetOldestReleasedNumber(ctx context.Context, branchID uint, dteType string, year int) (*ReservedSequence, error)

	MarkAsReserved(ctx context.Context, reservationID uint, expiresAt *time.Time) error

	UpdateStatus(ctx context.Context, branchID uint, dteType string, seqNum uint, year int, status string, timestamp time.Time) error

	UpdateStatusWithReason(ctx context.Context, branchID uint, dteType string, seqNum uint, year int, status string, timestamp time.Time, reason string, haciendaCode string) error

	UpdateStatusWithDocumentID(ctx context.Context, branchID uint, dteType string, seqNum uint, year int, status string, timestamp time.Time, documentID string) error

	UpdateAsContingency(ctx context.Context, reservationID uint, documentID string, contingencyDocID string) error

	UpdateAsContingencyByControlNumber(ctx context.Context, branchID uint, dteType string, seqNum uint, year int, documentID string, contingencyDocID string) error

	GetByDocumentID(ctx context.Context, documentID string) (*ReservedSequence, error)

	GetExpiredNonContingencyReservations(ctx context.Context) ([]ReservedSequence, error)
}
