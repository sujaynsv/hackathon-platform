package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/dogfood-platform/dogfood/internal/shared/response"
)

// JWT stub middleware. This will be fully implemented in epic-1-auth.
func JWT(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.HandleDomainError(w, r, response.ErrUnauthorized)
				return
			}
			
			// Extract token
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				response.HandleDomainError(w, r, response.ErrUnauthorized)
				return
			}

			// STUB: Real validation will go here
			// tokenString := parts[1]
			
			// For now, pass through
			ctx := context.WithValue(r.Context(), "user_id", "stub-user-id")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID extracts the user ID from the request context.
func GetUserID(ctx context.Context) string {
	if val, ok := ctx.Value("user_id").(string); ok {
		return val
	}
	return ""
}
