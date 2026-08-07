package models

import "time"

type Metrics struct {
	ProcessedDTEs DTEMetrics         `json:"processed_dtes"`
	Contingency   ContingencyMetrics `json:"contingency"`
	Timestamp     string             `json:"timestamp"`
}

type DTEMetrics struct {
	Total     int64            `json:"total"`
	LastMonth int64            `json:"last_month"`
	ByType    map[string]int64 `json:"by_type"`
	ByStatus  map[string]int64 `json:"by_status"`
}

type RequestMetric struct {
	Duration   int64     `json:"duration"`
	Timestamp  time.Time `json:"timestamp"`
	StatusCode int       `json:"status_code"`
}

type ContingencyMetrics struct {
	PendingDTEs int64 `json:"pending_dtes"`
}

type EndpointMetrics struct {
	Path           string  `json:"path"`
	Method         string  `json:"method"`
	SystemNIT      string  `json:"system_nit"`
	TotalRequests  int64   `json:"total_requests"`
	SuccessCount   int64   `json:"success_count"`
	ErrorCount     int64   `json:"error_count"`
	CurrentAverage int64   `json:"current_average_ms"`
	MaxDuration    int64   `json:"max_duration_ms"`
	MinDuration    int64   `json:"min_duration_ms"`
	LastDurations  []int64 `json:"last_durations"`
}

type EndpointCounters struct {
	TotalRequests int64 `json:"total_requests"`
	SuccessCount  int64 `json:"success_count"`
	ErrorCount    int64 `json:"error_count"`
}
