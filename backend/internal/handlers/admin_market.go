package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"seer/internal/market"
	"seer/internal/utils"
	"slices"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type AdminMarketHandler struct {
	validate *validator.Validate
	am       *market.AdminManager
	tm       *market.TransactionManager
}

func NewAdminMarketHandler(validate *validator.Validate, am *market.AdminManager, tm *market.TransactionManager) *AdminMarketHandler {
	return &AdminMarketHandler{
		validate: validate,
		am:       am,
		tm:       tm,
	}
}

type outcomeCreateReq struct {
	Name     string `json:"name" validate:"required"`
	Position int64  `json:"position" validate:"required,gt=0"`
}

type createMarketReq struct {
	Name        string                   `json:"name" validate:"required,min=5"`
	Description string                   `json:"description" validate:"required,min=10"`
	Q0_Seeding  int64                    `json:"q0Seeding" validate:"required,min=1,max=100000"`     // Max seeding of 1000 USDT per shares
	AlphaPPM    int64                    `json:"alphaPPM" validate:"required,min=10000,max=1000000"` // Between 0.01 and 1
	FeePPM      int64                    `json:"feePPM" validate:"required,min=10000,max=100000"`    // Between 1% and 10%
	OutcomeSort market.MarketOutcomeSort `json:"outcomeSort" validate:"required,oneof=price position"`
	CloseTime   *time.Time               `json:"closeTime"` // Between 1 hour and 7 days
	CategoryIDs []int64                  `json:"categoryIds" validate:"required,dive,gt=0"`
	Outcomes    []outcomeCreateReq       `json:"outcomes" validate:"required,min=2"`
}

func (h *AdminMarketHandler) CreateMarket(c echo.Context) error {

	r := &createMarketReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, r, h.validate); err != nil {
		return err
	}

	m := &market.Market{
		Name:        r.Name,
		Description: r.Description,
		AlphaPPM:    r.AlphaPPM,
		FeePPM:      r.FeePPM,
		Q0Seeding:   r.Q0_Seeding,
		OutcomeSort: r.OutcomeSort,
		Status:      market.StatusDraft,
	}

	if r.CloseTime != nil {
		m.CloseTime = sql.NullTime{Valid: true, Time: *r.CloseTime}
	}

	outcomes := make([]*market.Outcome, 0, len(r.Outcomes))
	for _, o := range r.Outcomes {
		outcomes = append(outcomes, &market.Outcome{Name: o.Name, Position: o.Position})
	}

	ctx := c.Request().Context()

	err := h.am.CreateMarket(ctx, m, r.CategoryIDs, outcomes)
	if err != nil {
		return fmt.Errorf("failed to create market: %w", err)
	}

	return c.JSON(http.StatusCreated, utils.Envelope{"market_id": m.ID})

}

type statusMarketReq struct {
	MarketID uuid.UUID `json:"marketId"`
}

func (h *AdminMarketHandler) ResumeMarket(c echo.Context) error {

	r := &statusMarketReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, r, h.validate); err != nil {
		return err
	}

	ctx := c.Request().Context()

	// Validate current market status is compatible with resuming
	status, err := h.am.GetMarketStatus(ctx, r.MarketID)

	if err != nil {
		return fmt.Errorf("failed to get market status: %w", err)
	}

	if slices.Contains([]market.MarketStatus{market.StatusCancelled, market.StatusResolved}, status) {
		return echo.NewHTTPError(http.StatusBadRequest, "market cancelled or closed")
	}

	if err := h.am.ResumeMarket(ctx, r.MarketID); err != nil {
		return fmt.Errorf("failed to resume market: %w", err)
	}

	return c.JSON(http.StatusOK, utils.Envelope{"message": "market successfully resumed"})

}

func (h *AdminMarketHandler) PauseMarket(c echo.Context) error {
	r := &statusMarketReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, r, h.validate); err != nil {
		return err
	}

	ctx := c.Request().Context()

	// Validate current market status is compatible with resuming
	status, err := h.am.GetMarketStatus(ctx, r.MarketID)

	if err != nil {
		return fmt.Errorf("failed to get market status: %w", err)
	}

	if slices.Contains([]market.MarketStatus{market.StatusCancelled, market.StatusResolved}, status) {
		return echo.NewHTTPError(http.StatusBadRequest, "market cancelled or closed")
	}

	if err := h.am.ResumeMarket(ctx, r.MarketID); err != nil {
		return fmt.Errorf("failed to pause market: %w", err)
	}

	return c.JSON(http.StatusOK, utils.Envelope{"message": "market successfully paused"})
}

func (h *AdminMarketHandler) CancelMarket(c echo.Context) error {

	r := &statusMarketReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, r, h.validate); err != nil {
		return err
	}

	ctx := c.Request().Context()

	// Validate current market status is compatible with cancelling
	status, err := h.am.GetMarketStatus(ctx, r.MarketID)

	if err != nil {
		return fmt.Errorf("failed to get market status: %w", err)
	}

	if slices.Contains([]market.MarketStatus{market.StatusCancelled, market.StatusResolved}, status) {
		return echo.NewHTTPError(http.StatusBadRequest, "market cancelled or closed")
	}

	if err := h.tm.CancelMarket(ctx, r.MarketID); err != nil {
		return fmt.Errorf("failed to cancel market: %w", err)
	}

	return c.JSON(http.StatusOK, utils.Envelope{"message": "market successfully cancelled"})
}

type resolveMarketReq struct {
	MarketID         uuid.UUID `json:"marketId"`
	WinningOutcomeID int64     `json:"winningOutcomeId" validate:"required"`
}

func (h *AdminMarketHandler) ResolveMarket(c echo.Context) error {

	r := &resolveMarketReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, r, h.validate); err != nil {
		return err
	}

	ctx := c.Request().Context()

	// Validate current market status is compatible with resolving
	status, err := h.am.GetMarketStatus(ctx, r.MarketID)

	if err != nil {
		return fmt.Errorf("failed to get market status: %w", err)
	}

	if slices.Contains([]market.MarketStatus{market.StatusCancelled, market.StatusResolved}, status) {
		return echo.NewHTTPError(http.StatusBadRequest, "market cancelled or closed")
	}

	if err := h.tm.SettleMarket(ctx, r.MarketID, r.WinningOutcomeID); err != nil {
		return fmt.Errorf("failed to cancel market: %w", err)
	}

	return c.JSON(http.StatusOK, utils.Envelope{"message": "market successfully cancelled"})
}

type updateMarketFeeReq struct {
	MarketID uuid.UUID `json:"marketId"`
	FeePPM   int64     `json:"feePPM" validate:"required,min=10000,max=100000"`
}

func (h *AdminMarketHandler) UpdateMarketFee(c echo.Context) error {

	r := &updateMarketFeeReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, r, h.validate); err != nil {
		return err
	}

	ctx := c.Request().Context()

	// Validate current market status is compatible with updating fee
	status, err := h.am.GetMarketStatus(ctx, r.MarketID)

	if err != nil {
		return fmt.Errorf("failed to get market status: %w", err)
	}

	if slices.Contains([]market.MarketStatus{market.StatusCancelled, market.StatusResolved}, status) {
		return echo.NewHTTPError(http.StatusBadRequest, "market cancelled or closed")
	}

	if err := h.am.UpdateMarketFees(ctx, r.MarketID, r.FeePPM); err != nil {
		return fmt.Errorf("failed to update market fee: %w", err)
	}

	return c.JSON(http.StatusOK, utils.Envelope{"message": "market fee successfully updated"})

}

type updateOutcomeSortReq struct {
	MarketID    uuid.UUID                `json:"marketId"`
	OutcomeSort market.MarketOutcomeSort `json:"outcomeSort" validate:"required,oneof=price position"`
}

func (h *AdminMarketHandler) UpdateOutcomeSort(c echo.Context) error {

	r := &updateOutcomeSortReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, r, h.validate); err != nil {
		return err
	}

	ctx := c.Request().Context()

	err := h.am.UpdateOutcomeSort(ctx, r.MarketID, r.OutcomeSort)
	if err != nil {
		return fmt.Errorf("failed to update outcome sort: %w", err)
	}

	return c.JSON(http.StatusOK, utils.Envelope{"message": "market outcome sort successfully updated"})

}

type outcomePosition struct {
	OutcomeID int64 `json:"outcomeId" validate:"required"`
	Position  int64 `json:"position" validate:"required,gt=0"`
}

type updateOutcomePositionsReq struct {
	MarketID          uuid.UUID         `json:"marketId"`
	OutcomesPositions []outcomePosition `json:"outcomesPositions" validate:"required,dive"`
}

func (h *AdminMarketHandler) UpdateOutcomesPositions(c echo.Context) error {

	r := &updateOutcomePositionsReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, r, h.validate); err != nil {
		return err
	}

	// Validate poistions are unique, sequential and start at 1
	slices.SortFunc(r.OutcomesPositions, func(a, b outcomePosition) int { return int(a.Position - b.Position) })

	for i, o := range r.OutcomesPositions {
		if o.Position != int64(i+1) {
			return echo.NewHTTPError(http.StatusBadRequest, "positions must be sequential and start at 1")
		}
	}

	ctx := c.Request().Context()

	outcomes := make([]*market.Outcome, 0, len(r.OutcomesPositions))
	for _, o := range r.OutcomesPositions {
		outcomes = append(outcomes, &market.Outcome{ID: o.OutcomeID, Position: o.Position})
	}

	err := h.am.UpdateOutcomePositions(ctx, r.MarketID, outcomes)
	if err != nil {
		return fmt.Errorf("failed to update outcome positions: %w", err)
	}

	return c.JSON(http.StatusOK, utils.Envelope{"message": "market outcome positions successfully updated"})

}
