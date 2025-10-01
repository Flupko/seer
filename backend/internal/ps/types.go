package ps

import "github.com/google/uuid"

type BetUpdate struct {
	BetID uuid.UUID `json:"betId"`
}

type MarketUpdate struct {
	MarketID uuid.UUID `json:"marketId"`
}

type MarketResolvedUpdate struct {
	WinningOutcomeID int64 `json:"winningOutcomeID"`
}

type BalanceUpdate struct {
	LedgerAccountID uuid.UUID `json:"ledgerAccountId"`
}

const (
	BalanceUpdateChannel  = "balance:update"
	BetUpdateChannel      = "bet:update"
	MarketResolvedChannel = "market:resolved"
	MarketUpdateChannel   = "market:update"
)
