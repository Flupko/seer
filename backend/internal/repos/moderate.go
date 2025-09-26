package repos

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ModerateRepo struct {
	db *pgxpool.Pool
}

func NewModerateRepo(db *pgxpool.Pool) *ModerateRepo {
	return &ModerateRepo{
		db: db,
	}
}

type UserMute struct {
	ID             int64
	UserID         uuid.UUID
	Reason         string
	EffectiveUntil time.Time
	CreatedAt      time.Time
}

type MuteChatMessage struct {
	ID            int64
	ChatMessageID uuid.UUID
	UserMuteID    int64
}

type MuteComment struct {
	ID         int64
	CommentID  int64
	UserMuteID int64
}

type MuteChatMessageView struct {
	ID        uuid.UUID
	Content   string
	CreatedAt time.Time
}

type MuteCommentView struct {
	ID        int64
	MarketID  uuid.UUID
	Content   string
	CreatedAt time.Time
}

type UserMuteView struct {
	UserMute
	ChatMessages []*MuteChatMessageView
	Comments     []*MuteCommentView
}

func (r *ModerateRepo) MuteUser(ctx context.Context, um *UserMute, chatMessagesIDs []uuid.UUID, commentsIDs []int64) error {
	tx, err := r.db.Begin(ctx)

	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer tx.Rollback(ctx)

	query := `INSERT INTO user_mutes(user_id, reason, effective_until) VALUES($1, $2, $3) RETURNING id, created_at`

	err = tx.QueryRow(ctx, query, um.UserID, um.Reason, um.EffectiveUntil).Scan(&um.ID, &um.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		switch {
		case errors.As(err, &pgErr) && pgErr.Code == pgerrcode.InvalidForeignKey:
			return fmt.Errorf("user not found: %w", ErrRecordNotFound)
		default:
			return fmt.Errorf("failed to insert user_mute: %w", err)
		}
	}

	// Delete chat messages
	if len(chatMessagesIDs) > 0 {
		query = `UPDATE chat_messages SET is_deleted = TRUE, deleted_at = NOW() WHERE id = ANY($1) AND user_id = $2`
		cmd, err := tx.Exec(ctx, query, chatMessagesIDs, um.UserID)

		if err != nil {
			return fmt.Errorf("failed to set chat messages to deleted: %w", err)
		}

		if int(cmd.RowsAffected()) != len(chatMessagesIDs) {
			return fmt.Errorf("chats messages not found for user %s", um.UserID)
		}
	}

	// Delete comments
	if len(commentsIDs) > 0 {
		query = `UPDATE comments SET is_deleted = TRUE, deleted_at = NOW() WHERE id = ANY($1) AND user_id = $2`
		cmd, err := tx.Exec(ctx, query, commentsIDs, um.UserID)

		if err != nil {
			return fmt.Errorf("failed to set comments to deleted: %w", err)
		}

		if int(cmd.RowsAffected()) != len(commentsIDs) {
			return fmt.Errorf("comments not found for user %s", um.UserID)
		}
	}

	// Insert mute records for chat messages
	query = `INSERT INTO mute_chat_messages(chat_message_id, user_mute_id) VALUES ($1, $2)`
	for _, id := range chatMessagesIDs {
		_, err = tx.Exec(ctx, query, id, um.ID)
		if err != nil {
			var pgErr *pgconn.PgError
			switch {
			case errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation:
				return fmt.Errorf("chat message already included in a mute: %w", ErrUniqueViolation)
			default:
				return fmt.Errorf("failed to insert mute chat message: %w", err)
			}
		}
	}

	// Insert mute records for comments
	query = `INSERT INTO mute_comments(comment_id, user_mute_id) VALUES ($1, $2)`
	for _, id := range commentsIDs {
		_, err = tx.Exec(ctx, query, id, um.ID)
		if err != nil {
			var pgErr *pgconn.PgError
			switch {
			case errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation:
				return fmt.Errorf("comment already included in a mute: %w", ErrUniqueViolation)
			default:
				return fmt.Errorf("failed to insert mute comment: %w", err)
			}
		}
	}

	return tx.Commit(ctx)
}

func (r *ModerateRepo) GetUserMuteView(ctx context.Context, muteID int64) (*UserMuteView, error) {

	query := `SELECT um.id, um.user_id, um.reason, um.effective_until, um.created_at
	FROM user_mutes um
	WHERE um.id = $1
	`

	mute := &UserMuteView{}

	err := r.db.QueryRow(ctx, query, muteID).Scan(&mute.ID, &mute.UserID, &mute.Reason, &mute.EffectiveUntil, &mute.CreatedAt)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, fmt.Errorf("failed to query user_mute: %w", err)
		}
	}

	// Retrieving comments
	query = `SELECT c.id, c.market_id, c.content, c.created_at 
	FROM mute_comments mc 
	JOIN comments c ON c.id = mc.comment_id
	WHERE mc.user_mute_id = $1
	`

	rows, err := r.db.Query(ctx, query, mute.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to query comment mutes: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		c := &MuteCommentView{}
		if err = rows.Scan(&c.ID, &c.MarketID, &c.Content, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan comment view: %w", err)
		}
		mute.Comments = append(mute.Comments, c)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating mute comments: %w", rows.Err())
	}

	rows.Close()

	query = `SELECT cm.id, cm.content, cm.created_at
	FROM mute_chat_messages mcm
	JOIN chat_messages cm ON cm.id = mcm.chat_message_id
	WHERE mcm.user_mute_id = $1`

	rows, err = r.db.Query(ctx, query, mute.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to query comment mutes: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		m := &MuteChatMessageView{}
		if err = rows.Scan(&m.ID, &m.Content, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan chat message view: %w", err)
		}
		mute.ChatMessages = append(mute.ChatMessages, m)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating mute chat messages: %w", rows.Err())
	}

	return mute, nil

}
