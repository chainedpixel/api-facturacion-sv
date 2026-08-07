package test_endpoint

import (
	"context"

	"github.com/chainedpixel/ordo-factus/internal/domain/test_endpoint/models"
)

// TestManager is an interface that defines the methods that a system test service must implement.
type TestManager interface {
	RunSystemTest(ctx context.Context) (*models.TestResult, error)
}
