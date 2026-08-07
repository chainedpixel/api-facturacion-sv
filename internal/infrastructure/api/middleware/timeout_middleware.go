package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/chainedpixel/ordo-factus/config"
	"github.com/chainedpixel/ordo-factus/internal/infrastructure/api/response"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
)

type TimeoutMiddleware struct {
	responseWriter *response.ResponseWriter
}

func NewTimeoutMiddleware() *TimeoutMiddleware {
	return &TimeoutMiddleware{
		responseWriter: response.NewResponseWriter(),
	}
}

func (m *TimeoutMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 14*time.Second)
		defer cancel()

		r = r.WithContext(ctx)
		rw := &responseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		done := make(chan struct{})
		go func() {
			next.ServeHTTP(rw, r.WithContext(ctx))
			close(done)
		}()

		select {
		case <-done:
			return
		case <-ctx.Done():
			timeoutTitle := config.TranslateServiceArgs("RequestTimeOutTitle")
			timeoutMessage := config.TranslateServiceArgs("RequestTimeOut")

			logs.Warn("Request timed out", map[string]interface{}{
				"method":              r.Method,
				"path":                r.URL.Path,
				"error":               ctx.Err(),
				"error shown":         timeoutMessage,
				"error message shown": timeoutTitle,
			})
			if !rw.written {
				m.responseWriter.Error(
					w,
					http.StatusRequestTimeout,
					timeoutTitle,
					[]string{timeoutMessage},
				)
			}
		}
	})
}
