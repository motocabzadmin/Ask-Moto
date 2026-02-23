package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/moto/ask-moto/internal/identity"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const (
	// UserIDKey is the context key for the authenticated user's ID
	UserIDKey ContextKey = "userID"
	// RoleKey is the context key for the user's role (driver or rider)
	RoleKey ContextKey = "role"
	// ClaimsKey is the context key for the full claims object
	ClaimsKey ContextKey = "claims"
)

// AuthMiddleware creates an HTTP middleware that validates JWT tokens
// from motocabz driver/rider services and extracts the user_id
func AuthMiddleware(tokenService *identity.TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get the Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondUnauthorized(w, "Authorization header required")
				return
			}

			// Extract Bearer token
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				respondUnauthorized(w, "Invalid authorization header format")
				return
			}
			tokenString := parts[1]

			// Validate token
			claims, err := tokenService.ValidateToken(tokenString)
			if err != nil {
				respondUnauthorized(w, err.Error())
				return
			}

			// Add user info to request context
			ctx := r.Context()
			ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, RoleKey, claims.Role)
			ctx = context.WithValue(ctx, ClaimsKey, claims)

			// Continue with the next handler
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuthMiddleware extracts user info if present but doesn't require authentication
func OptionalAuthMiddleware(tokenService *identity.TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
					if claims, err := tokenService.ValidateToken(parts[1]); err == nil {
						ctx := r.Context()
						ctx = context.WithValue(ctx, UserIDKey, claims.UserID)
						ctx = context.WithValue(ctx, RoleKey, claims.Role)
						ctx = context.WithValue(ctx, ClaimsKey, claims)
						r = r.WithContext(ctx)
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// GetUserID extracts the user ID from the request context
func GetUserID(r *http.Request) string {
	if userID, ok := r.Context().Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

// GetRole extracts the user role from the request context
func GetRole(r *http.Request) string {
	if role, ok := r.Context().Value(RoleKey).(string); ok {
		return role
	}
	return ""
}

// GetClaims extracts the full claims from the request context
func GetClaims(r *http.Request) *identity.Claims {
	if claims, ok := r.Context().Value(ClaimsKey).(*identity.Claims); ok {
		return claims
	}
	return nil
}

// IsDriver checks if the authenticated user is a driver
func IsDriver(r *http.Request) bool {
	return GetRole(r) == identity.RoleDriver
}

// IsRider checks if the authenticated user is a rider
func IsRider(r *http.Request) bool {
	return GetRole(r) == identity.RoleRider
}

func respondUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
