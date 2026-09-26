package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dogfood-platform/dogfood/internal/auth/port"
	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/golang-jwt/jwt/v5"
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

// GetJTI extracts the token JTI from the request context.
func GetJTI(ctx context.Context) string {
	if val, ok := ctx.Value("jti").(string); ok {
		return val
	}
	return ""
}

// GetTokenExpiry extracts the token expiry from the request context.
func GetTokenExpiry(ctx context.Context) time.Time {
	if val, ok := ctx.Value("exp").(time.Time); ok {
		return val
	}
	return time.Time{}
}

// JWTMiddleware validates the JWT and checks if it is blacklisted.
func JWTMiddleware(secret string, cache port.Cache) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.HandleDomainError(w, r, fmt.Errorf("%w: missing authorization header", response.ErrUnauthorized))
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				response.HandleDomainError(w, r, fmt.Errorf("%w: invalid authorization header format", response.ErrUnauthorized))
				return
			}

			tokenString := parts[1]

			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method")
				}
				return []byte(secret), nil
			})

			if err != nil || !token.Valid {
				response.HandleDomainError(w, r, fmt.Errorf("%w: invalid or expired token", response.ErrUnauthorized))
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				response.HandleDomainError(w, r, fmt.Errorf("%w: invalid token claims", response.ErrUnauthorized))
				return
			}

			jti, _ := claims["jti"].(string)
			if jti == "" {
				response.HandleDomainError(w, r, fmt.Errorf("%w: token missing jti", response.ErrUnauthorized))
				return
			}

			// Check if blacklisted in Redis
			key := fmt.Sprintf("revoked:%s", jti)
			val, err := cache.Get(r.Context(), key)
			if err != nil {
				// Fail closed on storage errors
				response.HandleDomainError(w, r, fmt.Errorf("blacklist check failed: %w", err))
				return
			}
			if val != nil {
				// Token is revoked
				response.HandleDomainError(w, r, response.ErrTokenRevoked)
				return
			}

			// Extract claims
			userID, _ := claims["sub"].(string)
			expFloat, _ := claims["exp"].(float64)
			exp := time.Unix(int64(expFloat), 0)

			ctx := context.WithValue(r.Context(), "user_id", userID)
			ctx = context.WithValue(ctx, "jti", jti)
			ctx = context.WithValue(ctx, "exp", exp)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalJWTMiddleware validates a bearer token when one is supplied, while
// allowing anonymous requests through for public resources.
func OptionalJWTMiddleware(secret string, cache port.Cache) func(http.Handler) http.Handler {
	requireJWT := JWTMiddleware(secret, cache)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.TrimSpace(r.Header.Get("Authorization")) == "" {
				next.ServeHTTP(w, r)
				return
			}
			requireJWT(next).ServeHTTP(w, r)
		})
	}
}
