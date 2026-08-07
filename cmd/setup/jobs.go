package setup

import (
	"fmt"
	"time"

	"github.com/chainedpixel/ordo-factus/config/drivers"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/event"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/contingency"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents"
	"github.com/chainedpixel/ordo-factus/internal/domain/ports"
	"github.com/go-co-op/gocron"

	"github.com/chainedpixel/ordo-factus/internal/infrastructure/jobs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

type JobConfig struct {
	StartTime   string
	EndTime     string
	Interval    int
	Environment string
}

func SetupJobs(contingencyService contingency.ContingencyManager, reservedSequenceRepo dte_documents.ReservedSequenceRepositoryPort, ambientCode string, connection *drivers.DbConnection, cache ports.CacheManager, bus event.Bus) error {
	scheduler := gocron.NewScheduler(time.UTC)
	retransmissionJob := jobs.NewRetransmissionJob(contingencyService, connection)
	if bus != nil {
		retransmissionJob.SetEventBus(bus)
	}
	cleanerJob := jobs.NewReservationCleanerJob(reservedSequenceRepo, connection)
	metricsCleanupJob := jobs.NewMetricsCleanupJob(cache)

	var jobConfig JobConfig
	if ambientCode == "01" {
		jobConfig = JobConfig{
			StartTime:   "22:00",
			EndTime:     "05:00",
			Interval:    30,
			Environment: "production",
		}
	} else {
		jobConfig = JobConfig{
			StartTime:   "08:00",
			EndTime:     "17:00",
			Interval:    5,
			Environment: "testing",
		}
	}

	if err := ScheduleContingencyJob(scheduler, retransmissionJob, jobConfig); err != nil {
		logs.Error("Failed to setup contingency job", map[string]interface{}{
			"error":  err.Error(),
			"config": jobConfig,
		})
		return err
	}

	if err := ScheduleReservationCleanerJob(scheduler, cleanerJob); err != nil {
		logs.Error("Failed to setup reservation cleaner job", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}

	if err := ScheduleMetricsCleanupJob(scheduler, metricsCleanupJob); err != nil {
		logs.Error("Failed to setup metrics cleanup job", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}

	logs.Info("Jobs scheduled successfully", map[string]interface{}{
		"environment": jobConfig.Environment,
		"startTime":   jobConfig.StartTime,
		"endTime":     jobConfig.EndTime,
		"interval":    jobConfig.Interval,
	})

	scheduler.StartAsync()
	return nil
}

func ScheduleContingencyJob(scheduler *gocron.Scheduler, job *jobs.RetransmissionJob, config JobConfig) error {
	_, err := scheduler.Every(config.Interval).Minutes().Do(func() {
		logs.Info("Starting contingency job execution", map[string]interface{}{
			"environment": config.Environment,
			"timestamp":   utils.TimeNow().Format(time.RFC3339),
		})
		job.Execute()
	})

	if err != nil {
		return fmt.Errorf("failed to schedule contingency job: %w", err)
	}

	return nil
}

func ScheduleReservationCleanerJob(scheduler *gocron.Scheduler, job *jobs.ReservationCleanerJob) error {
	_, err := scheduler.Every(10).Minutes().Do(func() {
		logs.Info("Starting reservation cleaner job execution", map[string]interface{}{
			"timestamp": utils.TimeNow().Format(time.RFC3339),
		})
		job.Execute()
	})

	if err != nil {
		return fmt.Errorf("failed to schedule reservation cleaner job: %w", err)
	}

	return nil
}

func ScheduleMetricsCleanupJob(scheduler *gocron.Scheduler, job *jobs.MetricsCleanupJob) error {
	_, err := scheduler.Every(6).Hours().Do(func() {
		logs.Info("Starting metrics cleanup job execution", map[string]interface{}{
			"timestamp": utils.TimeNow().Format(time.RFC3339),
		})
		job.Execute()
	})

	if err != nil {
		return fmt.Errorf("failed to schedule metrics cleanup job: %w", err)
	}

	return nil
}
