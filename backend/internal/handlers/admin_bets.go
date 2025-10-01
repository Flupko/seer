package handlers

import (
	"fmt"
	"net/http"
	"seer/internal/market"
	"seer/internal/utils"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type AdminBetHandler struct {
	validate *validator.Validate
	bm       *market.BetManager
}

func NewAdminBetHandler(v *validator.Validate, bm *market.BetManager) *AdminBetHandler {
	return &AdminBetHandler{
		validate: v,
		bm:       bm,
	}
}

type adminBetSearchReq struct {
	MarketID *uuid.UUID        `json:"marketID"`
	UserID   *uuid.UUID        `json:"userID"`
	Status   *market.BetStatus `json:"betStatus" validate:"omitempty,oneof=active won lost resolved"`

	MinPriceCents *int64 `json:"minPriceCents" validate:"omitempty,min=100"`          // Min 1 USDT
	MaxPriceCents *int64 `json:"maxPriceCents" validate:"omitempty,max=100000000000"` // Max 1B USDT

	FromTime *time.Time `json:"fromTime"`
	ToTime   *time.Time `json:"toTime" validate:"omitempty,gtfield=FromTime"`

	PageSize int64 `json:"pageSize" validate:"min=4,max=20"`
	Page     int64 `json:"page" validate:"min=1"`

	Sort    market.SortBet `json:"sort" validate:"omitempty,oneof=placedAt wager payout"`
	SortDir string         `json:"sortDir" validate:"omitempty,oneof=asc desc"`
}

type adminBetSearchRes struct {
	ID              uuid.UUID        `json:"id"`
	UserID          uuid.UUID        `json:"userId"`
	LedgerAccountID uuid.UUID        `json:"ledgerAccountId"`
	Status          market.BetStatus `json:"betStatus"`
	PricePaidCents  int64            `json:"pricePaidCents"`
	PayoutCents     int64            `json:"payoutCents"`
	FeePaidCents    int64            `json:"feePaidCents"`
	FeePPM          int64            `json:"feePPM"`
	PricePPM        int64            `json:"pricePPM"`
	MarketID        uuid.UUID        `json:"marketId"`
	MarketName      string           `json:"marketName"`
	OutcomeID       int64            `json:"outcomesId"`
	OutcomeName     string           `json:"outcomeName"`
	PlacedAt        time.Time        `json:"placeAt"`
	IdempotencyKey  string           `json:"idempotencyKey"`
}

func (h *AdminBetHandler) GetBetsAdmin(c echo.Context) error {

	r := &adminBetSearchReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, r, h.validate); err != nil {
		return err
	}

	if r.Sort == "" {
		r.Sort = market.SortPlacedAt
		r.SortDir = "desc"
	} else if r.SortDir == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "sortDir is required if sort is provided")
	}

	bsq := &market.BetSearchQuery{
		UserID:   r.UserID,
		MarketID: r.MarketID,
		Status:   r.Status,

		MinPriceCents: r.MinPriceCents,
		MaxPriceCents: r.MaxPriceCents,

		Page:     r.Page,
		PageSize: r.PageSize,

		Sort:    r.Sort,
		SortDir: r.SortDir,
	}

	ctx := c.Request().Context()
	betsView, metadata, err := h.bm.SearchBets(ctx, bsq)
	if err != nil {
		return fmt.Errorf("failed to get bets for user: %w", err)
	}

	betsResp := make([]*adminBetSearchRes, 0, len(betsView))
	for _, b := range betsView {
		br := &adminBetSearchRes{
			ID:              b.ID,
			Status:          b.Status,
			LedgerAccountID: b.LedgerAccountID,
			UserID:          b.User.ID,
			FeePaidCents:    b.FeePaidCents,
			FeePPM:          b.FeePPM,
			PricePaidCents:  b.TotalPricePaidCents,
			PricePPM:        b.PricePPM,
			PayoutCents:     b.PayoutCents,
			MarketID:        b.MarketID,
			MarketName:      b.MarketName,
			OutcomeID:       b.OutcomeID,
			OutcomeName:     b.OutcomeName,
			PlacedAt:        b.PlacedAt,
			IdempotencyKey:  b.IdempotencyKey,
		}
		betsResp = append(betsResp, br)
	}

	return c.JSON(http.StatusOK, utils.Envelope{"bets": betsResp, "metadata": metadata})
}
