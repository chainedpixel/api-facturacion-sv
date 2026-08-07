package jobs

import (
	"sync/atomic"
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/ports"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

const metricsTTL = 24 * time.Hour

type MetricsCleanupJob struct {
	cache     ports.CacheManager
	IsRunning atomic.Bool
}

func NewMetricsCleanupJob(cache ports.CacheManager) *MetricsCleanupJob {
	return &MetricsCleanupJob{
		cache: cache,
	}
}

// Execute scans for orphaned metrics keys without TTL and sets expiration on them
func (j *MetricsCleanupJob) Execute() {
	if !j.IsRunning.CompareAndSwap(false, true) {
		logs.Warn("Metrics cleanup job already running, skipping execution")
		return
	}
	defer j.IsRunning.Store(false)

	logs.Info("Starting metrics cleanup job", map[string]interface{}{
		"timestamp": utils.TimeNow().Format(time.RFC3339),
	})

	cleaned := j.cleanPattern("metrics:*:durations")
	cleaned += j.cleanPattern("metrics:*:counters")

	logs.Info("Metrics cleanup job completed", map[string]interface{}{
		"keysProcessed": cleaned,
		"timestamp":     utils.TimeNow().Format(time.RFC3339),
	})
}

func (j *MetricsCleanupJob) cleanPattern(pattern string) int64 {
	keys, err := j.cache.ScanKeys(pattern)
	if err != nil {
		logs.Error("Failed to scan metrics keys", map[string]interface{}{
			"pattern": pattern,
			"error":   err.Error(),
		})
		return 0
	}

	var processed int64
	for _, key := range keys {
		if err := j.cache.Expire(key, metricsTTL); err != nil {
			logs.Error("Failed to set TTL on metrics key", map[string]interface{}{
				"key":   key,
				"error": err.Error(),
			})
			continue
		}
		processed++
	}

	return processed
}
