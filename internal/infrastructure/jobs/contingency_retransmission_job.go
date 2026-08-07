package jobs

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/chainedpixel/ordo-factus/config/drivers"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/event"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/contingency"

	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

type RetransmissionJob struct {
	connection         *drivers.DbConnection
	ContingencyService contingency.ContingencyManager
	IsRunning          atomic.Bool
	MaxExecutionTime   time.Duration
	Bus                event.Bus
}

func NewRetransmissionJob(contingencyService contingency.ContingencyManager, connection *drivers.DbConnection) *RetransmissionJob {
	return &RetransmissionJob{
		connection:         connection,
		ContingencyService: contingencyService,
		MaxExecutionTime:   10 * time.Minute,
	}
}

func (j *RetransmissionJob) SetEventBus(bus event.Bus) {
	j.Bus = bus
}

// Execute ejecuta el trabajo de retransmisión de documentos en contingencia.
func (j *RetransmissionJob) Execute() {
	if !j.IsRunning.CompareAndSwap(false, true) {
		logs.Warn("Job already running, skipping execution")
		return
	}
	defer j.IsRunning.Store(false)

	ctx, cancel := context.WithTimeout(context.Background(), j.MaxExecutionTime)
	defer cancel()

	logs.Info("Starting retransmission job", map[string]interface{}{
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

	if err := j.ContingencyService.RetransmitPendingDocuments(ctx); err != nil {
		j.handleExecutionError(err)
		return
	}

	logs.Info("Retransmission job completed successfully", map[string]interface{}{
		"timestamp": utils.TimeNow().Format(time.RFC3339),
	})
}

func (j *RetransmissionJob) handleExecutionError(err error) {
	if errors.Is(err, context.DeadlineExceeded) {
		logs.Error("Job execution timed out", map[string]interface{}{
			"MaxExecutionTime": j.MaxExecutionTime,
			"error":            err.Error(),
		})
	} else {
		logs.Error("Job execution failed", map[string]interface{}{
			"error": err.Error(),
		})
	}

	if j.Bus != nil {
		j.Bus.Publish(context.Background(), event.RetransmissionJobFailedEvent{
			JobName:        "contingency_retransmission",
			LastError:      err.Error(),
			OccurredAtTime: time.Now(),
		})
	}
}
