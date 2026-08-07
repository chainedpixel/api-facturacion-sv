package health

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/core/event"
	"github.com/chainedpixel/ordo-factus/internal/domain/health"
	"github.com/chainedpixel/ordo-factus/internal/domain/health/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/health/models"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/health/checkers"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
	"gorm.io/gorm"
)

type healthService struct {
	checkers []health.ComponentChecker
}

type HealthServiceConfig struct {
	DB  *gorm.DB
	Bus event.Bus
}

func NewHealthService(cfg *HealthServiceConfig) health.HealthManager {
	service := &healthService{
		checkers: []health.ComponentChecker{
			checkers.NewDatabaseChecker(cfg.DB),
			checkers.NewRedisChecker(),
			checkers.NewHaciendaChecker(),
			checkers.NewFileSystemChecker(),
			checkers.NewSignerChecker(),
			checkers.NewDomainEventsChecker(cfg.Bus),
			checkers.NewSMTPChecker(),
		},
	}
	return service
}

func (s *healthService) CheckHealth() (*models.HealthStatus, error) {
	components := make(map[string]models.Health)
	status := constants.StatusUp

	for _, checker := range s.checkers {
		health := checker.Check()
		components[checker.Name()] = health

		if health.Status == constants.StatusDown {
			status = constants.StatusDown
		}
	}

	return &models.HealthStatus{
		Status:     status,
		Components: components,
		Timestamp:  utils.TimeNow().Format("02-01-2006 15:04:05"),
	}, nil
}
