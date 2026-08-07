package ports

import "context"

// SequentialNumberRepositoryPort establishes the methods that a sequential number repository must implement
type SequentialNumberRepositoryPort interface {
	GetNext(ctx context.Context, dteType string, branchID uint) (int, error)
}
