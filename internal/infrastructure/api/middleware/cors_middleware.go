package middleware

import (
	"net/http"
	"strings"
)

type CorsMiddleware struct {
	allowedOrigins []string
	allowedMethods []string
	allowedHeaders []string
}

func NewCorsMiddleware(origins, methods, headers []string) *CorsMiddleware {
	if len(methods) == 0 {
		methods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}
	if len(headers) == 0 {
		headers = []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"}
	}

	return &CorsMiddleware{
		allowedOrigins: origins,
		allowedMethods: methods,
		allowedHeaders: headers,
	}
}

func (m *CorsMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if len(m.allowedOrigins) > 0 {
			allowed := false
			for _, allowedOrigin := range m.allowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}
			if !allowed {
				origin = ""
			}
		}

		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(m.allowedMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(m.allowedHeaders, ", "))
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
