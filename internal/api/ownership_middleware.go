package api

import (
	"context"
	"net/http"

	"github.com/unalkalkan/TwelveReader/internal/identity"
	"github.com/unalkalkan/TwelveReader/pkg/types"
)

// RequireBookOwnership is middleware that enforces book-level ownership.
// The bookRepo check function receives a resourceID (book ID) and returns the
// book if found. If the requesting user owns the book or is an admin, access passes.
func RequireBookOwnership(
	pool *identity.DBPool,
	getBook func(bookID string) (*types.Book, error),
	extractResourceID func(r *http.Request) string,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUserFromContext(r.Context())
			if user == nil {
				WriteStructuredError(w, r, ErrCodeUnauthorized, "authentication required", http.StatusUnauthorized)
				return
			}

			resourceID := extractResourceID(r)
			if resourceID == "" {
				WriteStructuredError(w, r, ErrCodeBadRequest, "missing book identifier", http.StatusBadRequest)
				return
			}

			book, err := getBook(resourceID)
			if err != nil {
				WriteStructuredError(w, r, ErrCodeNotFound, "book not found", http.StatusNotFound)
				return
			}

			// Check ownership: same account OR admin role
			isOwner := book.AccountID == "" || book.AccountID == user.AccountID
			if !isOwner && book.UserID != "" && book.UserID == user.ID {
				isOwner = true
			}

			// Admin override
			isAdmin := false
			if !isOwner {
				role, roleErr := pool.Roles.GetRoleByID(r.Context(), user.RoleID)
				if roleErr == nil && role != nil && role.Name == "admin" {
					isAdmin = true
				}
			}

			if isOwner || isAdmin {
				next.ServeHTTP(w, r)
				return
			}

			WriteStructuredError(w, r, ErrCodeForbidden, "access denied", http.StatusForbidden)
		})
	}
}

// WithBookOwnership sets the book owner ID in context for downstream handlers.
type bookOwnerKey struct{}

func WithBookOwner(ctx context.Context, accountID, userID string) context.Context {
	ctx = context.WithValue(ctx, bookOwnerKey{}, bookOwner{accountID, userID})
	return ctx
}

type bookOwner struct {
	AccountID string
	UserID    string
}

func GetBookOwnerFromContext(ctx context.Context) (accountID, userID string, ok bool) {
	v, exists := ctx.Value(bookOwnerKey{}).(bookOwner)
	if !exists {
		return "", "", false
	}
	return v.AccountID, v.UserID, true
}
