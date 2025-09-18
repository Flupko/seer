package repos

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TokenScope string

const (
	ScopeEmaiVerification  TokenScope = "email_verification"
	ScopeProfileCompletion TokenScope = "profile_completion"
	ScopePasswordReset     TokenScope = "password_reset"
	ScopeAuthentication    TokenScope = "authentication"
)

type TokenRepo struct {
	db *pgxpool.Pool
}

type Token struct {
	Hash      []byte
	UserID    uuid.UUID
	Scope     TokenScope
	CreatedAt time.Time
	Expiry    time.Time
}

func NewTokenRepo(db *pgxpool.Pool) *TokenRepo {
	return &TokenRepo{
		db: db,
	}
}

// GenerateToken returns a plaintext and hashed token, both in base64
func GenerateToken(userID uuid.UUID, scope TokenScope, duration time.Duration) (string, *Token, error) {

	// 32 bytes of enthropy
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", &Token{}, err
	}

	tokenPlainText := base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(randomBytes)
	tokenHash := sha256.Sum256([]byte(tokenPlainText))

	token := &Token{
		UserID:    userID,
		Scope:     scope,
		Hash:      tokenHash[:],
		CreatedAt: time.Now().UTC(),
	}

	token.Expiry = token.CreatedAt.Add(duration)

	return tokenPlainText, token, nil

}

func (r *TokenRepo) Insert(ctx context.Context, token *Token) error {
	query := `
		INSERT INTO tokens (user_id, hash, scope, created_at, expiry)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.Exec(
		ctx,
		query,
		token.UserID,
		token.Hash,
		token.Scope,
		token.CreatedAt,
		token.Expiry,
	)

	return err
}

func (r *TokenRepo) GetUserForToken(ctx context.Context, scope TokenScope, tokenPlainText string) (*MinimalUser, error) {

	tokenHash := sha256.Sum256([]byte(tokenPlainText))

	query := `SELECT u.id, u.role, u.activated, u.version
	FROM users u
	JOIN tokens t ON u.id = t.user_id
	WHERE t.hash = $1 AND t.scope = $2 AND t.expiry > $3`

	var user MinimalUser

	err := r.db.QueryRow(ctx, query, tokenHash[:], scope, time.Now().UTC()).Scan(
		&user.ID,
		&user.Role,
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

func (r TokenRepo) DeleteAllForUser(ctx context.Context, scope TokenScope, userID uuid.UUID) error {

	query := `
		DELETE FROM tokens
		WHERE scope = $1 AND user_id = $2`

	_, err := r.db.Exec(ctx, query, scope, userID)

	return err
}
