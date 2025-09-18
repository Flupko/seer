package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"seer/internal/market"
	"seer/internal/utils"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type AdminHandler struct {
	validate *validator.Validate
	am       *market.AdminManager
}

func NewAdminHandler(validate *validator.Validate, am *market.AdminManager) *AdminHandler {
	return &AdminHandler{
		validate: validate,
		am:       am,
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

func (h *AdminHandler) CreateMarket(c echo.Context) error {

	ctx := c.Request().Context()
	_ = ctx

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

	err := h.am.CreateMarket(ctx, m, r.CategoryIDs, outcomes)
	if err != nil {
		return fmt.Errorf("failed to create market: %w", err)
	}

	return c.JSON(http.StatusCreated, utils.Envelope{"market_id": m.ID})

}
