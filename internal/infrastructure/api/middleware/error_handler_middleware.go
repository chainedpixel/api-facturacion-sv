package middleware

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"runtime/debug"

	"github.com/chainedpixel/ordo-factus/internal/infrastructure/api/response"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
)

type ErrorMiddleware struct {
	responseWriter *response.ResponseWriter
}

// NewErrorMiddleware creates a new instance of ErrorMiddleware
func NewErrorMiddleware() *ErrorMiddleware {
	return &ErrorMiddleware{
		responseWriter: response.NewResponseWriter(),
	}
}

// Handler is a middleware that captures panic errors and client errors.
func (m *ErrorMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				stackTrace := string(debug.Stack())
				logs.Error("Recovered from panic", map[string]interface{}{
					"error":      err,
					"stackTrace": stackTrace,
					"path":       r.URL.Path,
					"method":     r.Method,
				})

				m.responseWriter.Error(
					w,
					http.StatusInternalServerError,
					"An unexpected error occurred",
					[]string{fmt.Sprintf("%v", err)},
				)
			}
		}()

		sw := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r)

		if sw.status >= 400 && !sw.written {
			var msg string
			switch sw.status {
			case http.StatusNotFound:
				msg = "Resource not found"
			case http.StatusMethodNotAllowed:
				msg = "Method not allowed"
			case http.StatusBadRequest:
				msg = "Bad request"
			default:
				msg = http.StatusText(sw.status)
			}

			logs.Warn("Client error response", map[string]interface{}{
				"status":  sw.status,
				"message": msg,
				"path":    r.URL.Path,
				"method":  r.Method,
			})
			m.responseWriter.Error(w, sw.status, msg, nil)
		} else if sw.status >= 500 {
			logs.Error("Server error response", map[string]interface{}{
				"status": sw.status,
				"path":   r.URL.Path,
				"method": r.Method,
			})
		}
	})
}

// statusWriter is a wrapper for http.ResponseWriter that captures the status code
type statusWriter struct {
	http.ResponseWriter
	status  int
	written bool
}

// WriteHeader captures the status code
func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.written = true
	w.ResponseWriter.WriteHeader(status)
}

// Write captures the status code if it has not been written before
func (w *statusWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.written = true
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

// Hijack implements the http.Hijacker interface if needed
func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("hijacking not supported")
}
