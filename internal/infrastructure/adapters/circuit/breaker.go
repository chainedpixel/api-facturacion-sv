package circuit

import (
	"sync"
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"

	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

// CircuitBreaker implements the circuit breaker pattern to prevent cascading failures.
// It transitions between Closed, Open, and Half-Open states based on failure thresholds.
type CircuitBreaker struct {
	failures    int32
	lastFailure time.Time
	threshold   int32
	resetTime   time.Duration
	state       constants.State
	mu          sync.Mutex
}

// NewCircuitBreaker creates a new CircuitBreaker with the given failure threshold and reset duration.
func NewCircuitBreaker(threshold int32, resetTime time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		threshold: threshold,
		resetTime: resetTime,
		state:     constants.StateClosed,
	}
}

// AllowRequest returns true if the circuit breaker permits a request to proceed.
// In the Open state, it transitions to Half-Open once the reset timeout has elapsed.
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case constants.StateClosed:
		return true
	case constants.StateOpen:
		if time.Since(cb.lastFailure) > cb.resetTime {
			logs.Info("Circuit breaker entering half-open state", map[string]interface{}{
				"lastFailure": cb.lastFailure,
				"resetTime":   cb.resetTime,
			})
			cb.state = constants.StateHalfOpen
			return true
		}
		return false
	case constants.StateHalfOpen:
		return true
	default:
		return false
	}
}

// RecordSuccess resets the failure counter and closes the circuit.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	if cb.state != constants.StateClosed {
		logs.Info("Circuit breaker closing after success", map[string]interface{}{
			"previousState": cb.state,
		})
	}
	cb.state = constants.StateClosed
}

// RecordFailure increments the failure counter and opens the circuit when the threshold is reached.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = utils.TimeNow()

	if cb.failures >= cb.threshold && cb.state != constants.StateOpen {
		logs.Warn("Circuit breaker opening due to failures", map[string]interface{}{
			"failures":  cb.failures,
			"threshold": cb.threshold,
		})
		cb.state = constants.StateOpen
	}
}

// GetState returns the current state of the circuit breaker.
func (cb *CircuitBreaker) GetState() constants.State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// GetFailureCount returns the current number of recorded failures.
func (cb *CircuitBreaker) GetFailureCount() int32 {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.failures
}
