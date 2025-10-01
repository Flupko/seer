package repos

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrDepthLimitExceeded       = errors.New("depth limit exceeded")
	maxDepth              int64 = 5
)

const (
	CommentDelay = 1 * time.Minute
)

type Comment struct {
	ID        int64
	UserID    uuid.UUID
	MarketID  uuid.UUID
	Content   string
	CreatedAt time.Time

	IsDeleted bool
	DeletedAt sql.NullTime

	ParentID sql.NullInt64
	Depth    int64
}

type CommentRepo struct {
	db *pgxpool.Pool
}

func NewCommentRepo(db *pgxpool.Pool) *CommentRepo {
	return &CommentRepo{
		db: db,
	}
}

func (r *CommentRepo) CheckMarketExists(ctx context.Context, marketID uuid.UUID) (bool, error) {

	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM markets WHERE id = $1 AND status != 'draft')`
	err := r.db.QueryRow(ctx, query, marketID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check market exists: %w", err)
	}

	return exists, nil
}

func (r *CommentRepo) GetLastCommentTimeForUserMarket(ctx context.Context, userID uuid.UUID, marketID uuid.UUID) (time.Time, error) {

	query := `SELECT created_at 
	FROM comments
	WHERE user_id = $1 AND market_id = $2 
	ORDER BY created_at DESC 
	LIMIT 1`

	var lastCommentTime sql.NullTime

	err := r.db.QueryRow(ctx, query, userID, marketID).Scan(&lastCommentTime)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return time.Time{}, ErrRecordNotFound
		default:
			return time.Time{}, err
		}
	}

	return lastCommentTime.Time, nil
}

func (r *CommentRepo) AddComment(ctx context.Context, c *Comment) error {

	// Get parent comment's depth if parent_id is provided
	if c.ParentID.Valid {
		var parentDepth int64
		query := `SELECT depth FROM comments WHERE id = $1`
		err := r.db.QueryRow(ctx, query, c.ParentID.Int64).Scan(&parentDepth)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("parent comment not found: %w", ErrRecordNotFound)
			}
			return fmt.Errorf("failed to get parent comment depth: %w", err)
		}

		// Depth limit
		if parentDepth+1 > maxDepth {
			return ErrDepthLimitExceeded
		}

		c.Depth = parentDepth + 1
	} else {
		c.Depth = 0
	}

	query := `INSERT INTO comments (user_id, market_id, content, parent_id, depth)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING id, created_at`

	err := r.db.QueryRow(ctx, query, c.UserID, c.MarketID, c.Content, c.ParentID, c.Depth).Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert comment: %w", err)
	}

	return err
}

func (r *CommentRepo) DeleteComment(ctx context.Context, commentID int64, userID *uuid.UUID) error {

	query := `UPDATE comments SET is_deleted = TRUE, deleted_at = NOW()
	WHERE id = $1 AND (user_id::uuid IS NULL OR user_id = $2) AND is_deleted = FALSE`

	cmd, err := r.db.Exec(ctx, query, commentID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete comment: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return ErrRecordNotFound
	}

	return nil
}

type CommentQuery struct {
	UserID   *uuid.UUID
	MarketID *uuid.UUID
	ParentID *int64

	ShowDeleted bool

	Page     int64
	PageSize int64
}

func (sq *CommentQuery) Limit() int64 {
	return sq.PageSize
}

func (sq *CommentQuery) Offset() int64 {
	return (sq.Page - 1) * sq.PageSize
}

type CommentView struct {
	ID int64

	UserID   uuid.UUID
	Username string

	MarketID uuid.UUID

	NbReplies int64

	Content   string
	CreatedAt time.Time

	IsDeleted bool
	DeletedAt sql.NullTime
}

func (r *CommentRepo) SearchComments(ctx context.Context, cq *CommentQuery) ([]*CommentView, error) {

	query := `SELECT c.id, u.id, u.username, c.market_id, (SELECT COUNT(*) FROM comments cc WHERE cc.parent_id = c.id) as nb_replies, c.content, c.created_at, c.is_deleted, c.deleted_at
	FROM comments c
	JOIN users u ON u.id = c.user_id
	WHERE ($1::uuid IS NULL OR c.user_id = $1)
	AND ($2::uuid IS NULL OR c.market_id = $2)
	AND (c.parent_id IS NOT DISTINCT FROM $3::bigint)
	AND c.is_deleted = $4
	ORDER BY c.created_at DESC
	LIMIT $5 OFFSET $6`

	rows, err := r.db.Query(ctx, query, cq.UserID, cq.MarketID, cq.ParentID, cq.ShowDeleted, cq.Limit(), cq.Offset())
	if err != nil {
		return nil, fmt.Errorf("failed to query rows comments: %w", err)
	}

	defer rows.Close()

	comments := []*CommentView{}

	for rows.Next() {
		c := &CommentView{}
		err := rows.Scan(&c.ID, &c.UserID, &c.Username, &c.MarketID, &c.NbReplies, &c.Content, &c.CreatedAt, &c.IsDeleted, &c.DeletedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}
		comments = append(comments, c)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating markets rows: %w", rows.Err())
	}

	return comments, nil

}
