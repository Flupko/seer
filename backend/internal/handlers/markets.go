package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"seer/internal/finance"
	"seer/internal/market"
	"seer/internal/numeric"
	"seer/internal/repos"
	"seer/internal/utils"
	"slices"
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
	BetAmount *numeric.BigDecimal `query:"betAmount" validate:"required"`
	MarketID  uuid.UUID           `query:"marketId" validate:"required"`
	OutcomeID int64               `query:"outcomeId" validate:"required"`
}

type quoteRes struct {
	Gain     *numeric.BigDecimal `json:"gain"`
	AvgPrice *numeric.BigDecimal `json:"avgPrice"`
}

func (h *MarketHandler) GetQuoteBet(c echo.Context) error {

	q := &quoteReq{}
	if err := utils.ParseAndValidateQueryParams(c, q, h.validate); err != nil {
		return err
	}

	ctx := c.Request().Context()

	gain, avgPrice, err := h.msm.GetQuoteForBet(ctx, q.BetAmount, q.MarketID, q.OutcomeID)
	fmt.Println("gain", gain.String(), "avgPrice", avgPrice.String())
	if err != nil {
		fmt.Println("get quote failed", err)
		return mapErrorRepo(err)
	}

	return c.JSON(http.StatusOK, quoteRes{Gain: gain, AvgPrice: avgPrice})
}

type userBetSearchReq struct {
	MarketID *uuid.UUID        `query:"marketID"`
	Status   *market.BetStatus `query:"betStatus" validate:"omitempty,oneof=active won lost resolved"`
	PageSize int64             `query:"pageSize" validate:"min=4,max=20"`
	Page     int64             `query:"page" validate:"min=1"`
	Sort     market.SortBet    `query:"sort" validate:"omitempty,oneof=placedAt wager payout"`
	SortDir  string            `query:"sortDir" validate:"omitempty,oneof=asc desc"`
}

type userBetSearchRes struct {
	ID          uuid.UUID           `json:"id"`
	Status      market.BetStatus    `json:"betStatus"`
	PricePaid   *numeric.BigDecimal `json:"pricePaid"`
	Payout      *numeric.BigDecimal `json:"payout"`
	AvgPrice    *numeric.BigDecimal `json:"avgPrice"`
	MarketID    uuid.UUID           `json:"marketId"`
	MarketName  string              `json:"marketName"`
	OutcomeID   int64               `json:"outcomesId"`
	OutcomeName string              `json:"outcomeName"`
	PlacedAt    time.Time           `json:"placeAt"`
}

func (h *MarketHandler) GetPersonnalBets(c echo.Context) error {

	r := &userBetSearchReq{}
	if err := utils.ParseAndValidateQueryParams(c, r, h.validate); err != nil {
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
		fmt.Println("bets failed", err)
		return fmt.Errorf("failed to get bets for user: %w", err)
	}

	betsResp := make([]*userBetSearchRes, 0, len(betsView))
	for _, b := range betsView {
		br := &userBetSearchRes{
			ID:          b.ID,
			Status:      b.Status,
			PricePaid:   b.TotalPricePaid,
			Payout:      b.Payout,
			AvgPrice:    b.AvgPrice,
			MarketID:    b.MarketID,
			MarketName:  b.MarketName,
			OutcomeID:   b.OutcomeID,
			OutcomeName: b.OutcomeName,
			PlacedAt:    b.PlacedAt,
		}
		betsResp = append(betsResp, br)
	}

	return c.JSON(http.StatusOK, utils.Envelope{"bets": betsResp, "metadata": metadata})

}

type getBetReq struct {
	BetID uuid.UUID `param:"id" validate:"required"`
}

type publicBetRes struct {
	ID   uuid.UUID `json:"id"`
	User *struct {
		ID       uuid.UUID `json:"id"`
		Username string    `json:"username"`
	} `json:"user,omitempty"`
	Status      market.BetStatus    `json:"betStatus"`
	PricePaid   *numeric.BigDecimal `json:"pricePaid"`
	Payout      *numeric.BigDecimal `json:"payout"`
	AvgPrice    *numeric.BigDecimal `json:"avgPrice"`
	MarketID    uuid.UUID           `json:"marketId"`
	MarketName  string              `json:"marketName"`
	OutcomeID   int64               `json:"outcomesId"`
	OutcomeName string              `json:"outcomeName"`
	PlacedAt    time.Time           `json:"placeAt"`
}

func (h *MarketHandler) PublicGetBet(c echo.Context) error {

	r := &getBetReq{}
	if err := utils.ParseAndValidadePathParams(c, r, h.validate); err != nil {
		return err
	}

	ctx := c.Request().Context()
	betView, err := h.bm.GetBetView(ctx, r.BetID)
	if err != nil {
		switch {
		case errors.Is(err, repos.ErrRecordNotFound):
			return echo.NewHTTPError(http.StatusNotFound, "bet not found")
		default:
			return fmt.Errorf("failed to get bet view: %w", err)
		}
	}

	betResp := &publicBetRes{
		ID:          betView.ID,
		Status:      betView.Status,
		PricePaid:   betView.AvgPrice,
		Payout:      betView.Payout,
		AvgPrice:    betView.AvgPrice,
		MarketID:    betView.MarketID,
		MarketName:  betView.MarketName,
		OutcomeID:   betView.OutcomeID,
		OutcomeName: betView.OutcomeName,
		PlacedAt:    betView.PlacedAt,
	}
	if !betView.User.Hidden {
		betResp.User = &struct {
			ID       uuid.UUID `json:"id"`
			Username string    `json:"username"`
		}{
			ID:       betView.User.ID,
			Username: betView.User.Username,
		}
	}

	return c.JSON(http.StatusOK, utils.Envelope{"bet": betResp})

}

type marketSearchUserReq struct {
	Query        *string           `query:"query" validate:"omitempty,min=3,max=50"`
	CategorySlug *string           `query:"categorySlug" validate:"omitempty,min=1"`
	Sort         market.SortMarket `query:"sort" validate:"required,oneof=trending volume newest endingSoon"`
	Status       string            `query:"status" validate:"omitempty,oneof=active resolved"`
	PageSize     int64             `query:"pageSize" validate:"min=3,max=20"`
	Page         int64             `query:"page" validate:"min=1"`
}

type outcomeUserRes struct {
	ID       int64               `json:"id"`
	Name     string              `json:"name"`
	Position int64               `json:"position"`
	Quantity *numeric.BigDecimal `json:"quantity"`
}

type categoryRes struct {
	ID      int64  `json:"id"`
	Slug    string `json:"slug"`
	Label   string `json:"label"`
	IconUrl string `json:"iconUrl"`
}

type marketSearcUserRes struct {
	ID          uuid.UUID                `json:"id"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	ImgKey      string                   `json:"imgKey"`
	CloseTime   *time.Time               `json:"closeTime,omitempty"`
	Alpha       *numeric.BigDecimal      `json:"alpha"`
	Fee         *numeric.BigDecimal      `json:"fee"`
	CapPrice    *numeric.BigDecimal      `json:"capPrice"`
	OutcomeSort market.MarketOutcomeSort `json:"outcomeSort"`
	Categories  []*categoryRes           `json:"categories"`
	Outcomes    []*outcomeUserRes        `json:"outcomes"`
}

func (h *MarketHandler) GetMarketsUser(c echo.Context) error {

	r := &marketSearchUserReq{}
	if err := utils.ParseAndValidateQueryParams(c, r, h.validate); err != nil {
		return err
	}

	ctx := c.Request().Context()

	msq := &market.SearchQuery{
		Query:        r.Query,
		CategorySlug: r.CategorySlug,
		Sort:         r.Sort,
		Page:         r.Page,
		PageSize:     r.PageSize,
	}

	switch r.Status {
	case "active":
		msq.Status = market.StatusOpened
	case "resolved":
		msq.Status = market.StatusResolved
	default:
		msq.Status = market.StatusOpened
	}

	marketsView, metadata, err := h.qm.SearchMarkets(ctx, msq, false)
	if err != nil {
		fmt.Println("market search failed", err)
		return fmt.Errorf("failed to search markets: %w", err)
	}

	markets := make([]*marketSearcUserRes, 0, len(marketsView))
	for _, m := range marketsView {
		mr := &marketSearcUserRes{
			ID:          m.ID,
			Name:        m.Name,
			Description: m.Description,
			ImgKey:      m.ImgKey,
			OutcomeSort: m.OutcomeSort,
			Alpha:       m.Alpha,
			Fee:         m.Fee,
			CapPrice:    m.CapPrice,
			Categories:  make([]*categoryRes, 0, len(m.Categories)),
			Outcomes:    make([]*outcomeUserRes, 0, len(m.Outcomes)),
		}

		if m.CloseTime.Valid {
			mr.CloseTime = &m.CloseTime.Time
		}

		for _, c := range m.Categories {
			cr := &categoryRes{
				ID:      c.ID,
				Slug:    c.Slug,
				Label:   c.Label,
				IconUrl: c.IconUrl,
			}
			mr.Categories = append(mr.Categories, cr)
		}

		for _, o := range m.Outcomes {
			or := &outcomeUserRes{
				ID:       o.ID,
				Name:     o.Name,
				Position: o.Position,
			}
			mr.Outcomes = append(mr.Outcomes, or)
		}

		// Retrieve market state
		mState, err := h.msm.GetMarketState(ctx, m.ID)
		if err != nil {
			return fmt.Errorf("failed to get market state: %w", err)
		}

		prices, err := market.PricesBD(mState.QVec, m.Alpha, m.Fee)
		if err != nil {
			return fmt.Errorf("failed to compute prices: %w", err)
		}
		fmt.Println("market prices:", prices)

		// Attach quantities to outcomes
		for _, o := range mr.Outcomes {
			idx := slices.Index(mState.OutcomeIDs, o.ID)
			if idx == -1 {
				return fmt.Errorf("outcome id %d not found in market state for market %s", o.ID, m.ID)
			}
			o.Quantity = mState.QVec[idx]
		}

		markets = append(markets, mr)
	}

	return c.JSON(http.StatusOK, utils.Envelope{"markets": markets, "metadata": metadata})
}

type getMarketReq struct {
	MarketID uuid.UUID `path:"id" validate:"required"`
}

func (h *MarketHandler) GetMarketUser(c echo.Context) error {

	r := &getMarketReq{}
	if err := utils.ParseAndValidadePathParams(c, r, h.validate); err != nil {
		return err
	}

	ctx := c.Request().Context()
	m, err := h.qm.GetMarketByID(ctx, r.MarketID)
	if err != nil {
		return mapErrorRepo(err)
	}

	mr := &marketSearcUserRes{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		ImgKey:      m.ImgKey,
		OutcomeSort: m.OutcomeSort,
		Alpha:       m.Alpha,
		Fee:         m.Fee,
		CapPrice:    m.CapPrice,
		Categories:  make([]*categoryRes, 0, len(m.Categories)),
		Outcomes:    make([]*outcomeUserRes, 0, len(m.Outcomes)),
	}

	if m.CloseTime.Valid {
		mr.CloseTime = &m.CloseTime.Time
	}

	for _, c := range m.Categories {
		cr := &categoryRes{
			ID:      c.ID,
			Slug:    c.Slug,
			Label:   c.Label,
			IconUrl: c.IconUrl,
		}
		mr.Categories = append(mr.Categories, cr)
	}

	for _, o := range m.Outcomes {
		or := &outcomeUserRes{
			ID:       o.ID,
			Name:     o.Name,
			Position: o.Position,
		}
		mr.Outcomes = append(mr.Outcomes, or)
	}

	// Retrieve market state
	mState, err := h.msm.GetMarketState(ctx, m.ID)
	if err != nil {
		return fmt.Errorf("failed to get market state: %w", err)
	}

	// Attach quantities to outcomes
	for _, o := range mr.Outcomes {
		idx := slices.Index(mState.OutcomeIDs, o.ID)
		if idx == -1 {
			return fmt.Errorf("outcome id %d not found in market state for market %s", o.ID, m.ID)
		}
		o.Quantity = mState.QVec[idx]
	}

	return c.JSON(http.StatusOK, mr)

}

func (h *MarketHandler) GetAllFeaturedCategories(c echo.Context) error {
	ctx := c.Request().Context()
	categories, err := h.qm.GetAllFeaturedCategories(ctx)
	if err != nil {
		return fmt.Errorf("failed to get all categories: %w", err)
	}

	resp := make([]*categoryRes, 0, len(categories))
	for _, c := range categories {
		cr := &categoryRes{
			ID:      c.ID,
			Slug:    c.Slug,
			Label:   c.Label,
			IconUrl: c.IconUrl,
		}
		resp = append(resp, cr)
	}

	return c.JSON(http.StatusOK, resp)

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
