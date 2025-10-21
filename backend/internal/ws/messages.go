package ws

import (
	"seer/internal/numeric"
	"time"

	"github.com/google/uuid"
)

type UserState struct {
	ID              uuid.UUID `json:"id"`
	Username        string    `json:"username"`
	ProfileImageKey string    `json:"profileImageKey,omitempty"`
}

type BalanceUpdate struct {
	UserID   uuid.UUID           `json:"-"`
	Currency string              `json:"currency"`
	Balance  *numeric.BigDecimal `json:"balance"`
	Version  int64               `json:"version"`
}

type OutcomeUpdate struct {
	ID       int64               `json:"id"`
	Quantity *numeric.BigDecimal `json:"quantity"`
}

type MarketUpdate struct {
	ID       uuid.UUID       `json:"marketID"`
	Version  int64           `json:"marketVersion"`
	Outcomes []OutcomeUpdate `json:"outcomes"`
}

type ChatMessage struct {
	ID        uuid.UUID `json:"id"`
	ChatSlug  string    `json:"chatSlug"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	Type      string    `json:"type"` // "user" or "system"
	User      UserState `json:"user"`
}

type OnlineUpdate struct {
	UsersOnlineCount int64 `json:"usersOnlineCount"`
}

type BetUpdate struct {
	ID          uuid.UUID           `json:"id"`
	MarketID    uuid.UUID           `json:"marketID"`
	MarketName  string              `json:"marketName"`
	MarketSlug  string              `json:"marketSlug"`
	OutcomeID   int64               `json:"outcomeId"`
	OutcomeName string              `json:"outcomeName"`
	User        *UserState          `json:"user,omitempty"`
	Wager       *numeric.BigDecimal `json:"wager"`
	Payout      *numeric.BigDecimal `json:"payout"`
	AvgPrice    *numeric.BigDecimal `json:"avgPrice"`
	PlacedAt    time.Time           `json:"placedAt"`
}

type WSError struct {
	Error string `json:"error"`
}

const (
	ChatRoomPrefix    = "chat:"
	MarketsUpdateRoom = "markets_update"
	BetsLatestRoom    = "bets:latest"
	BetsHighRoom      = "bets:high"
	BalanceRoom       = "balance"
	OnlineRoom        = "online"
)
