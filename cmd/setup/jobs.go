package setup

import (
	"fmt"
	"github.com/MarlonG1/api-facturacion-sv/config/drivers"
	"github.com/MarlonG1/api-facturacion-sv/internal/domain/dte/contingency"
	"github.com/go-co-op/gocron"
	"time"

	"github.com/MarlonG1/api-facturacion-sv/internal/infrastructure/jobs"
	"github.com/MarlonG1/api-facturacion-sv/pkg/shared/logs"
	"github.com/MarlonG1/api-facturacion-sv/pkg/shared/utils"
)

type JobConfig struct {
	StartTime   string
	EndTime     string
	Interval    int
	Environment string
}

func SetupJobs(contingencyService contingency.ContingencyManager, ambientCode string, connection *drivers.DbConnection) error {
	scheduler := gocron.NewScheduler(time.UTC)
	job := jobs.NewRetransmissionJob(contingencyService, connection)

	// Configuración segun ambiente
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

	if err := ScheduleContingencyJob(scheduler, job, jobConfig); err != nil {
		logs.Error("Failed to setup contingency job", map[string]interface{}{
			"error":  err.Error(),
			"config": jobConfig,
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
