package models

import (
	"time"

	"github.com/chainedpixel/ordo-factus/config"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/contingency/models"
)

// TransmissionConfig specific configuration for the contingency service
type TransmissionConfig struct {
	Ambient       string
	BatchSize     int
	RetryInterval time.Duration
	MaxInterval   time.Duration
	BackoffFactor float64
}

// GetAmbient returns the configured environment
func (c *TransmissionConfig) GetAmbient() string {
	return c.Ambient
}

// GetBatchSize returns the configured batch size
func (c *TransmissionConfig) GetBatchSize() int {
	return c.BatchSize
}

// GetRetryInterval returns the initial retry interval
func (c *TransmissionConfig) GetRetryInterval() time.Duration {
	return c.RetryInterval
}

// GetMaxInterval returns the maximum retry interval
func (c *TransmissionConfig) GetMaxInterval() time.Duration {
	return c.MaxInterval
}

// GetBackoffFactor returns the growth factor for exponential backoff
func (c *TransmissionConfig) GetBackoffFactor() float64 {
	return c.BackoffFactor
}

// GetRetryPolicy builds a retry policy
func (c *TransmissionConfig) GetRetryPolicy() models.RetryPolicy {
	return models.RetryPolicy{
		MaxAttempts:     3,
		InitialInterval: c.RetryInterval,
		MaxInterval:     c.MaxInterval,
		BackoffFactor:   c.BackoffFactor,
	}
}

func NewTransmissionConfig(retryInterval, maxInterval time.Duration, backoffFactor float64) *TransmissionConfig {
	return &TransmissionConfig{
		Ambient:       config.Server.AmbientCode,
		BatchSize:     config.Server.MaxBatchSize,
		RetryInterval: retryInterval,
		MaxInterval:   maxInterval,
		BackoffFactor: backoffFactor,
	}
}
