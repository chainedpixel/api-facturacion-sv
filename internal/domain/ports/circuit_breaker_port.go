package ports

import "github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"

// CircuitManager defines an interface for circuit breaker implementations
type CircuitManager interface {
	AllowRequest() bool
	RecordSuccess()
	RecordFailure()
	GetState() constants.State
	GetFailureCount() int32
}
