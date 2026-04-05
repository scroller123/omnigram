package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const (
	SessionIDKey contextKey = "session_id"
)

// SessionIDMiddleware extracts X-Session-ID header.
func SessionIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionID := r.Header.Get("X-Session-ID")
		if sessionID == "" {
			http.Error(w, "X-Session-ID header required", http.StatusBadRequest)
			return
		}
		ctx := context.WithValue(r.Context(), SessionIDKey, sessionID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetSessionID(ctx context.Context) string {
	if v, ok := ctx.Value(SessionIDKey).(string); ok {
		return v
	}
	return ""
}

// Middleware returns a chi-compatible middleware that validates JWT Bearer tokens.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, "Authorization header must be 'Bearer <token>'", http.StatusUnauthorized)
			return
		}

		if _, err := ValidateToken(parts[1]); err != nil {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
