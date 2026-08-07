package middleware

import (
	"context"
	"net/http"
	"strings"
)

type TokenExtractor struct{}

func NewTokenExtractor() *TokenExtractor {
	return &TokenExtractor{}
}

func (te *TokenExtractor) ExtractToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		parts := strings.Split(authHeader, " ")

		ctx := context.WithValue(r.Context(), "token", parts[1])
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
