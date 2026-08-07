package metrics

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/metrics/models"
)

// MetricsManager is an interface that defines the methods to retrieve and record metrics
type MetricsManager interface {
	GetAllMetricsEndpoint(systemNIT string) (map[string]*models.EndpointMetrics, error)
	GetEndpointMetrics(systemNIT, method, endpoint string) (*models.EndpointMetrics, error)
}
