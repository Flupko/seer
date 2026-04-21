package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"seer/internal/repos"
	"seer/internal/utils"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type CommentHandler struct {
	validate *validator.Validate
	cr       *repos.CommentRepo
}

func NewCommentHandler(validate *validator.Validate, cr *repos.CommentRepo) *CommentHandler {
	return &CommentHandler{
		validate: validate,
		cr:       cr,
	}
}

type addComentReq struct {
	MarketID uuid.UUID `json:"marketId"`
	Content  string    `json:"content" validate:"min=3,max=1000"`
	ParentID *int64    `json:"parentId"` // 0 if no parent
}

type userCommentRes struct {
	ID         int64          `json:"id"`
	ParentID   *int64         `json:"parentId,omitempty"`
	User       *PublicUserRes `json:"user"`
	MarketID   uuid.UUID      `json:"marketId"`
	NbReplies  int64          `json:"nbReplies"`
	NbLikes    int64          `json:"nbLikes"`
	IsLiked    bool           `json:"isLiked"`
	IsReported bool           `json:"isReported"`
	Depth      int64          `json:"depth"`
	Content    string         `json:"content"`
	CreatedAt  time.Time      `json:"createdAt"`
}

func (h *CommentHandler) PostComment(c echo.Context) error {

	r := &addComentReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, r, h.validate); err != nil {
		return err
	}

	user := utils.ContextGetUser(c)
	if user.MutedUntil.Valid && user.MutedUntil.Time.After(time.Now()) {
		return echo.NewHTTPError(http.StatusForbidden, "Muted user can't post comments")
	}

	ctx := c.Request().Context()

	exists, err := h.cr.CheckMarketExists(ctx, r.MarketID)
	if err != nil {
		return fmt.Errorf("failed to check market exists: %w", err)
	}
	if !exists {
		return echo.NewHTTPError(http.StatusBadRequest, "market doesn't exist")
	}

	lastCommentTime, err := h.cr.GetLastCommentTimeForUserMarket(ctx, user.ID, r.MarketID)
	if err != nil && !errors.Is(err, repos.ErrRecordNotFound) {
		return fmt.Errorf("failed to get last comment time for user: %w", err)
	}

	if err == nil && time.Since(lastCommentTime) < repos.CommentDelay {
		return echo.NewHTTPError(http.StatusTooManyRequests, "rate limit comments")
	}

	comment := &repos.Comment{
		UserID:   user.ID,
		MarketID: r.MarketID,
		Content:  r.Content,
	}

	if r.ParentID != nil {
		comment.ParentID = sql.NullInt64{Int64: *r.ParentID, Valid: true}
	} else {
		comment.ParentID = sql.NullInt64{Valid: false}
	}

	err = h.cr.AddComment(ctx, comment)
	if err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}

	// Retrieve the comment's view
	cv, err := h.cr.GetCommentViewByID(ctx, comment.ID, user.ID)
	if err != nil {
		return fmt.Errorf("failed to get comment view: %w", err)
	}

	resp := &userCommentRes{
		ID:         cv.ID,
		User:       UserViewToPublicRes(&cv.User),
		MarketID:   cv.MarketID,
		NbReplies:  cv.NbReplies,
		Content:    cv.Content,
		CreatedAt:  cv.CreatedAt,
		Depth:      cv.Depth,
		NbLikes:    cv.NbLikes,
		IsLiked:    cv.IsLiked,
		IsReported: cv.IsReported,
	}

	if cv.ParentID.Valid {
		resp.ParentID = &cv.ParentID.Int64
	}

	return c.JSON(http.StatusCreated, utils.Envelope{"comment": resp})

}

type toggleLikeReq struct {
	CommentID int64 `json:"commentId" validate:"required"`
}

func (h *CommentHandler) LikeComment(c echo.Context) error {
	r := &toggleLikeReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, r, h.validate); err != nil {
		return err
	}

	ctx := c.Request().Context()
	user := utils.ContextGetUser(c)

	l := &repos.Like{
		UserID:    user.ID,
		CommentID: r.CommentID,
	}

	if err := h.cr.LikeComment(ctx, l); err != nil {
		switch {
		case errors.Is(err, repos.ErrRecordNotFound):
			return echo.NewHTTPError(http.StatusNotFound, "comment not found")
		default:
			return fmt.Errorf("failed to like comment: %w", err)
		}
	}

	return c.JSON(http.StatusOK, utils.Envelope{"message": "comment liked successfully"})

}

func (h *CommentHandler) UnlikeComment(c echo.Context) error {
	r := &toggleLikeReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, r, h.validate); err != nil {
		return err
	}

	ctx := c.Request().Context()
	user := utils.ContextGetUser(c)

	l := &repos.Like{
		UserID:    user.ID,
		CommentID: r.CommentID,
	}

	if err := h.cr.UnlikeComment(ctx, l); err != nil {
		switch {
		case errors.Is(err, repos.ErrRecordNotFound):
			return echo.NewHTTPError(http.StatusNotFound, "comment not found")
		default:
			return fmt.Errorf("failed to unlike comment: %w", err)
		}
	}

	return c.JSON(http.StatusOK, utils.Envelope{"message": "comment unliked successfully"})

}

type deleteCommentReq struct {
	CommentID int64 `json:"commentId" validate:"required"`
}

func (h *CommentHandler) UserDeleteComment(c echo.Context) error {

	r := &deleteCommentReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, r, h.validate); err != nil {
		return err
	}

	user := utils.ContextGetUser(c)
	ctx := c.Request().Context()

	err := h.cr.DeleteComment(ctx, r.CommentID, &user.ID)
	if err != nil {
		switch {
		case errors.Is(err, repos.ErrRecordNotFound):
			return echo.NewHTTPError(http.StatusNotFound, "comment not found")
		default:
			return fmt.Errorf("failed to delete comment: %w", err)
		}
	}

	return c.JSON(http.StatusOK, utils.Envelope{"message": "comment successfully deleted"})
}

func (h *CommentHandler) AdminDeleteComment(c echo.Context) error {
	r := &deleteCommentReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, r, h.validate); err != nil {
		return err
	}

	ctx := c.Request().Context()

	err := h.cr.DeleteComment(ctx, r.CommentID, nil)
	if err != nil {
		switch {
		case errors.Is(err, repos.ErrRecordNotFound):
			return echo.NewHTTPError(http.StatusNotFound, "comment not found")
		default:
			return fmt.Errorf("failed to delete comment: %w", err)
		}
	}

	return c.JSON(http.StatusNoContent, utils.Envelope{"message": "comment succesfully deleted"})
}

type commentSearchUserReq struct {
	MarketID uuid.UUID `query:"marketId" validate:"required"`
	ParentID *int64    `query:"parentId"`

	Page     int64 `query:"page" validate:"min=1"`
	PageSize int64 `query:"pageSize" validate:"min=4,max=20"`
}

func (h *CommentHandler) UserGetComments(c echo.Context) error {

	r := &commentSearchUserReq{}
	if err := utils.ParseAndValidateQueryParams(c, r, h.validate); err != nil {
		return err
	}

	ctx := c.Request().Context()
	user := utils.ContextGetUser(c)

	cq := &repos.CommentQuery{
		MarketID:    &r.MarketID,
		ParentID:    r.ParentID,
		ShowDeleted: false, // Pointer to false
		Page:        r.Page,
		PageSize:    r.PageSize,
	}

	comments, meta, err := h.cr.SearchComments(ctx, cq, user.ID)
	if err != nil {
		return fmt.Errorf("failed to search comments: %w", err)
	}

	resp := make([]*userCommentRes, 0, len(comments))
	for _, cv := range comments {
		cr := &userCommentRes{
			ID:         cv.ID,
			User:       UserViewToPublicRes(&cv.User),
			MarketID:   cv.MarketID,
			NbReplies:  cv.NbReplies,
			NbLikes:    cv.NbLikes,
			IsLiked:    cv.IsLiked,
			IsReported: cv.IsReported,
			Content:    cv.Content,
			CreatedAt:  cv.CreatedAt,
			Depth:      cv.Depth,
		}

		if cv.ParentID.Valid {
			cr.ParentID = &cv.ParentID.Int64
		}
		resp = append(resp, cr)
	}

	return c.JSON(http.StatusOK, utils.Envelope{"comments": resp, "metadata": meta})

}
