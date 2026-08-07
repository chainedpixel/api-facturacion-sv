package jobs

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/chainedpixel/ordo-factus/config/drivers"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

type ReservationCleanerJob struct {
	connection       *drivers.DbConnection
	repository       dte_documents.ReservedSequenceRepositoryPort
	IsRunning        atomic.Bool
	MaxExecutionTime time.Duration
}

func NewReservationCleanerJob(repository dte_documents.ReservedSequenceRepositoryPort, connection *drivers.DbConnection) *ReservationCleanerJob {
	return &ReservationCleanerJob{
		connection:       connection,
		repository:       repository,
		MaxExecutionTime: 5 * time.Minute,
	}
}

// Execute ejecuta el trabajo de limpieza de reservas expiradas.
func (j *ReservationCleanerJob) Execute() {
	if !j.IsRunning.CompareAndSwap(false, true) {
		logs.Warn("Reservation cleaner job already running, skipping execution")
		return
	}
	defer j.IsRunning.Store(false)

	ctx, cancel := context.WithTimeout(context.Background(), j.MaxExecutionTime)
	defer cancel()

	logs.Info("Starting reservation cleaner job", map[string]interface{}{
		"MaxExecutionTime": j.MaxExecutionTime,
		"timestamp":        utils.TimeNow().Format(time.RFC3339),
	})

	sqlDb, err := j.connection.Db.DB()
	if err != nil {
		logs.Error("Error connecting to database", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	sqlDb.Ping()

	if err := j.cleanExpiredReservations(ctx); err != nil {
		j.handleExecutionError(err)
		return
	}

	logs.Info("Reservation cleaner job completed successfully", map[string]interface{}{
		"timestamp": utils.TimeNow().Format(time.RFC3339),
	})
}

func (j *ReservationCleanerJob) cleanExpiredReservations(ctx context.Context) error {
	expired, err := j.repository.GetExpiredNonContingencyReservations(ctx)
	if err != nil {
		return err
	}

	if len(expired) == 0 {
		logs.Info("No expired reservations found")
		return nil
	}

	logs.Info("Found expired reservations to clean", map[string]interface{}{
		"count": len(expired),
	})

	for _, reservation := range expired {
		err := j.repository.UpdateStatus(
			ctx,
			reservation.BranchID,
			reservation.DTEType,
			reservation.SequenceNumber,
			reservation.Year,
			dte_documents.ReservationStatusReleased,
			time.Now(),
		)

		if err != nil {
			logs.Error("Error releasing expired reservation", map[string]interface{}{
				"reservationID":  reservation.ID,
				"branchID":       reservation.BranchID,
				"dteType":        reservation.DTEType,
				"sequenceNumber": reservation.SequenceNumber,
				"year":           reservation.Year,
				"error":          err.Error(),
			})
			continue
		}

		logs.Info("Auto-released expired reservation", map[string]interface{}{
			"reservationID":  reservation.ID,
			"branchID":       reservation.BranchID,
			"dteType":        reservation.DTEType,
			"sequenceNumber": reservation.SequenceNumber,
			"year":           reservation.Year,
		})
	}

	return nil
}

func (j *ReservationCleanerJob) handleExecutionError(err error) {
	if errors.Is(err, context.DeadlineExceeded) {
		logs.Error("Reservation cleaner job execution timed out", map[string]interface{}{
			"MaxExecutionTime": j.MaxExecutionTime,
			"error":            err.Error(),
		})
		return
	}

	logs.Error("Reservation cleaner job execution failed", map[string]interface{}{
		"error": err.Error(),
	})
}
