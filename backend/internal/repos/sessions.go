package repos

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepo struct {
	db *pgxpool.Pool
}

type Session struct {
	ID     uuid.UUID
	UserID uuid.UUID
	Hash   []byte

	CreatedAt  time.Time
	LastUsedAt time.Time
	ExpiresAt  time.Time
	RevokedAt  sql.NullTime

	UserAgent     sql.NullString
	ClientOS      sql.NullString
	ClientBrowser sql.NullString
	ClientDevice  sql.NullString

	IPFirst sql.NullString
	IPLast  sql.NullString

	GeoCountry sql.NullString
	GeoCity    sql.NullString
}

func NewSessionRepo(db *pgxpool.Pool) *SessionRepo {
	return &SessionRepo{
		db: db,
	}
}

// GenerateSession creates a new session and returns the plaintext token and the session object
func GenerateSession(UserID uuid.UUID, duration time.Duration, ip string, userAgent string, clientOS string, clientBrowser string, clientDevice string, geoCountry string, geoCity string) (string, *Session, error) {
	// 32 bytes of enthropy
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", &Session{}, err
	}

	sessionPlainText := base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(randomBytes)
	sessionHash := sha256.Sum256([]byte(sessionPlainText))

	session := &Session{
		UserID:        UserID,
		Hash:          sessionHash[:],
		CreatedAt:     time.Now(),
		LastUsedAt:    time.Now(),
		ExpiresAt:     time.Now().Add(duration),
		IPFirst:       sql.NullString{String: ip, Valid: ip != ""},
		IPLast:        sql.NullString{String: ip, Valid: ip != ""},
		UserAgent:     sql.NullString{String: userAgent, Valid: userAgent != ""},
		ClientOS:      sql.NullString{String: clientOS, Valid: clientOS != ""},
		ClientBrowser: sql.NullString{String: clientBrowser, Valid: clientBrowser != ""},
		ClientDevice:  sql.NullString{String: clientDevice, Valid: clientDevice != ""},
		GeoCountry:    sql.NullString{String: geoCountry, Valid: geoCountry != ""},
		GeoCity:       sql.NullString{String: geoCity, Valid: geoCity != ""},
	}

	return sessionPlainText, session, nil

}

func (r *SessionRepo) CreateSession(ctx context.Context, s *Session) error {

	query := `INSERT INTO sessions(user_id, hash, 
	created_at, last_used_at, expires_at, 
	ip_first, ip_last, 
	user_agent, client_os, client_browser, client_device,
	geo_country, geo_city) 
	VALUES ($1, $2, $3,
	$4, $5, $6,
	$7, $8,
	$9, $10, $11,
	$12, $13)
	RETURNING id`

	err := r.db.QueryRow(ctx, query,
		s.UserID,
		s.Hash,
		s.CreatedAt,
		s.LastUsedAt,
		s.ExpiresAt,
		s.IPFirst,
		s.IPLast,
		s.UserAgent,
		s.ClientOS,
		s.ClientBrowser,
		s.ClientDevice,
		s.GeoCountry,
		s.GeoCity,
	).Scan(&s.ID)

	return err
}

func (r *SessionRepo) GetUserFromPlain(ctx context.Context, plain string, ip string) (*MinimalUser, error) {

	sum := sha256.Sum256([]byte(plain))
	hash := sum[:]

	query := `
        UPDATE sessions AS s
        SET last_used_at = NOW(), ip_last = $2
        FROM users AS u
        WHERE s.hash = $1 AND s.expires_at > NOW() AND s.revoked_at IS NULL AND u.id = s.user_id
        RETURNING u.id, u.role, u.status;
    `

	user := &MinimalUser{}

	err := r.db.QueryRow(ctx, query, hash, ip).Scan(
		&user.ID,
		&user.Role,
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

	return user, nil
}

func (r *SessionRepo) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	query := `
		UPDATE sessions
		SET revoked_at = $1
		WHERE id = $2 AND revoked_at IS NULL
	`

	cmd, err := r.db.Exec(ctx, query, time.Now(), sessionID)
	if err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return ErrRecordNotFound
	}

	return nil
}

func (r *SessionRepo) GetAllActiveForUser(ctx context.Context, userID uuid.UUID) ([]*Session, error) {
	query := `
		SELECT id, user_id, hash, 
		created_at, last_seen_at, expires_at, revoked_at, 
		ip_first, ip_last, 
		user_agent, client_os, client_browser, client_device, 
		geo_country, geo_city
		FROM sessions
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
		ORDER BY last_seen_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		var s Session
		err := rows.Scan(
			&s.ID,
			&s.UserID,
			&s.Hash,
			&s.CreatedAt,
			&s.LastUsedAt,
			&s.ExpiresAt,
			&s.RevokedAt,
			&s.IPFirst,
			&s.IPLast,
			&s.UserAgent,
			&s.ClientOS,
			&s.ClientBrowser,
			&s.ClientDevice,
			&s.GeoCountry,
			&s.GeoCity,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, &s)
	}

	return sessions, nil
}

func (r *SessionRepo) CleanupExpired(ctx context.Context) error {
	query := `
		DELETE FROM sessions
		WHERE (expires_at <= NOW()) OR (revoked_at IS NOT NULL)
	`

	_, err := r.db.Exec(ctx, query)
	return err
}

func (r *SessionRepo) LimitSessionsUser(ctx context.Context, userID uuid.UUID, maxSessions int) error {
	// Keep only most recent maxSessions sessions
	query := `
		DELETE FROM sessions
		WHERE user_id = $1 
		AND id NOT IN (
            SELECT id FROM sessions 
            WHERE user_id = $1
            ORDER BY created_at DESC 
            LIMIT $2
		)`

	_, err := r.db.Exec(ctx, query, userID, maxSessions)

	if err != nil {
		return fmt.Errorf("failed to delete oldest sessions: %w", err)
	}

	return nil

}
