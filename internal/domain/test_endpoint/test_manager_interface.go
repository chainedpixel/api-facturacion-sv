package test_endpoint

import (
	"context"

	"github.com/chainedpixel/api-facturacion-sv/internal/domain/test_endpoint/models"
)

// TestManager is an interface that defines the methods that a system test service must implement.
type TestManager interface {
	RunSystemTest(ctx context.Context) (*models.TestResult, error)
}
