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

type TransactionHandler struct {
	validate *validator.Validate
	tm       *market.TransactionManager
}

func NewTransactionHandler(validate *validator.Validate, tm *market.TransactionManager) *TransactionHandler {
	return &TransactionHandler{
		validate: validate,
		tm:       tm,
	}
}

type betReq struct {
	BetAmountCents  int64     `json:"betAmountCents" validate:"required,min=100,max=1000000"` // Min 1 USDT, max 10k USDT
	QuotedGainCents int64     `json:"quotedGainCents" validate:"required,gtfield=BetAmountCents"`
	MarketID        uuid.UUID `json:"marketId" validate:"required"`
	OutcomeID       int64     `json:"outcomeId" validate:"required"`
	Currency        string    `json:"currency" validate:"required,oneof=USDT"`
	IdempotencyKey  string    `json:"idempotencyKey" validate:"required,max=36"`
}

func (h *TransactionHandler) PlaceBet(c echo.Context) error {

	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		fmt.Println("elpased", elapsed)
	}()

	b := &betReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, b, h.validate); err != nil {
		return err
	}

	user := utils.ContextGetUser(c)
	ctx := c.Request().Context()

	br := market.BetRequest{
		UserID:          user.ID,
		MarketID:        b.MarketID,
		OutcomeID:       b.OutcomeID,
		BetAmountCents:  b.BetAmountCents,
		QuotedGainCents: b.QuotedGainCents,
		IdempotencyKey:  b.IdempotencyKey,
		Currency:        "USDT",
	}

	if _, err := h.tm.AddBet(ctx, br); err != nil {
		return mapErrorRepo(err)
	}

	return echo.NewHTTPError(http.StatusOK, "bet succesfully placed")
}
