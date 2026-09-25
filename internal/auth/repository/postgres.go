package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/dogfood-platform/dogfood/internal/auth/domain"
	"github.com/dogfood-platform/dogfood/internal/shared/response"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"time"
)

type userRow struct {
	ID           uuid.UUID `db:"id"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
	DisplayName  string    `db:"display_name"`
	AvatarURL    *string   `db:"avatar_url"`
	IsActive     bool      `db:"is_active"`
	IsAdmin      bool      `db:"is_admin"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

func toUserRow(u *domain.User) userRow {
	return userRow{
		ID:           u.ID,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		DisplayName:  u.DisplayName,
		AvatarURL:    u.AvatarURL,
		IsActive:     u.IsActive,
		IsAdmin:      u.IsAdmin,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

func toDomainUser(r userRow) *domain.User {
	return &domain.User{
		ID:           r.ID,
		Email:        r.Email,
		PasswordHash: r.PasswordHash,
		DisplayName:  r.DisplayName,
		AvatarURL:    r.AvatarURL,
		IsActive:     r.IsActive,
		IsAdmin:      r.IsAdmin,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

type refreshTokenRow struct {
	ID        uuid.UUID  `db:"id"`
	UserID    uuid.UUID  `db:"user_id"`
	TokenHash string     `db:"token_hash"`
	ExpiresAt time.Time  `db:"expires_at"`
	RevokedAt *time.Time `db:"revoked_at"`
	IssuedAt  time.Time  `db:"issued_at"`
	IPHash    *string    `db:"ip_hash"`
}

func toRefreshTokenRow(t *domain.RefreshToken) refreshTokenRow {
	return refreshTokenRow{
		ID:        t.ID,
		UserID:    t.UserID,
		TokenHash: t.Hash,
		ExpiresAt: t.ExpiresAt,
		RevokedAt: t.RevokedAt,
		IssuedAt:  t.CreatedAt,
		IPHash:    nil,
	}
}

func toDomainRefreshToken(r refreshTokenRow) *domain.RefreshToken {
	return &domain.RefreshToken{
		ID:        r.ID,
		UserID:    r.UserID,
		Hash:      r.TokenHash,
		ExpiresAt: r.ExpiresAt,
		RevokedAt: r.RevokedAt,
		CreatedAt: r.IssuedAt,
	}
}

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Save(ctx context.Context, user *domain.User) error {
	const q = `
        INSERT INTO users (id, email, password_hash, display_name, avatar_url, is_active, is_admin, created_at, updated_at)
        VALUES (:id, :email, :password_hash, :display_name, :avatar_url, :is_active, :is_admin, :created_at, :updated_at)`
	if _, err := r.db.NamedExecContext(ctx, q, toUserRow(user)); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%w: email already registered", response.ErrDuplicate)
		}
		return fmt.Errorf("user repo save: %w", err)
	}
	return nil
}

func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count, "SELECT COUNT(1) FROM users WHERE email = $1", email)
	return count > 0, err
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var row userRow
	err := r.db.GetContext(ctx, &row, "SELECT id, email, password_hash, display_name, avatar_url, is_active, is_admin, created_at, updated_at FROM users WHERE email = $1", email)
	if err != nil {
		return nil, fmt.Errorf("%w: user not found", response.ErrNotFound)
	}
	return toDomainUser(row), nil
}

func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var row userRow
	err := r.db.GetContext(ctx, &row, "SELECT id, email, password_hash, display_name, avatar_url, is_active, is_admin, created_at, updated_at FROM users WHERE id = $1", id)
	if err != nil {
		return nil, fmt.Errorf("%w: user not found", response.ErrNotFound)
	}
	return toDomainUser(row), nil
}

// RefreshTokenRepository implements port.RefreshTokenRepository
type RefreshTokenRepository struct {
	db *sqlx.DB
}

func NewRefreshTokenRepository(db *sqlx.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Save(ctx context.Context, token *domain.RefreshToken) error {
	const q = `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, revoked_at, issued_at)
		VALUES (:id, :user_id, :token_hash, :expires_at, :revoked_at, :issued_at)`
	_, err := r.db.NamedExecContext(ctx, q, toRefreshTokenRow(token))
	return err
}

func (r *RefreshTokenRepository) FindByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	var row refreshTokenRow
	err := r.db.GetContext(ctx, &row, "SELECT id, user_id, token_hash, expires_at, revoked_at, issued_at, ip_hash FROM refresh_tokens WHERE token_hash = $1", hash)
	if err != nil {
		return nil, fmt.Errorf("%w: refresh token not found", response.ErrNotFound)
	}
	return toDomainRefreshToken(row), nil
}

func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	const q = `UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := r.db.ExecContext(ctx, q, userID)
	return err
}
