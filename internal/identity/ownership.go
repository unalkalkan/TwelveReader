package identity

import (
	"context"
	"log"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

// BootstrapAccount returns the bootstrap/default account from the database.
// Returns nil if it does not yet exist.
func (p *DBPool) BootstrapAccount(ctx context.Context) (*types.Account, error) {
	return p.Accounts.GetAccountBySlug(ctx, "bootstrap")
}

// GetBootstrapAdminUser returns the first active admin user belonging to the
// bootstrap account. Returns nil if none found.
func (p *DBPool) GetBootstrapAdminUser(ctx context.Context) (*types.User, error) {
	acc, err := p.BootstrapAccount(ctx)
	if err != nil || acc == nil {
		return nil, nil
	}

	adminRole, err := p.Roles.GetRoleByName(ctx, "admin")
	if err != nil || adminRole == nil {
		return nil, nil
	}

	rows, err := p.db.QueryContext(ctx,
		"SELECT id, account_id, email, name, role_id, status, created_at, updated_at, deleted_at "+
			"FROM users WHERE account_id = ? AND role_id = ? AND status = 'active' AND deleted_at IS NULL",
		acc.ID, adminRole.ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users, _ := scanUsers(rows)
	if len(users) == 0 {
		return nil, nil
	}
	return users[0], nil
}

// MigrateLocalBooksToBootstrap migrates all books that lack ownership to the
// given bootstrap account and user. Returns the number of books migrated.
// This is called once at startup after bootstrap data is ensured.
func (s *AuthService) MigrateLocalBooksToBootstrap(
	ctx context.Context,
	bootstrapAccountID string,
	bootstrapUserID    string,
	listBooks  func(ctx context.Context) ([]*types.Book, error),
	updateBook func(ctx context.Context, book *types.Book) error,
) (int, error) {
	if bootstrapAccountID == "" || bootstrapUserID == "" {
		return 0, nil
	}

	books, err := listBooks(ctx)
	if err != nil {
		return 0, err
	}

	migrated := 0
	for _, book := range books {
		if book.AccountID == "" && book.UserID == "" {
			book.AccountID = bootstrapAccountID
			book.UserID = bootstrapUserID
			if err := updateBook(ctx, book); err != nil {
				log.Printf("[IDENTITY] Failed to update ownership for book %s: %v", book.ID, err)
				continue
			}
			migrated++

			s.writeAudit(ctx, bootstrapUserID, bootstrapAccountID, types.AuditEventOwnership,
				"migrate_local_book_to_bootstrap", map[string]string{
					"book_id":     book.ID,
					"book_title":  book.Title,
					"account_id":  bootstrapAccountID,
					"user_id":     bootstrapUserID,
					"migration":   "startup",
				})
		}
	}

	if migrated > 0 {
		log.Printf("[IDENTITY] Migrated %d local book(s) to bootstrap account %s / user %s",
			migrated, bootstrapAccountID, bootstrapUserID)
	}
	return migrated, nil
}
