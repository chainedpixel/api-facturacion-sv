package ports

import "context"

// DTEService is an interface for DTE services
type DTEService interface {
	Create(ctx context.Context, data interface{}, branchID uint) (interface{}, error)
}
