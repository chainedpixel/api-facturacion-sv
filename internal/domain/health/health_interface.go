package health

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/health/models"
)

// ComponentChecker is an interface that defines the methods to check the status of a component
type ComponentChecker interface {
	Check() models.Health
	Name() string
}

// HealthManager is an interface that defines the methods to check the status of all components
type HealthManager interface {
	CheckHealth() (*models.HealthStatus, error)
}
