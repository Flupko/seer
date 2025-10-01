package handlers

import (
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

type AdminModerateHandler struct {
	validate *validator.Validate
	mr       *repos.ModerateRepo
}

func NewAdminModerateHandler(v *validator.Validate, mr *repos.ModerateRepo) *AdminModerateHandler {
	return &AdminModerateHandler{
		validate: v,
		mr:       mr,
	}
}

type createMuteReq struct {
	UserID          uuid.UUID   `json:"userId"`
	MuteDurationSec int64       `json:"muteDurationSec"`
	Reason          string      `json:"reason"`
	ChatMessagesIDs []uuid.UUID `json:"chatMessagesIds"`
	CommentsIDs     []int64     `json:"commentsIds"`
}

func (h *AdminModerateHandler) MuteUser(c echo.Context) error {

	r := &createMuteReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, r, h.validate); err != nil {
		return err
	}

	if r.ChatMessagesIDs == nil {
		r.ChatMessagesIDs = []uuid.UUID{}
	}

	if r.CommentsIDs == nil {
		r.CommentsIDs = []int64{}
	}

	muteDuration := time.Duration(r.MuteDurationSec * int64(time.Second))

	m := &repos.Mute{
		UserID:         r.UserID,
		Reason:         r.Reason,
		EffectiveUntil: time.Now().Add(muteDuration),
	}

	if err := h.mr.MuteUser(c.Request().Context(), m, r.ChatMessagesIDs, r.CommentsIDs); err != nil {
		switch {
		case errors.Is(err, repos.ErrRecordNotFound):
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		case errors.Is(err, repos.ErrUniqueViolation):
			return echo.NewHTTPError(http.StatusConflict, "one of the chat messages or comments is already included in another mute")
		default:
			return fmt.Errorf("failed to mute user: %w", err)
		}
	}

	return c.JSON(http.StatusCreated, utils.Envelope{"userMuteId": m.ID})

}

type getMuteReq struct {
	MuteID int64 `param:"muteId" validate:"required"`
}

type muteChatMessageRes struct {
	ID        uuid.UUID `json:"id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type muteCommentRes struct {
	ID        int64     `json:"id"`
	MarketID  uuid.UUID `json:"marketId"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type muteRes struct {
	ID             int64
	UserID         uuid.UUID
	Reason         string
	EffectiveUntil time.Time
	CreatedAt      time.Time
	ChatMessages   []*muteChatMessageRes
	Comments       []*muteCommentRes
}

func (h *AdminModerateHandler) GetUserMute(c echo.Context) error {

	r := &getMuteReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, r, h.validate); err != nil {
		return err
	}

	ctx := c.Request().Context()
	m, err := h.mr.GetUserMuteView(ctx, r.MuteID)
	if err != nil {
		switch {
		case errors.Is(err, repos.ErrRecordNotFound):
			return echo.NewHTTPError(http.StatusNotFound, "mute not found")
		default:
			return fmt.Errorf("failed to get mute: %w", err)
		}
	}

	resp := &muteRes{
		ID:             m.ID,
		UserID:         m.UserID,
		Reason:         m.Reason,
		EffectiveUntil: m.EffectiveUntil,
		CreatedAt:      m.CreatedAt,
		ChatMessages:   []*muteChatMessageRes{},
		Comments:       []*muteCommentRes{},
	}

	for _, cm := range m.ChatMessages {
		resp.ChatMessages = append(resp.ChatMessages, &muteChatMessageRes{
			ID:        cm.ID,
			Content:   cm.Content,
			CreatedAt: cm.CreatedAt,
		})
	}

	for _, cm := range m.Comments {
		resp.Comments = append(resp.Comments, &muteCommentRes{
			ID:        cm.ID,
			MarketID:  cm.MarketID,
			Content:   cm.Content,
			CreatedAt: cm.CreatedAt,
		})
	}

	return c.JSON(http.StatusOK, utils.Envelope{"userMute": resp})

}

type searchMutesReq struct {
	UserID     *uuid.UUID `json:"userId"`
	ActiveOnly bool       `json:"activeOnly"`

	FromTime *time.Time `json:"fromTime"`
	ToTime   *time.Time `json:"toTime" validate:"omitempty,gtfield=FromTime"`

	PageSize int64 `json:"pageSize" validate:"min=4,max=20"`
	Page     int64 `json:"page" validate:"min=1"`
}

func (h *AdminModerateHandler) SearchMutes(c echo.Context) error {

	r := &searchMutesReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, r, h.validate); err != nil {
		return err
	}

	mq := &repos.MuteQuery{
		UserID:     r.UserID,
		ActiveOnly: r.ActiveOnly,
		FromTime:   r.FromTime,
		ToTime:     r.ToTime,
		Page:       r.Page,
		PageSize:   r.PageSize,
	}

	ctx := c.Request().Context()
	mutes, err := h.mr.GetMutes(ctx, mq)
	if err != nil {
		return fmt.Errorf("failed to search mutes: %w", err)
	}

	return c.JSON(http.StatusOK, utils.Envelope{"userMutes": mutes})
}

type unmuteReq struct {
	MuteID int64 `json:"muteId" validate:"required"`
}

func (h *AdminModerateHandler) UnmuteUser(c echo.Context) error {
	r := &unmuteReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, r, h.validate); err != nil {
		return err
	}

	ctx := c.Request().Context()
	err := h.mr.UnmuteUser(ctx, r.MuteID)
	if err != nil {
		switch {
		case errors.Is(err, repos.ErrRecordNotFound):
			return echo.NewHTTPError(http.StatusNotFound, "mute not found or already inactive")
		default:
			return fmt.Errorf("failed to unmute user: %w", err)
		}
	}

	return c.JSON(http.StatusOK, utils.Envelope{"message": "user successfully unmuted"})
}

type getReportsReq struct {
	ReportedUserID *uuid.UUID `json:"reportedUserId"`
	MarketID       *uuid.UUID `json:"marketId"`
	NonDeletedOnly bool       `json:"nonDeletedOnly"`

	FromTime *time.Time `json:"fromTime"`
	ToTime   *time.Time `json:"toTime" validate:"omitempty,gtfield=FromTime"`

	PageSize int64 `json:"pageSize" validate:"min=4,max=20"`
	Page     int64 `json:"page" validate:"min=1"`

	Sort repos.ReportSort `json:"sort" validate:"oneof=newest mostReported"`
}

func (h *AdminModerateHandler) GetReportedComments(c echo.Context) error {

	r := &getReportsReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, r, h.validate); err != nil {
		return err
	}

	rq := &repos.ReportQuery{
		ReportedUserID: r.ReportedUserID,
		MarketID:       r.MarketID,
		NonDeletedOnly: r.NonDeletedOnly,
		FromTime:       r.FromTime,
		ToTime:         r.ToTime,
		Page:           r.Page,
		PageSize:       r.PageSize,
		Sort:           r.Sort,
	}

	ctx := c.Request().Context()
	reports, metadata, err := h.mr.SearchReportedComments(ctx, rq)
	if err != nil {
		return fmt.Errorf("failed to search reported comments: %w", err)
	}

	return c.JSON(http.StatusOK, utils.Envelope{"reportedComments": reports, "metadata": metadata})

}
