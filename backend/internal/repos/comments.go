package repos

import (
	"context"
	"database/sql"
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

var (
	ErrDepthLimitExceeded       = errors.New("depth limit exceeded")
	maxDepth              int64 = 5
)

const (
	CommentDelay = 0
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

type Like struct {
	ID        int64
	UserID    uuid.UUID
	CommentID int64
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

	return nil
}

func (r *CommentRepo) LikeComment(ctx context.Context, like *Like) error {

	query := `INSERT INTO likes(user_id, comment_id)
	VALUES($1, $2)
	`

	_, err := r.db.Exec(ctx, query, like.UserID, like.CommentID)
	if err != nil {
		var pgErr *pgconn.PgError
		switch {
		case errors.As(err, &pgErr) && (pgErr.Code == pgerrcode.UniqueViolation):
			if pgErr.Code == pgerrcode.UniqueViolation {
				return nil // Comment already liked, consider as a success
			}

			if pgErr.Code == pgerrcode.ForeignKeyViolation {
				return ErrRecordNotFound
			}

			fallthrough

		default:
			return fmt.Errorf("failed to like comment: %w", err)
		}
	}

	return nil
}

func (r *CommentRepo) UnlikeComment(ctx context.Context, like *Like) error {
	query := `DELETE FROM likes WHERE user_id = $1 AND comment_id = $2
	`

	_, err := r.db.Exec(ctx, query, like.UserID, like.CommentID)
	if err != nil {
		return fmt.Errorf("failed to unlike: %w", err)
	}

	return nil
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
	Comment
	NbReplies  int64
	NbLikes    int64
	IsLiked    bool
	IsReported bool
	User       UserView
}

func (r *CommentRepo) SearchComments(ctx context.Context, cq *CommentQuery, userID uuid.UUID) ([]*CommentView, *meta.Metadata, error) {

	query := `SELECT count(*) OVER() AS total_count,
	c.id, c.market_id, (SELECT COUNT(*) FROM comments cc WHERE cc.parent_id = c.id) as nb_replies, (SELECT COUNT(*) FROM likes l WHERE l.comment_id = c.id) as nb_likes,
	EXISTS(SELECT 1 FROM likes l WHERE l.comment_id = c.id AND l.user_id = $1) as is_liked, 
	EXISTS(SELECT 1 FROM report_comments r WHERE r.comment_id = c.id AND r.reporter_user_id = $1) as is_reported,
	c.content, c.created_at, c.is_deleted, c.deleted_at, c.depth, c.parent_id,
	u.id, u.username, u.profile_image_key, u.total_wagered, u.created_at
	FROM comments c
	JOIN users u ON u.id = c.user_id
	WHERE ($2::uuid IS NULL OR c.user_id = $2)
	AND ($3::uuid IS NULL OR c.market_id = $3)
	AND (c.parent_id IS NOT DISTINCT FROM $4::bigint)
	AND c.is_deleted = $5
	ORDER BY c.created_at DESC
	LIMIT $6 OFFSET $7`

	rows, err := r.db.Query(ctx, query, userID, cq.UserID, cq.MarketID, cq.ParentID, cq.ShowDeleted, cq.Limit(), cq.Offset())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query rows comments: %w", err)
	}

	defer rows.Close()

	comments := []*CommentView{}
	var totalCount int64

	for rows.Next() {
		c := &CommentView{}
		err := rows.Scan(&totalCount, &c.ID, &c.MarketID, &c.NbReplies, &c.NbLikes, &c.IsLiked, &c.IsReported, &c.Content, &c.CreatedAt, &c.IsDeleted, &c.DeletedAt, &c.Depth, &c.ParentID,
			&c.User.ID, &c.User.Username, &c.User.ProfileImageKey, &c.User.TotalWagered, &c.User.CreatedAt)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to scan comment: %w", err)
		}
		comments = append(comments, c)
	}

	if rows.Err() != nil {
		return nil, nil, fmt.Errorf("error iterating markets rows: %w", rows.Err())
	}

	metadata := meta.CalculateMetadata(totalCount, cq.Page, cq.PageSize)

	return comments, metadata, nil

}

func (r *CommentRepo) GetCommentViewByID(ctx context.Context, commentID int64, userID uuid.UUID) (*CommentView, error) {

	query := `SELECT c.id, c.market_id, (SELECT COUNT(*) FROM comments cc WHERE cc.parent_id = c.id) as nb_replies, (SELECT COUNT(*) FROM likes l WHERE l.comment_id = c.id) as nb_likes,
	EXISTS(SELECT 1 FROM likes l WHERE l.comment_id = c.id AND l.user_id = $1) as is_liked, 
	EXISTS(SELECT 1 FROM report_comments r WHERE r.comment_id = c.id AND r.reporter_user_id = $1) as is_reported,
	c.content, c.created_at, c.is_deleted, c.deleted_at, c.depth, c.parent_id,
	u.id, u.username, u.profile_image_key, u.total_wagered, u.created_at
	FROM comments c
	JOIN users u ON u.id = c.user_id
	WHERE c.id = $2`

	c := &CommentView{}
	err := r.db.QueryRow(ctx, query, userID, commentID).Scan(&c.ID, &c.MarketID, &c.NbReplies, &c.NbLikes, &c.IsLiked, &c.IsReported, &c.Content, &c.CreatedAt, &c.IsDeleted, &c.DeletedAt, &c.Depth, &c.ParentID,
		&c.User.ID, &c.User.Username, &c.User.ProfileImageKey, &c.User.TotalWagered, &c.User.CreatedAt)

	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, fmt.Errorf("failed to query comment view: %w", err)
		}

	}

	return c, nil
}
