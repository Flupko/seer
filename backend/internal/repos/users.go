package repos

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"seer/internal/numeric"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type AuthProvider string

const (
	CredentialsProvider AuthProvider = "credentials"
	GoogleProvider      AuthProvider = "google"
	TwitchProvider      AuthProvider = "twitch"
)

type Role string

const (
	AdminRole Role = "admin"
	UserRole  Role = "user"
)

type UserStatus string

const (
	PendingEmailVerification UserStatus = "pending_email_verification"
	PendingProfile           UserStatus = "pending_profile_completion"
	Activated                UserStatus = "activated"
)

type User struct {
	ID       uuid.UUID
	Email    string
	Username sql.NullString

	PasswordHash   []byte
	ProviderID     AuthProvider
	ProviderUserID string

	Role            Role
	ProfileImageKey sql.NullString
	TotalWagered    numeric.BigDecimal
	Hidden          bool

	CreatedAt time.Time

	Status UserStatus

	Version int64
}

type UserPreferences struct {
	Hidden                 bool
	ReceiveMarketingEmails bool
}

type UserView struct {
	User
}

type MinimalUser struct {
	SessionID       uuid.UUID
	ID              uuid.UUID
	Username        string
	Role            Role
	ProfileImageKey sql.NullString
	Status          UserStatus
	MutedUntil      sql.NullTime
	Version         int64
}

var AnonymousUser = &MinimalUser{}

type UserRepo struct {
	db *pgxpool.Pool
}

var (
	ErrUniqueViolation = errors.New("unique violation")
)

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{
		db: db,
	}
}

func GetHashedPassword(plainText string) ([]byte, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plainText), 12)
	if err != nil {
		return nil, err
	}
	return hash, nil
}

func MatchPassword(hashedPassword []byte, plainText string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(hashedPassword, []byte(plainText))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *UserRepo) UsernameTaken(ctx context.Context, username string) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM users WHERE username=$1)`

	var ok bool
	if err := r.db.QueryRow(ctx, q, username).Scan(&ok); err != nil {
		return false, err
	}

	return ok, nil
}

func (r *UserRepo) EmailTaken(ctx context.Context, email string) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM users WHERE email=$1)`

	var ok bool
	if err := r.db.QueryRow(ctx, q, email).Scan(&ok); err != nil {
		return false, err
	}

	return ok, nil
}

func (r *UserRepo) Insert(ctx context.Context, user *User) error {

	// Insert user and ledger account
	tx, err := r.db.Begin(ctx)

	if err != nil {
		return fmt.Errorf("failed to begin tx")
	}

	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	query := `INSERT INTO users(email, username, profile_image_key, password_hash, provider_id, provider_user_id, status)
    VALUES ($1, $2, $3, $4, $5, $6, $7)
    RETURNING id
	`

	err = tx.QueryRow(ctx, query,
		user.Email,
		user.Username,
		user.ProfileImageKey,
		user.PasswordHash,
		user.ProviderID,
		user.ProviderUserID,
		user.Status,
	).Scan(&user.ID)

	if err != nil {
		var pgErr *pgconn.PgError
		switch {
		case errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation:
			return ErrUniqueViolation
		default:
			return fmt.Errorf("failed to insert user")
		}
	}

	query = `
		INSERT INTO ledger_accounts (user_id, account_type, currency, allow_negative_balance, allow_positive_balance) VALUES 
		($1, 'custody', 'USDT', true, true),
    	($1, 'liability', 'USDT', false, true);`

	_, err = tx.Exec(ctx, query, user.ID)
	if err != nil {
		return fmt.Errorf("failed to insert ledger account")
	}

	return tx.Commit(ctx)
}

func (r *UserRepo) GetBySubProvider(ctx context.Context, sub string, provider AuthProvider) (*User, error) {

	query := `SELECT u.id, u.status
	FROM users u
	WHERE u.provider_id = $1 AND u.provider_user_id = $2`

	row := r.db.QueryRow(ctx, query, provider, sub)
	var user User

	err := row.Scan(
		&user.ID,
		&user.Status,
	)

	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}

func (r *UserRepo) GetByEmailOrUsername(ctx context.Context, login string) (*User, error) {

	query := `SELECT u.id, u.status, u.password_hash
	FROM users u
	WHERE u.email = $1 OR u.username = $1`

	row := r.db.QueryRow(ctx, query, login)
	var user User
	err := row.Scan(
		&user.ID,
		&user.Status,
		&user.PasswordHash,
	)

	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}

func (r *UserRepo) GetByID(ctx context.Context, userID uuid.UUID) (*User, error) {

	query := `SELECT u.id, u.email, u.username, 
	u.password_hash, u.provider_id,
	u.role, u.profile_image_key, u.total_wagered, u.hidden,
	u.created_at, 
	u.status, u.version
	FROM users u
	WHERE u.id = $1`

	row := r.db.QueryRow(ctx, query, userID)
	var user User
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.ProviderID,
		&user.Role,
		&user.ProfileImageKey,
		&user.TotalWagered,
		&user.Hidden,
		&user.CreatedAt,
		&user.Status,
		&user.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}

func (r *UserRepo) GetViewByIDOrUsername(ctx context.Context, userID uuid.UUID, username string) (*UserView, error) {

	query := `SELECT u.id, u.email, u.username, u.password_hash, u.provider_id, u.role, 
	u.profile_image_key, u.total_wagered, u.hidden,
	u.created_at, 
	u.status, u.version
	FROM users u
	WHERE u.id = $1 OR u.username = $2`

	row := r.db.QueryRow(ctx, query, userID, username)
	var user UserView
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.ProviderID,
		&user.Role,
		&user.ProfileImageKey,
		&user.TotalWagered,
		&user.Hidden,
		&user.CreatedAt,
		&user.Status,
		&user.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil

}

func (r *UserRepo) ChangePassword(ctx context.Context, user *User) error {

	tx, err := r.db.Begin(ctx)

	if err != nil {
		return fmt.Errorf("failed to begin tx")
	}

	defer func() { _ = tx.Rollback(ctx) }()

	query := `
        UPDATE users
        SET password_hash = $1, version = version + 1
        WHERE id = $2 AND version = $3
        RETURNING version
    `

	err = tx.QueryRow(ctx, query, user.PasswordHash, user.ID, user.Version).Scan(&user.Version)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	query = `UPDATE sessions 
    SET revoked_at = NOW() 
    WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()`

	_, err = tx.Exec(ctx, query, user.ID)
	if err != nil {
		return fmt.Errorf("failed to revoke sessions: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *UserRepo) CompleteProfile(ctx context.Context, userID uuid.UUID, username string, version int64) error {

	query := `
        UPDATE users 
        SET username = $1, status = 'activated', version = version + 1
        WHERE id = $2 AND version = $3
    `

	cmd, err := r.db.Exec(ctx, query, username, userID, version)

	if err != nil {
		var pgErr *pgconn.PgError
		switch {
		case errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation:
			return ErrUniqueViolation
		default:
			return fmt.Errorf("failed to update user: %w", err)
		}
	}

	if cmd.RowsAffected() == 0 {
		return ErrEditConflict
	}

	return err
}

func (r *UserRepo) UpdateProfileImageKey(ctx context.Context, userID uuid.UUID, newKey string) (string, error) {

	var oldKey sql.NullString

	query := `
        WITH old_key AS (SELECT profile_image_key FROM users WHERE id = $2)
        UPDATE users SET profile_image_key = $1
        WHERE id = $2
        RETURNING (SELECT * FROM old_key)
    `

	err := r.db.QueryRow(ctx, query, newKey, userID).Scan(&oldKey)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return "", ErrRecordNotFound
		default:
			return "", err
		}
	}

	if oldKey.Valid {
		return oldKey.String, nil
	}

	return "", nil

}

func (r *UserRepo) GetPreferences(ctx context.Context, userID uuid.UUID) (*UserPreferences, error) {

	query := `
		SELECT hidden, receive_marketing_emails
		FROM users
		WHERE id = $1
	`

	var prefs UserPreferences
	err := r.db.QueryRow(ctx, query, userID).Scan(&prefs.Hidden, &prefs.ReceiveMarketingEmails)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &prefs, nil
}

func (r *UserRepo) UpdatePreferences(ctx context.Context, userID uuid.UUID, prefs *UserPreferences) error {

	query := `
		UPDATE users
		SET hidden = $1, receive_marketing_emails = $2
		WHERE id = $3
	`

	cmd, err := r.db.Exec(ctx, query, prefs.Hidden, prefs.ReceiveMarketingEmails, userID)
	if err != nil {
		return fmt.Errorf("failed to update user preferences: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return ErrRecordNotFound
	}

	return nil
}
