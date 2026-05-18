package identity

import (
	"context"
	"testing"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

func TestMigrateLocalBooksToBootstrap(t *testing.T) {
	authService, pool := newTestAuthService(t)
	defer pool.Close()
	ctx := context.Background()

	// Ensure bootstrap account and admin exist
	adminUser, err := authService.EnsureBootstrapAdmin(ctx, "admin@test.local")
	if err != nil {
		t.Fatalf("EnsureBootstrapAdmin: %v", err)
	}

	bootstrapAccount, err := pool.BootstrapAccount(ctx)
	if err != nil || bootstrapAccount == nil {
		t.Fatal("BootstrapAccount should exist after EnsureBootstrapAdmin")
	}

	// Simulate existing books (localStorage mock in-memory)
	existingBooks := map[string]*types.Book{
		"book_1": {ID: "book_1", Title: "Old Book 1", AccountID: "", UserID: ""},
		"book_2": {ID: "book_2", Title: "Old Book 2", AccountID: "", UserID: ""},
		"book_3": {ID: "book_3", Title: "Already Owned", AccountID: bootstrapAccount.ID, UserID: adminUser.ID},
	}

	migrated, err := authService.MigrateLocalBooksToBootstrap(
		ctx,
		bootstrapAccount.ID,
		adminUser.ID,
		func(ctx context.Context) ([]*types.Book, error) {
			var result []*types.Book
			for _, b := range existingBooks {
				result = append(result, b)
			}
			return result, nil
		},
		func(ctx context.Context, book *types.Book) error {
			existingBooks[book.ID] = book
			return nil
		},
	)
	if err != nil {
		t.Fatalf("MigrateLocalBooksToBootstrap: %v", err)
	}

	// Only 2 books should be migrated (book_3 already has ownership)
	if migrated != 2 {
		t.Errorf("expected 2 migrated, got %d", migrated)
	}

	// Verify ownership was set correctly for previously unowned books
	for id, book := range existingBooks {
		if book.AccountID == "" || book.UserID == "" {
			t.Errorf("book %s missing ownership: account=%q, user=%q", id, book.AccountID, book.UserID)
		}
	}
}

func TestBootstrapAccountAndUser(t *testing.T) {
	authService, pool := newTestAuthService(t)
	defer pool.Close()
	ctx := context.Background()

	// Bootstrap account should not exist yet
	account, err := pool.BootstrapAccount(ctx)
	if err == nil || account != nil {
		t.Error("BootstrapAccount should return nil before creation")
	}

	adminUser, err := authService.EnsureBootstrapAdmin(ctx, "admin@test.local")
	if err != nil {
		t.Fatalf("EnsureBootstrapAdmin: %v", err)
	}

	// Now bootstrap account should exist (same pool as authService)
	account, err = pool.BootstrapAccount(ctx)
	if err != nil || account == nil {
		t.Fatal("BootstrapAccount should exist after EnsureBootstrapAdmin")
	}

	// Bootstrap admin user should be retrievable
	adminFromPool, err := pool.GetBootstrapAdminUser(ctx)
	if err != nil || adminFromPool == nil {
		t.Fatal("GetBootstrapAdminUser should return the admin user")
	}
	if adminFromPool.ID != adminUser.ID {
		t.Errorf("admin ID mismatch: got %s, want %s", adminFromPool.ID, adminUser.ID)
	}
}

func TestMigrateNoBooks(t *testing.T) {
	authService, pool := newTestAuthService(t)
	defer pool.Close()
	ctx := context.Background()

	adminUser, err := authService.EnsureBootstrapAdmin(ctx, "admin@test.local")
	if err != nil {
		t.Fatalf("EnsureBootstrapAdmin: %v", err)
	}
	bootstrapAccount, _ := pool.BootstrapAccount(ctx)

	updateCalled := false
	migrated, err := authService.MigrateLocalBooksToBootstrap(
		ctx,
		bootstrapAccount.ID,
		adminUser.ID,
		func(ctx context.Context) ([]*types.Book, error) {
			return []*types.Book{}, nil
		},
		func(ctx context.Context, book *types.Book) error {
			updateCalled = true
			return nil
		},
	)
	if err != nil {
		t.Fatalf("MigrateLocalBooksToBootstrap: %v", err)
	}
	if migrated != 0 {
		t.Errorf("expected 0 migrated, got %d", migrated)
	}
	if updateCalled {
		t.Error("updateBook should not be called when no books exist")
	}
}

func TestMigrateEmptyBootstrapIDs(t *testing.T) {
	authService, pool := newTestAuthService(t)
	defer pool.Close()
	ctx := context.Background()

	updateCalled := false
	migrated, err := authService.MigrateLocalBooksToBootstrap(
		ctx,
		"", // empty account ID
		"", // empty user ID
		func(ctx context.Context) ([]*types.Book, error) {
			return []*types.Book{{ID: "book_1", Title: "Test"}}, nil
		},
		func(ctx context.Context, book *types.Book) error {
			updateCalled = true
			return nil
		},
	)
	if err != nil {
		t.Fatalf("MigrateLocalBooksToBootstrap: %v", err)
	}
	if migrated != 0 {
		t.Errorf("expected 0 migrated with empty IDs, got %d", migrated)
	}
	if updateCalled {
		t.Error("updateBook should not be called with empty bootstrap IDs")
	}
}

func TestMigratePartialOwnership(t *testing.T) {
	authService, pool := newTestAuthService(t)
	defer pool.Close()
	ctx := context.Background()

	adminUser, err := authService.EnsureBootstrapAdmin(ctx, "admin@test.local")
	if err != nil {
		t.Fatalf("EnsureBootstrapAdmin: %v", err)
	}
	bootstrapAccount, _ := pool.BootstrapAccount(ctx)

	existingBooks := map[string]*types.Book{
		"book_1":     {ID: "book_1", Title: "No ownership", AccountID: "", UserID: ""},
		"book_2":     {ID: "book_2", Title: "Has account", AccountID: bootstrapAccount.ID, UserID: ""},
		"book_3":     {ID: "book_3", Title: "Has user", AccountID: "", UserID: adminUser.ID},
		"book_full":  {ID: "book_full", Title: "Full ownership", AccountID: bootstrapAccount.ID, UserID: adminUser.ID},
	}

	migrated, err := authService.MigrateLocalBooksToBootstrap(
		ctx,
		bootstrapAccount.ID,
		adminUser.ID,
		func(ctx context.Context) ([]*types.Book, error) {
			var result []*types.Book
			for _, b := range existingBooks {
				result = append(result, b)
			}
			return result, nil
		},
		func(ctx context.Context, book *types.Book) error {
			existingBooks[book.ID] = book
			return nil
		},
	)
	if err != nil {
		t.Fatalf("MigrateLocalBooksToBootstrap: %v", err)
	}

	// Only book_1 has both fields empty; books with partial ownership are skipped
	if migrated != 1 {
		t.Errorf("expected 1 migrated (only fully unowned), got %d", migrated)
	}
}
