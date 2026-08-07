package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/auth/models"
	metricsModels "github.com/chainedpixel/ordo-factus/internal/domain/metrics/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/ports"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

type MetricsMiddleware struct {
	cache      ports.CacheManager
	maxMetrics int
}

var (
	uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

	endpointMappings = map[string]string{
		"GET:/api/v1/dte":               "dte",
		"GET:/api/v1/dte/{id}":          "dte/{id}",
		"POST:/api/v1/dte/invoices":     "invoices",
		"POST:/api/v1/dte/ccf":          "ccf",
		"POST:/api/v1/dte/invalidation": "invalidation",
		"POST:/api/v1/dte/retention":    "retention",
		"POST:/api/v1/dte/creditnote":   "creditnote",
	}
)

func NewMetricsMiddleware(cache ports.CacheManager) *MetricsMiddleware {
	return &MetricsMiddleware{
		cache:      cache,
		maxMetrics: 20,
	}
}

func extractEndpoint(method, path string) string {
	path = strings.TrimSuffix(path, "/")

	if endpoint, exists := endpointMappings[method+":"+path]; exists {
		return endpoint
	}

	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/dte/"), "/")

	if len(parts) == 0 || parts[0] == "" {
		return "dte"
	}

	if uuidRegex.MatchString(parts[0]) {
		templatePath := "/api/v1/dte/{id}"
		if len(parts) > 1 {
			templatePath += "/" + strings.Join(parts[1:], "/")
		}

		if endpoint, exists := endpointMappings[method+":"+templatePath]; exists {
			return endpoint
		}

		if method == "GET" {
			return "consult"
		}
	}

	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}

	return "dte"
}

func (m *MetricsMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
			written:        false,
		}

		systemNIT := r.Context().Value("claims").(*models.AuthClaims).NIT

		start := utils.TimeNow()
		next.ServeHTTP(rw, r)
		duration := time.Since(start)

		endpoint := extractEndpoint(r.Method, r.URL.Path)
		durationsKey := fmt.Sprintf("metrics:%s:%s:%s:durations", systemNIT, r.Method, endpoint)
		countersKey := fmt.Sprintf("metrics:%s:%s:%s:counters", systemNIT, r.Method, endpoint)

		metric := metricsModels.RequestMetric{
			Duration:   duration.Milliseconds(),
			Timestamp:  utils.TimeNow(),
			StatusCode: rw.status,
		}

		metricData, _ := json.Marshal(metric)

		if err := m.cache.LPush(durationsKey, metricData); err != nil {
			logs.Error("Failed to store metric", map[string]interface{}{
				"error": err.Error(),
			})
		}

		if err := m.cache.LTrim(durationsKey, 0, 19); err != nil {
			logs.Error("Failed to trim metrics", map[string]interface{}{
				"error": err.Error(),
			})
		}

		if err := m.cache.Expire(durationsKey, 24*time.Hour); err != nil {
			logs.Error("Failed to set expiration on metrics", map[string]interface{}{
				"error": err.Error(),
			})
		}

		var counters metricsModels.EndpointCounters
		if countersData, err := m.cache.Get(countersKey); err == nil {
			json.Unmarshal([]byte(countersData), &counters)
		}

		counters.TotalRequests++
		if rw.status >= 200 && rw.status < 300 {
			counters.SuccessCount++
		} else {
			counters.ErrorCount++
		}

		if countersData, err := json.Marshal(counters); err == nil {
			if err := m.cache.Set(countersKey, countersData, 24*time.Hour); err != nil {
				logs.Error("Failed to update counters", map[string]interface{}{
					"error": err.Error(),
				})
			}
		}
	})
}

type responseWriter struct {
	http.ResponseWriter
	status  int
	written bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.status = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.status = http.StatusOK
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}
