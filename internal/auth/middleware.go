package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/render"
)

type contextKey string

const (
	userEmailKey  contextKey = "user_email"
	userRoleKey   contextKey = "user_role"
	userClaimsKey contextKey = "user_claims"
)

// AuthMiddleware creates a middleware that validates JWT tokens.
func AuthMiddleware(authService *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, map[string]string{"error": "authorization header required"})

				return
			}

			// Check if it's a Bearer token
			if !strings.HasPrefix(authHeader, "Bearer ") {
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, map[string]string{"error": "invalid authorization header format"})

				return
			}

			// Extract token
			token := strings.TrimPrefix(authHeader, "Bearer ")

			// Validate token
			claims, err := authService.ValidateToken(token)
			if err != nil {
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, map[string]string{"error": "invalid token"})

				return
			}

			// Add user info to context
			ctx := context.WithValue(r.Context(), userEmailKey, claims.Email)
			ctx = context.WithValue(ctx, userRoleKey, claims.Role)
			ctx = context.WithValue(ctx, userClaimsKey, claims)

			// Continue with the request
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuthMiddleware creates a middleware that validates JWT tokens if present.
// This is useful for endpoints that can work with or without authentication.
func OptionalAuthMiddleware(authService *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// No token provided, continue without authentication
				next.ServeHTTP(w, r)

				return
			}

			// Check if it's a Bearer token
			if !strings.HasPrefix(authHeader, "Bearer ") {
				// Invalid format, continue without authentication
				next.ServeHTTP(w, r)

				return
			}

			// Extract token
			token := strings.TrimPrefix(authHeader, "Bearer ")

			// Validate token
			claims, err := authService.ValidateToken(token)
			if err != nil {
				// Invalid token, continue without authentication
				next.ServeHTTP(w, r)

				return
			}

			// Add user info to context
			ctx := context.WithValue(r.Context(), userEmailKey, claims.Email)
			ctx = context.WithValue(ctx, userRoleKey, claims.Role)
			ctx = context.WithValue(ctx, userClaimsKey, claims)

			// Continue with the request
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserFromContext extracts user information from the request context.
func GetUserFromContext(ctx context.Context) (string, string, bool) {
	emailVal := ctx.Value(userEmailKey)
	roleVal := ctx.Value(userRoleKey)

	if emailVal == nil || roleVal == nil {
		return "", "", false
	}

	email, ok1 := emailVal.(string)
	role, ok2 := roleVal.(string)

	if !ok1 || !ok2 {
		return "", "", false
	}

	return email, role, true
}

// GetClaimsFromContext extracts JWT claims from the request context.
func GetClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claimsVal := ctx.Value(userClaimsKey)
	if claimsVal == nil {
		return nil, false
	}

	claims, ok := claimsVal.(*Claims)

	return claims, ok
}

// RequirePermission creates a middleware that requires a specific permission.
func RequirePermission(permission Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user role from context
			_, role, ok := GetUserFromContext(r.Context())
			if !ok {
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, map[string]string{"error": "authentication required"})

				return
			}

			// Check if user has the required permission
			userRole := GetRole(role)
			if !userRole.HasPermission(permission) {
				render.Status(r, http.StatusForbidden)
				render.JSON(w, r, map[string]string{"error": "insufficient permissions"})

				return
			}

			// Continue with the request
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyPermission creates a middleware that requires any of the specified permissions.
func RequireAnyPermission(permissions ...Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user role from context
			_, role, ok := GetUserFromContext(r.Context())
			if !ok {
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, map[string]string{"error": "authentication required"})

				return
			}

			// Check if user has any of the required permissions
			userRole := GetRole(role)
			if !userRole.HasAnyPermission(permissions...) {
				render.Status(r, http.StatusForbidden)
				render.JSON(w, r, map[string]string{"error": "insufficient permissions"})

				return
			}

			// Continue with the request
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAllPermissions creates a middleware that requires all of the specified permissions.
func RequireAllPermissions(permissions ...Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user role from context
			_, role, ok := GetUserFromContext(r.Context())
			if !ok {
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, map[string]string{"error": "authentication required"})

				return
			}

			// Check if user has all of the required permissions
			userRole := GetRole(role)
			if !userRole.HasAllPermissions(permissions...) {
				render.Status(r, http.StatusForbidden)
				render.JSON(w, r, map[string]string{"error": "insufficient permissions"})

				return
			}

			// Continue with the request
			next.ServeHTTP(w, r)
		})
	}
}
