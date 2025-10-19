package handlers

import (
	"fmt"
	"net/http"
	"seer/internal/finance"
	"seer/internal/market"
	"seer/internal/numeric"
	"seer/internal/utils"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type TransactionHandler struct {
	validate *validator.Validate
	tm       *market.TransactionManager
	fm       *finance.FinanceManager
}

func NewTransactionHandler(validate *validator.Validate, tm *market.TransactionManager, fm *finance.FinanceManager) *TransactionHandler {
	return &TransactionHandler{
		validate: validate,
		tm:       tm,
		fm:       fm,
	}
}

type betReq struct {
	BetAmount      *numeric.BigDecimal `json:"betAmount" validate:"required"`
	MinWantedGain  *numeric.BigDecimal `json:"minWantedGain" validate:"required,dec_scale=2,dec_min=0.5,dec_max=1000000"`
	MarketID       uuid.UUID           `json:"marketId" validate:"required"`
	OutcomeID      int64               `json:"outcomeId" validate:"required"`
	Currency       finance.Currency    `json:"currency" validate:"required,oneof=USDT"`
	IdempotencyKey string              `json:"idempotencyKey" validate:"required,max=36"`
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

	// Retrieve the user's ledger account ID for the specified currency
	userLedgerAccountID, err := h.fm.GetLedgerAccountForCurrency(ctx, user.ID, b.Currency, finance.AccountLiability)
	if err != nil {
		return fmt.Errorf("failed to get user ledger account ID: %w", err)
	}

	br := market.BetRequest{
		LedgerAccountID: userLedgerAccountID,
		MarketID:        b.MarketID,
		OutcomeID:       b.OutcomeID,
		BetAmount:       b.BetAmount,
		MinWantedGain:   b.MinWantedGain,
		IdempotencyKey:  b.IdempotencyKey,
		Currency:        "USDT",
	}

	if _, err := h.tm.AddBet(ctx, br); err != nil {
		return mapErrorRepo(err)
	}

	return echo.NewHTTPError(http.StatusOK, "bet succesfully placed")
}

type cashoutReq struct {
	BetID          uuid.UUID           `json:"betId" validate:"required"`
	MinWantedGain  *numeric.BigDecimal `json:"minWantedGain" validate:"required,dec_scale=2,dec_max=1000000"`
	IdempotencyKey string              `json:"idempotencyKey" validate:"required,max=36"`
}

func (h *TransactionHandler) CashoutBet(c echo.Context) error {
	r := &cashoutReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, r, h.validate); err != nil {
		return err
	}

	user := utils.ContextGetUser(c)
	ctx := c.Request().Context()

	cr := &market.CashoutRequest{
		BetID:          r.BetID,
		UserID:         user.ID,
		MinWantedGain:  r.MinWantedGain,
		IdempotencyKey: r.IdempotencyKey,
	}

	if _, err := h.tm.CashoutBet(ctx, cr); err != nil {
		fmt.Println("error cashing out bet:", err)
		return mapErrorRepo(err)
	}

	return echo.NewHTTPError(http.StatusOK, "bet succesfully cashed out")

}

type balanceReq struct {
	Currency finance.Currency `query:"currency" validate:"required,oneof=USDT"`
}

func (h *TransactionHandler) GetBalance(c echo.Context) error {
	r := &balanceReq{}
	if err := utils.ParseAndValidateQueryParams(c, r, h.validate); err != nil {
		return err
	}

	user := utils.ContextGetUser(c)
	ctx := c.Request().Context()

	balanceCents, err := h.fm.GetUserBalanceLiabiliy(ctx, user.ID, r.Currency)
	if err != nil {
		return fmt.Errorf("failed to get user balance: %w", err)
	}

	return c.JSON(http.StatusOK, utils.Envelope{"balanceCents": balanceCents})
}
