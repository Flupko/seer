package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"seer/internal/finance"
	"seer/internal/market"
	"seer/internal/utils"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type MarketHandler struct {
	validate *validator.Validate
	msm      *market.StateManager
	bm       *market.BetManager
	qm       *market.QueryManager
	blm      *market.BetLiveManager
}

func NewMarketHandler(v *validator.Validate, msm *market.StateManager, bm *market.BetManager, qm *market.QueryManager, blm *market.BetLiveManager) *MarketHandler {
	return &MarketHandler{
		validate: v,
		msm:      msm,
		bm:       bm,
		qm:       qm,
		blm:      blm,
	}
}

type quoteReq struct {
	BetAmountCents int64     `json:"betAmountCents" validate:"required,min=100,max=1000000"` // Min 1 USDT, max 10k USDT
	MarketID       uuid.UUID `json:"marketId" validate:"required"`
	OutcomeID      int64     `json:"outcomeId" validate:"required"`
}

type quoteRes struct {
	GainCents int64 `json:"gainCents"`
	ProbPPM   int64 `json:"probPPM"`
}

func (h *MarketHandler) GetQuote(c echo.Context) error {

	q := &quoteReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, q, h.validate); err != nil {
		return err
	}

	ctx := c.Request().Context()
	gainCents, pricePPM, err := h.msm.GetQuoteForBet(ctx, q.BetAmountCents, q.MarketID, q.OutcomeID)
	if err != nil {
		return mapErrorRepo(err)
	}

	return c.JSON(http.StatusOK, quoteRes{GainCents: gainCents, ProbPPM: pricePPM})
}

type userBetSearchReq struct {
	MarketID *uuid.UUID        `json:"marketID"`
	Status   *market.BetStatus `json:"betStatus" validate:"omitempty,oneof=active won lost resolved"`
	PageSize int64             `json:"pageSize" validate:"min=4,max=20"`
	Page     int64             `json:"page" validate:"min=1"`
	Sort     market.SortBet    `json:"sort" validate:"omitempty,oneof=placedAt wager payout"`
	SortDir  string            `json:"sortDir" validate:"omitempty,oneof=asc desc"`
}

type userBetSearchRes struct {
	ID             uuid.UUID        `json:"id"`
	Status         market.BetStatus `json:"betStatus"`
	PricePaidCents int64            `json:"pricePaidCents"`
	PayoutCents    int64            `json:"payoutCents"`
	MarketID       uuid.UUID        `json:"marketId"`
	MarketName     string           `json:"marketName"`
	OutcomeID      int64            `json:"outcomesId"`
	OutcomeName    string           `json:"outcomeName"`
	PlacedAt       time.Time        `json:"placeAt"`
}

func (h *MarketHandler) GetBetsUser(c echo.Context) error {

	r := &userBetSearchReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, r, h.validate); err != nil {
		return err
	}

	if r.Sort == "" {
		r.Sort = market.SortPlacedAt
		r.SortDir = "desc"
	} else if r.SortDir == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "sortDir is required if sort is provided")
	}

	user := utils.ContextGetUser(c)
	ctx := c.Request().Context()

	bsq := &market.BetSearchQuery{
		UserID:   &user.ID,
		MarketID: r.MarketID,
		Status:   r.Status,
		Page:     r.Page,
		PageSize: r.PageSize,
		Sort:     r.Sort,
		SortDir:  r.SortDir,
	}

	betsView, metadata, err := h.bm.SearchBets(ctx, bsq)
	if err != nil {
		return fmt.Errorf("failed to get bets for user: %w", err)
	}

	betsResp := make([]*userBetSearchRes, 0, len(betsView))
	for _, b := range betsView {
		br := &userBetSearchRes{
			ID:             b.ID,
			Status:         b.Status,
			PricePaidCents: b.TotalPricePaidCents,
			PayoutCents:    b.PayoutCents,
			MarketID:       b.MarketID,
			MarketName:     b.MarketName,
			OutcomeID:      b.OutcomeID,
			OutcomeName:    b.OutcomeName,
			PlacedAt:       b.PlacedAt,
		}
		betsResp = append(betsResp, br)
	}

	return c.JSON(http.StatusOK, utils.Envelope{"bets": betsResp, "metadata": metadata})

}

type marketSearchUserReq struct {
	Query      *string           `json:"query" validate:"omitempty,min=3,max=50"`
	CategoryID *int64            `json:"categoryId" validate:"omitempty,gt=0"`
	Sort       market.SortMarket `json:"sort" validate:"required,oneof=hot volume newest endingSoon"`
	PageSize   int64             `json:"pageSize" validate:"min=4,max=20"`
	Page       int64             `json:"page" validate:"min=1"`
}

type outcomeUserRes struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Position int64  `json:"position"`
	ProbPPM  int64  `json:"probPPM"`
	Active   bool   `json:"active"`
}

type categoryRes struct {
	ID    int64  `json:"id"`
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type marketSearcUserhRes struct {
	ID          uuid.UUID                `json:"id"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	CloseTime   time.Time                `json:"closeTime"`
	OutcomeSort market.MarketOutcomeSort `json:"outcomeSort"`
	Categories  []*categoryRes           `json:"categories"`
	Outcomes    []*outcomeUserRes        `json:"outcomes"`
}

func (h *MarketHandler) GetMarketsUser(c echo.Context) error {

	r := &marketSearchUserReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, r, h.validate); err != nil {
		return err
	}

	ctx := c.Request().Context()

	msq := &market.SearchQuery{
		Query:      r.Query,
		CategoryID: r.CategoryID,
		Sort:       r.Sort,
		Status:     market.StatusOpened,
		Page:       r.Page,
		PageSize:   r.PageSize,
	}

	marketsView, metadata, err := h.qm.SearchMarkets(ctx, msq, false)
	if err != nil {
		return fmt.Errorf("failed to search markets: %w", err)
	}

	markets := make([]*marketSearcUserhRes, 0, len(marketsView))
	for _, m := range marketsView {
		mr := &marketSearcUserhRes{
			ID:          m.ID,
			Name:        m.Name,
			Description: m.Description,
			CloseTime:   m.CloseTime.Time,
			OutcomeSort: m.OutcomeSort,
			Categories:  make([]*categoryRes, 0, len(m.Categories)),
			Outcomes:    make([]*outcomeUserRes, 0, len(m.Outcomes)),
		}

		for _, c := range m.Categories {
			cr := &categoryRes{
				ID:    c.ID,
				Slug:  c.Slug,
				Label: c.Label,
			}
			mr.Categories = append(mr.Categories, cr)
		}

		for _, o := range m.Outcomes {
			or := &outcomeUserRes{
				ID:       o.ID,
				Name:     o.Name,
				Position: o.Position,
				Active:   o.Active,
				ProbPPM:  o.PricePPM,
			}
			mr.Outcomes = append(mr.Outcomes, or)
		}

		markets = append(markets, mr)
	}

	return c.JSON(http.StatusOK, utils.Envelope{"markets": markets, "metadata": metadata})
}

func mapErrorRepo(err error) *echo.HTTPError {
	switch {
	case errors.Is(err, finance.ErrIdempotency):
		return echo.NewHTTPError(http.StatusBadRequest, finance.ErrIdempotency.Error())
	case errors.Is(err, market.ErrMarketNotFound):
		return echo.NewHTTPError(http.StatusBadRequest, market.ErrMarketNotFound.Error())
	case errors.Is(err, market.ErrOutcomeNotFound):
		return echo.NewHTTPError(http.StatusBadRequest, market.ErrOutcomeNotFound.Error())
	case errors.Is(err, market.ErrInvalidQuotedGain):
		return echo.NewHTTPError(http.StatusBadRequest, market.ErrInvalidQuotedGain.Error())
	case errors.Is(err, market.ErrInvalidBetAmount):
		return echo.NewHTTPError(http.StatusBadRequest, market.ErrInvalidBetAmount.Error())
	case errors.Is(err, finance.ErrInsufficientFunds):
		return echo.NewHTTPError(http.StatusBadRequest, finance.ErrInsufficientFunds.Error())
	case errors.Is(err, finance.ErrAccountNotFound):
		return echo.NewHTTPError(http.StatusBadRequest, finance.ErrAccountNotFound.Error())
	default:
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
}
