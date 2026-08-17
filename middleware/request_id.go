package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type contextKey string

const RequestIDKey contextKey = "requestID"

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get("X-Request-Id"))
		if id == "" {
			id = uuid.New().String()
		}
		w.Header().Set("X-Request-Id", id)
		ctx := context.WithValue(r.Context(), RequestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetRequestID(r *http.Request) string {
	if id, ok := r.Context().Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}
