package repository_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dogfood-platform/dogfood/internal/auth/domain"
	"github.com/dogfood-platform/dogfood/internal/auth/repository"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func runMigrations(t *testing.T, db *sqlx.DB) {
	// Read migrations relative to project root
	// Since tests run in internal/auth/repository, we go up 3 levels
	projectRoot := filepath.Join("..", "..", "..")

	migrations := []string{
		"migrations/001_create_users.sql",
		"migrations/007_create_refresh_tokens.sql",
	}

	for _, m := range migrations {
		path := filepath.Join(projectRoot, m)
		content, err := os.ReadFile(path)
		require.NoError(t, err, "failed to read migration %s", path)
		_, err = db.Exec(string(content))
		require.NoError(t, err, "failed to execute migration %s", path)
	}
}

func setupTestDB(t *testing.T) (*sqlx.DB, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	provider, err := testcontainers.NewDockerProvider()
	if err != nil {
		t.Skip("Docker is not available, skipping test")
	}
	defer provider.Close()

	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16-alpine"),
		postgres.WithDatabase("dogfood_test"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(10*time.Second)),
	)
	require.NoError(t, err)

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := sqlx.Connect("pgx", connStr)
	require.NoError(t, err)

	runMigrations(t, db)

	cleanup := func() {
		db.Close()
		pgContainer.Terminate(context.Background())
	}

	return db, cleanup
}

func TestUserRepository_SaveAndFindByEmail(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := repository.NewUserRepository(db)
	user, err := domain.NewUser("alice@example.com", "Alice Chen")
	require.NoError(t, err)
	user.PasswordHash = "hash"

	err = repo.Save(context.Background(), user)
	require.NoError(t, err)

	found, err := repo.FindByEmail(context.Background(), "alice@example.com")
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, user.ID, found.ID)
	assert.Equal(t, "alice@example.com", found.Email)
	assert.Equal(t, "Alice Chen", found.DisplayName)

	exists, err := repo.ExistsByEmail(context.Background(), "alice@example.com")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.ExistsByEmail(context.Background(), "bob@example.com")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestRefreshTokenRepository_SaveAndFindByHash(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	userRepo := repository.NewUserRepository(db)
	user, err := domain.NewUser("alice@example.com", "Alice Chen")
	require.NoError(t, err)
	user.PasswordHash = "hash"
	require.NoError(t, userRepo.Save(context.Background(), user))

	tokenRepo := repository.NewRefreshTokenRepository(db)

	token := &domain.RefreshToken{
		UserID:    user.ID,
		Hash:      "hash123",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	err = tokenRepo.Save(context.Background(), token)
	require.NoError(t, err)

	found, err := tokenRepo.FindByHash(context.Background(), "hash123")
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, user.ID, found.UserID)
	assert.Equal(t, "hash123", found.Hash)
}
