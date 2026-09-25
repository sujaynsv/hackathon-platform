package repository

import (
	"context"
	"database/sql"
	"encoding/json"
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
	IsVerified   bool      `db:"is_verified"`
	IsActive     bool      `db:"is_active"`
	IsAdmin      bool      `db:"is_admin"`
	VerifiedAt   *time.Time`db:"verified_at"`
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
		IsVerified:   u.IsVerified,
		IsActive:     u.IsActive,
		IsAdmin:      u.IsAdmin,
		VerifiedAt:   u.VerifiedAt,
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
		IsVerified:   r.IsVerified,
		IsActive:     r.IsActive,
		IsAdmin:      r.IsAdmin,
		VerifiedAt:   r.VerifiedAt,
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
        INSERT INTO users (id, email, password_hash, display_name, avatar_url, is_verified, is_active, is_admin, verified_at, created_at, updated_at)
        VALUES (:id, :email, :password_hash, :display_name, :avatar_url, :is_verified, :is_active, :is_admin, :verified_at, :created_at, :updated_at)`
	if _, err := r.db.NamedExecContext(ctx, q, toUserRow(user)); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%w: email already registered", response.ErrDuplicate)
		}
		return fmt.Errorf("user repo save: %w", err)
	}
	return nil
}

func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	const q = `
		UPDATE users 
		SET email = :email, password_hash = :password_hash, display_name = :display_name, 
		    avatar_url = :avatar_url, is_verified = :is_verified, is_active = :is_active, 
		    is_admin = :is_admin, verified_at = :verified_at, updated_at = :updated_at
		WHERE id = :id`
	_, err := r.db.NamedExecContext(ctx, q, toUserRow(user))
	if err != nil {
		return fmt.Errorf("user repo update: %w", err)
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
	err := r.db.GetContext(ctx, &row, "SELECT id, email, password_hash, display_name, avatar_url, is_verified, is_active, is_admin, verified_at, created_at, updated_at FROM users WHERE email = $1", email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: user not found", response.ErrNotFound)
		}
		return nil, fmt.Errorf("find by email: %w", err)
	}
	return toDomainUser(row), nil
}

func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var row userRow
	err := r.db.GetContext(ctx, &row, "SELECT id, email, password_hash, display_name, avatar_url, is_verified, is_active, is_admin, verified_at, created_at, updated_at FROM users WHERE id = $1", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: user not found", response.ErrNotFound)
		}
		return nil, fmt.Errorf("find by id: %w", err)
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
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: refresh token not found", response.ErrNotFound)
		}
		return nil, fmt.Errorf("find refresh token: %w", err)
	}
	return toDomainRefreshToken(row), nil
}

func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	const q = `UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := r.db.ExecContext(ctx, q, userID)
	return err
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE refresh_tokens SET revoked_at = NOW() WHERE id = $1 AND revoked_at IS NULL`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

type emailTokenRow struct {
	ID        uuid.UUID `db:"id"`
	UserID    uuid.UUID `db:"user_id"`
	TokenHash string    `db:"token_hash"`
	ExpiresAt time.Time `db:"expires_at"`
	IssuedAt  time.Time `db:"issued_at"`
	IsUsed    bool      `db:"is_used"`
}

type EmailVerificationRepository struct {
	db *sqlx.DB
}

func NewEmailVerificationRepository(db *sqlx.DB) *EmailVerificationRepository {
	return &EmailVerificationRepository{db: db}
}

func (r *EmailVerificationRepository) Save(ctx context.Context, token *domain.EmailVerificationToken) error {
	const q = `
		INSERT INTO email_verification_tokens (id, user_id, token_hash, expires_at, issued_at, is_used)
		VALUES (:id, :user_id, :token_hash, :expires_at, :issued_at, :is_used)`
	row := emailTokenRow{
		ID:        token.ID,
		UserID:    token.UserID,
		TokenHash: token.TokenHash,
		ExpiresAt: token.ExpiresAt,
		IssuedAt:  token.IssuedAt,
		IsUsed:    token.IsUsed,
	}
	_, err := r.db.NamedExecContext(ctx, q, row)
	if err != nil {
		return fmt.Errorf("email verification token save: %w", err)
	}
	return nil
}

func (r *EmailVerificationRepository) FindByHash(ctx context.Context, hash string) (*domain.EmailVerificationToken, error) {
	var row emailTokenRow
	err := r.db.GetContext(ctx, &row, "SELECT id, user_id, token_hash, expires_at, issued_at, is_used FROM email_verification_tokens WHERE token_hash = $1", hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: email token not found", response.ErrNotFound)
		}
		return nil, fmt.Errorf("find email token by hash: %w", err)
	}
	return &domain.EmailVerificationToken{
		ID:        row.ID,
		UserID:    row.UserID,
		TokenHash: row.TokenHash,
		ExpiresAt: row.ExpiresAt,
		IssuedAt:  row.IssuedAt,
		IsUsed:    row.IsUsed,
	}, nil
}

func (r *EmailVerificationRepository) MarkUsed(ctx context.Context, tokenID uuid.UUID) error {
	const q = `UPDATE email_verification_tokens SET is_used = true WHERE id = $1 AND is_used = false`
	res, err := r.db.ExecContext(ctx, q, tokenID)
	if err != nil {
		return fmt.Errorf("mark email token used: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("token already used or not found")
	}
	return nil
}

type webAuthnCredentialRow struct {
	ID                  uuid.UUID `db:"id"`
	UserID              uuid.UUID `db:"user_id"`
	CredentialID        []byte    `db:"credential_id"`
	PublicKey           []byte    `db:"public_key"`
	AttestationType     string    `db:"attestation_type"`
	Transport           []byte    `db:"transport"`
	Flags               []byte    `db:"flags"`
	AuthenticatorAAGUID []byte    `db:"authenticator_aaguid"`
	SignCount           uint32    `db:"sign_count"`
	CloneWarning        bool      `db:"clone_warning"`
	CreatedAt           time.Time `db:"created_at"`
	LastUsedAt          *time.Time`db:"last_used_at"`
}

type WebAuthnRepository struct {
	db *sqlx.DB
}

func NewWebAuthnRepository(db *sqlx.DB) *WebAuthnRepository {
	return &WebAuthnRepository{db: db}
}

func (r *WebAuthnRepository) Save(ctx context.Context, cred *domain.WebAuthnCredential) error {
	flagsJSON, err := cred.Flags.ToJSON()
	if err != nil {
		return fmt.Errorf("marshal flags: %w", err)
	}
	// transport json
	transportJSON, err := json.Marshal(cred.Transport)
	if err != nil {
		return fmt.Errorf("marshal transport: %w", err)
	}

	row := webAuthnCredentialRow{
		ID:                  cred.ID,
		UserID:              cred.UserID,
		CredentialID:        cred.CredentialID,
		PublicKey:           cred.PublicKey,
		AttestationType:     cred.AttestationType,
		Transport:           transportJSON,
		Flags:               flagsJSON,
		AuthenticatorAAGUID: cred.AuthenticatorAAGUID,
		SignCount:           cred.SignCount,
		CloneWarning:        cred.CloneWarning,
		CreatedAt:           cred.CreatedAt,
		LastUsedAt:          cred.LastUsedAt,
	}

	const q = `
		INSERT INTO webauthn_credentials (
			id, user_id, credential_id, public_key, attestation_type, transport, flags, authenticator_aaguid, sign_count, clone_warning, created_at, last_used_at
		) VALUES (
			:id, :user_id, :credential_id, :public_key, :attestation_type, :transport, :flags, :authenticator_aaguid, :sign_count, :clone_warning, :created_at, :last_used_at
		)`
	if _, err := r.db.NamedExecContext(ctx, q, row); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%w: passkey already registered", response.ErrDuplicate)
		}
		return fmt.Errorf("webauthn save: %w", err)
	}
	return nil
}

func (r *WebAuthnRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.WebAuthnCredential, error) {
	var rows []webAuthnCredentialRow
	const q = `
		SELECT id, user_id, credential_id, public_key, attestation_type, transport, flags, authenticator_aaguid, sign_count, clone_warning, created_at, last_used_at 
		FROM webauthn_credentials WHERE user_id = $1`
	if err := r.db.SelectContext(ctx, &rows, q, userID); err != nil {
		return nil, fmt.Errorf("find webauthn by user: %w", err)
	}

	var creds []*domain.WebAuthnCredential
	for _, row := range rows {
		var flags domain.CredentialFlags
		_ = flags.ScanJSON(row.Flags)
		var transport []string
		_ = json.Unmarshal(row.Transport, &transport)
		
		creds = append(creds, &domain.WebAuthnCredential{
			ID:                  row.ID,
			UserID:              row.UserID,
			CredentialID:        row.CredentialID,
			PublicKey:           row.PublicKey,
			AttestationType:     row.AttestationType,
			Transport:           transport,
			Flags:               flags,
			AuthenticatorAAGUID: row.AuthenticatorAAGUID,
			SignCount:           row.SignCount,
			CloneWarning:        row.CloneWarning,
			CreatedAt:           row.CreatedAt,
			LastUsedAt:          row.LastUsedAt,
		})
	}
	return creds, nil
}

func (r *WebAuthnRepository) Update(ctx context.Context, cred *domain.WebAuthnCredential) error {
	flagsJSON, _ := cred.Flags.ToJSON()
	transportJSON, _ := json.Marshal(cred.Transport)

	row := webAuthnCredentialRow{
		ID:                  cred.ID,
		UserID:              cred.UserID,
		CredentialID:        cred.CredentialID,
		PublicKey:           cred.PublicKey,
		AttestationType:     cred.AttestationType,
		Transport:           transportJSON,
		Flags:               flagsJSON,
		AuthenticatorAAGUID: cred.AuthenticatorAAGUID,
		SignCount:           cred.SignCount,
		CloneWarning:        cred.CloneWarning,
		CreatedAt:           cred.CreatedAt,
		LastUsedAt:          cred.LastUsedAt,
	}

	const q = `
		UPDATE webauthn_credentials
		SET sign_count = :sign_count, clone_warning = :clone_warning, last_used_at = :last_used_at, flags = :flags
		WHERE id = :id`
	if _, err := r.db.NamedExecContext(ctx, q, row); err != nil {
		return fmt.Errorf("update webauthn: %w", err)
	}
	return nil
}

