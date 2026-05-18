package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/unalkalkan/TwelveReader/internal/identity"
	"github.com/unalkalkan/TwelveReader/pkg/types"
)

type sessionIDKey struct{}
type userContextKey struct{}

// SessionAuthMiddleware creates middleware that validates the session token from the Authorization header.
// If valid, it injects the session ID and user into the request context.
func SessionAuthMiddleware(authService *identity.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			session, err := authService.GetSessionByTokenHash(r.Context(), token)
			if err != nil {
				// Token invalid - pass through; the handler will return 401 if it needs auth
				next.ServeHTTP(w, r)
				return
			}

			user, err := authService.GetUserByID(r.Context(), session.UserID)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), sessionIDKey{}, session.ID)
			ctx = context.WithValue(ctx, userContextKey{}, user)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth is a middleware that enforces authentication. Returns 401 if no valid session.
func RequireAuth(authService *identity.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				WriteStructuredError(w, r, ErrCodeUnauthorized, "authentication required", http.StatusUnauthorized)
				return
			}

			session, err := authService.GetSessionByTokenHash(r.Context(), token)
			if err != nil {
				WriteStructuredError(w, r, ErrCodeUnauthorized, "invalid or expired session", http.StatusUnauthorized)
				return
			}

			user, err := authService.GetUserByID(r.Context(), session.UserID)
			if err != nil {
				WriteStructuredError(w, r, ErrCodeUnauthorized, "user not found", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), sessionIDKey{}, session.ID)
			ctx = context.WithValue(ctx, userContextKey{}, user)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole checks if the authenticated user has the required role name.
func RequireRole(pool *identity.DBPool, requiredRoleName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUserFromContext(r.Context())
			if user == nil {
				WriteStructuredError(w, r, ErrCodeUnauthorized, "authentication required", http.StatusUnauthorized)
				return
			}

			role, err := pool.Roles.GetRoleByID(r.Context(), user.RoleID)
			if err != nil || role.Name != requiredRoleName {
				WriteStructuredError(w, r, ErrCodeForbidden, "insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetSessionIDFromContext extracts the session ID from request context.
func GetSessionIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(sessionIDKey{}).(string); ok {
		return v
	}
	return ""
}

// GetUserFromContext extracts the user from request context.
func GetUserFromContext(ctx context.Context) *types.User {
	if v, ok := ctx.Value(userContextKey{}).(*types.User); ok {
		return v
	}
	return nil
}

// extractBearerToken extracts the Bearer token from the Authorization header.
func extractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) > 7 && strings.ToUpper(authHeader[:7]) == "BEARER " {
		return strings.TrimSpace(authHeader[7:])
	}
	return ""
}
