package repos

import (
	"context"
	"errors"
	"fmt"
	"seer/internal/utils/meta"
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

type Mute struct {
	ID             int64
	UserID         uuid.UUID
	Reason         string
	EffectiveUntil time.Time
	CreatedAt      time.Time
}

type MuteChatMessage struct {
	ID            int64
	ChatMessageID uuid.UUID
	Mute          int64
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

type MuteView struct {
	Mute
	ChatMessages []*MuteChatMessageView
	Comments     []*MuteCommentView
}

func (r *ModerateRepo) MuteUser(ctx context.Context, m *Mute, chatMessagesIDs []uuid.UUID, commentsIDs []int64) error {
	tx, err := r.db.Begin(ctx)

	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() { _ = tx.Rollback(ctx) }()

	query := `INSERT INTO mutes(user_id, reason, effective_until) VALUES($1, $2, $3) RETURNING id, created_at`

	err = tx.QueryRow(ctx, query, m.UserID, m.Reason, m.EffectiveUntil).Scan(&m.ID, &m.CreatedAt)
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
		cmd, err := tx.Exec(ctx, query, chatMessagesIDs, m.UserID)

		if err != nil {
			return fmt.Errorf("failed to set chat messages to deleted: %w", err)
		}

		if int(cmd.RowsAffected()) != len(chatMessagesIDs) {
			return fmt.Errorf("chats messages not found for user %s", m.UserID)
		}
	}

	// Delete comments
	if len(commentsIDs) > 0 {
		query = `UPDATE comments SET is_deleted = TRUE, deleted_at = NOW() WHERE id = ANY($1) AND user_id = $2`
		cmd, err := tx.Exec(ctx, query, commentsIDs, m.UserID)

		if err != nil {
			return fmt.Errorf("failed to set comments to deleted: %w", err)
		}

		if int(cmd.RowsAffected()) != len(commentsIDs) {
			return fmt.Errorf("comments not found for user %s", m.UserID)
		}
	}

	// Insert mute records for chat messages
	query = `INSERT INTO mute_chat_messages(chat_message_id, mute_id) VALUES ($1, $2)`
	for _, id := range chatMessagesIDs {
		_, err = tx.Exec(ctx, query, id, m.ID)
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
	query = `INSERT INTO mute_comments(comment_id, mute_id) VALUES ($1, $2)`
	for _, id := range commentsIDs {
		_, err = tx.Exec(ctx, query, id, m.ID)
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

func (r *ModerateRepo) GetUserMuteView(ctx context.Context, muteID int64) (*MuteView, error) {

	query := `SELECT m.id, m.user_id, m.reason, m.effective_until, m.created_at
	FROM mutes m
	WHERE m.id = $1
	`

	mute := &MuteView{}

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
	WHERE mc.mute_id = $1
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

	// Retrieve chat messages
	query = `SELECT cm.id, cm.content, cm.created_at
	FROM mute_chat_messages mcm
	JOIN chat_messages cm ON cm.id = mcm.chat_message_id
	WHERE mcm.mute_id = $1`

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

func (r *ModerateRepo) UnmuteUser(ctx context.Context, muteID int64) error {

	// Set effective until to now()
	query := `UPDATE mutes SET effective_until = NOW() WHERE id = $1 AND effective_until > NOW()`

	cmd, err := r.db.Exec(ctx, query, muteID)
	if err != nil {
		return fmt.Errorf("failed to unmute user: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return ErrRecordNotFound
	}

	return nil
}

type MuteQuery struct {
	UserID     *uuid.UUID
	ActiveOnly bool

	FromTime *time.Time
	ToTime   *time.Time

	Page     int64
	PageSize int64
}

func (sq *MuteQuery) Limit() int64 {
	return sq.PageSize
}

func (sq *MuteQuery) Offset() int64 {
	return (sq.Page - 1) * sq.PageSize
}

func (r *ModerateRepo) GetMutes(ctx context.Context, mq *MuteQuery) ([]*Mute, error) {
	query := `SELECT id, user_id, reason, effective_until, created_at
	FROM mutes
	WHERE ($1::UUID IS NULL OR user_id = $1)
		AND ($2::BOOLEAN IS FALSE OR effective_until > NOW())
		AND ($3::TIMESTAMPTZ IS NULL OR created_at >= $3)
		AND ($4::TIMESTAMPTZ IS NULL OR created_at <= $4)
	ORDER BY created_at DESC
	LIMIT $5 OFFSET $6
	`

	rows, err := r.db.Query(ctx, query, mq.UserID, mq.ActiveOnly, mq.FromTime, mq.ToTime, mq.Limit(), mq.Offset())
	if err != nil {
		return nil, fmt.Errorf("failed to query user mutes: %w", err)
	}

	defer rows.Close()

	mutes := []*Mute{}
	for rows.Next() {
		m := &Mute{}
		if err = rows.Scan(&m.ID, &m.UserID, &m.Reason, &m.EffectiveUntil, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user mute: %w", err)
		}
		mutes = append(mutes, m)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating user mutes: %w", rows.Err())
	}

	return mutes, nil
}

type ReportComment struct {
	ID             int64
	ReporterUserID uuid.UUID
	CommentID      int64
	CreatedAt      time.Time
}

func (r *ModerateRepo) ReportComment(ctx context.Context, rc *ReportComment) error {
	query := `INSERT INTO report_comments(reporter_user_id, comment_id) VALUES($1, $2) RETURNING id, created_at`
	err := r.db.QueryRow(ctx, query, rc.ReporterUserID, rc.CommentID).Scan(&rc.ID, &rc.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		switch {
		case errors.As(err, &pgErr) && pgErr.Code == pgerrcode.InvalidForeignKey:
			return fmt.Errorf("reporter user or comment not found: %w", ErrRecordNotFound)
		case errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation:
			return fmt.Errorf("comment already reported by this user: %w", ErrUniqueViolation)
		default:
			return fmt.Errorf("failed to insert report comment: %w", err)
		}
	}

	return nil
}

type ReportSort string

const (
	ReportQuerySortNewest       ReportSort = "newest"
	ReportQuerySortMostReported ReportSort = "mostReported"
)

var reportSortSafeMap = map[ReportSort]string{
	ReportQuerySortNewest:       "c.created_at DESC",
	ReportQuerySortMostReported: "nb_reports DESC",
}

type ReportQuery struct {
	ReportedUserID *uuid.UUID
	MarketID       *uuid.UUID
	NonDeletedOnly bool

	FromTime *time.Time
	ToTime   *time.Time

	Page     int64
	PageSize int64

	Sort ReportSort
}

func (sq *ReportQuery) Limit() int64 {
	return sq.PageSize
}

func (sq *ReportQuery) Offset() int64 {
	return (sq.Page - 1) * sq.PageSize
}

func (sq *ReportQuery) GetOrderBy() string {
	if orderBy, ok := reportSortSafeMap[sq.Sort]; ok {
		return orderBy
	}
	panic("unsafe sort value")
}

type CommentReportView struct {
	CommentID  int64
	Content    string
	MarketID   uuid.UUID
	MarketName string
	UserID     uuid.UUID
	Username   string
	CreatedAt  time.Time
	NbReports  int64
}

func (r *ModerateRepo) SearchReportedComments(ctx context.Context, rq *ReportQuery) ([]*CommentReportView, *meta.Metadata, error) {
	query := fmt.Sprintf(`SELECT count(*) OVER() AS total_count,
	c.id, c.content, 
	c.market_id, m.name AS market_name,
	c.user_id, u.username, 
	c.created_at, 
	COUNT(rc.id) AS nb_reports
	FROM report_comments rc
	JOIN comments c ON c.id = rc.comment_id
	JOIN users u ON u.id = c.user_id
	JOIN markets m ON m.id = c.market_id
	WHERE ($1::UUID IS NULL OR c.user_id = $1)
		AND ($2::UUID IS NULL OR c.market_id = $2)
		AND ($3::BOOLEAN IS FALSE OR c.is_deleted = FALSE)
		AND ($4::TIMESTAMPTZ IS NULL OR rc.created_at >= $4)
		AND ($5::TIMESTAMPTZ IS NULL OR rc.created_at <= $5)
	GROUP BY c.id, c.market_id, m.name, c.user_id, u.username
	ORDER BY %s, id DESC
	LIMIT $6 OFFSET $7
	`, rq.GetOrderBy())

	rows, err := r.db.Query(ctx, query, rq.ReportedUserID, rq.MarketID, rq.NonDeletedOnly, rq.FromTime, rq.ToTime, rq.Limit(), rq.Offset())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query reported comments: %w", err)
	}

	defer rows.Close()

	reports := []*CommentReportView{}
	var totalCount int64

	for rows.Next() {
		cr := &CommentReportView{}
		if err = rows.Scan(&totalCount,
			&cr.CommentID, &cr.Content,
			&cr.MarketID, &cr.MarketName,
			&cr.UserID, &cr.Username,
			&cr.CreatedAt,
			&cr.NbReports); err != nil {
			return nil, nil, fmt.Errorf("failed to scan reported comment: %w", err)
		}
		reports = append(reports, cr)
	}

	if rows.Err() != nil {
		return nil, nil, fmt.Errorf("error iterating bets rows: %w", rows.Err())
	}

	metadata := meta.CalculateMetadata(totalCount, rq.Page, rq.PageSize)

	return reports, metadata, nil

}
