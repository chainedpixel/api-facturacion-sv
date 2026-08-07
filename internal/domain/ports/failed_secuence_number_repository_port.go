package ports

import (
	"context"

	"github.com/chainedpixel/ordo-factus/internal/infrastructure/database/db_models"
)

// FailedSequenceNumberRepositoryPort defines the interface for the failed sequence number repository
type FailedSequenceNumberRepositoryPort interface {
	RegisterFailedSequence(
		ctx context.Context,
		branchID uint,
		dteType string,
		sequenceNumber uint,
		year uint,
		failureReason string,
		responseCode string,
		originalRequestData interface{},
		mhResponse string,
	) error

	GetFailedSequences(
		ctx context.Context,
		branchID uint,
		dteType string,
		limit int,
	) ([]db_models.FailedSequenceNumber, error)

	GetFailedSequencesByYear(
		ctx context.Context,
		branchID uint,
		dteType string,
		year uint,
		limit int,
	) ([]db_models.FailedSequenceNumber, error)
}
