package services

import (
	"testing"
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/adapters/circuit"
	test "github.com/chainedpixel/ordo-factus/tests"
	"github.com/stretchr/testify/assert"
)

// TestCircuitBreaker_InitialState verifies the breaker starts closed.
func TestCircuitBreaker_InitialState(t *testing.T) {
	test.TestMain(t)

	cb := circuit.NewCircuitBreaker(3, 10*time.Second)

	assert.Equal(t, constants.StateClosed, cb.GetState())
	assert.Equal(t, int32(0), cb.GetFailureCount())
	assert.True(t, cb.AllowRequest())
}

// TestCircuitBreaker_OpensAfterThreshold verifies the breaker transitions to Open
// once failure count reaches the threshold.
func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	test.TestMain(t)

	cb := circuit.NewCircuitBreaker(3, 10*time.Second)

	cb.RecordFailure()
	assert.Equal(t, constants.StateClosed, cb.GetState())
	assert.True(t, cb.AllowRequest())

	cb.RecordFailure()
	assert.Equal(t, constants.StateClosed, cb.GetState())

	cb.RecordFailure() // hits threshold
	assert.Equal(t, constants.StateOpen, cb.GetState())
	assert.Equal(t, int32(3), cb.GetFailureCount())
	assert.False(t, cb.AllowRequest())
}

// TestCircuitBreaker_SuccessResetsClosed verifies RecordSuccess closes the circuit
// and resets the failure counter.
func TestCircuitBreaker_SuccessResetsClosed(t *testing.T) {
	test.TestMain(t)

	cb := circuit.NewCircuitBreaker(2, 10*time.Second)

	cb.RecordFailure()
	cb.RecordFailure() // opens
	assert.Equal(t, constants.StateOpen, cb.GetState())

	cb.RecordSuccess()
	assert.Equal(t, constants.StateClosed, cb.GetState())
	assert.Equal(t, int32(0), cb.GetFailureCount())
	assert.True(t, cb.AllowRequest())
}

// TestCircuitBreaker_HalfOpenAfterResetTimeout verifies the breaker transitions
// to Half-Open after the reset timeout has elapsed.
func TestCircuitBreaker_HalfOpenAfterResetTimeout(t *testing.T) {
	test.TestMain(t)

	// Use a very short reset time so the test doesn't actually sleep long.
	cb := circuit.NewCircuitBreaker(1, 10*time.Millisecond)

	cb.RecordFailure() // opens
	assert.Equal(t, constants.StateOpen, cb.GetState())
	assert.False(t, cb.AllowRequest()) // blocked while open

	// Wait for the reset window to elapse.
	time.Sleep(20 * time.Millisecond)

	// AllowRequest should now transition to Half-Open and return true.
	assert.True(t, cb.AllowRequest())
	assert.Equal(t, constants.StateHalfOpen, cb.GetState())
}

// TestCircuitBreaker_HalfOpenClosesOnSuccess verifies that a success in
// Half-Open state fully closes the circuit.
func TestCircuitBreaker_HalfOpenClosesOnSuccess(t *testing.T) {
	test.TestMain(t)

	cb := circuit.NewCircuitBreaker(1, 10*time.Millisecond)

	cb.RecordFailure()
	time.Sleep(20 * time.Millisecond)
	cb.AllowRequest() // transitions to HalfOpen

	cb.RecordSuccess()
	assert.Equal(t, constants.StateClosed, cb.GetState())
	assert.True(t, cb.AllowRequest())
}

// TestCircuitBreaker_ExtraFailuresBeyondThreshold verifies that recording
// additional failures beyond the threshold keeps the circuit Open without
// duplicate state transitions.
func TestCircuitBreaker_ExtraFailuresBeyondThreshold(t *testing.T) {
	test.TestMain(t)

	cb := circuit.NewCircuitBreaker(2, 10*time.Second)

	cb.RecordFailure()
	cb.RecordFailure() // threshold reached → Open
	cb.RecordFailure() // extra failure
	cb.RecordFailure() // extra failure

	assert.Equal(t, constants.StateOpen, cb.GetState())
	assert.Equal(t, int32(4), cb.GetFailureCount())
	assert.False(t, cb.AllowRequest())
}

// TestCircuitBreaker_ThresholdOne verifies a breaker with threshold=1 opens
// on the very first failure.
func TestCircuitBreaker_ThresholdOne(t *testing.T) {
	test.TestMain(t)

	cb := circuit.NewCircuitBreaker(1, 5*time.Second)

	assert.True(t, cb.AllowRequest())
	cb.RecordFailure()
	assert.Equal(t, constants.StateOpen, cb.GetState())
	assert.False(t, cb.AllowRequest())
}
