package market

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type MarketStatus string

const (
	StatusDraft     MarketStatus = "draft"
	StatusOpened    MarketStatus = "opened"
	StatusPaused    MarketStatus = "paused"
	StatusSettling  MarketStatus = "settling"
	StatusResolved  MarketStatus = "resolved"
	StatusCancelled MarketStatus = "cancelled"
)

type MarketOutcomeSort string

const (
	SortPrice    MarketOutcomeSort = "price"
	SortPosition MarketOutcomeSort = "position"
)

type Market struct {
	ID          uuid.UUID
	Name        string
	Description string
	Status      MarketStatus

	HouseLedgerAccountID uuid.UUID
	Q0Seeding            int64
	AlphaPPM             int64
	FeePPM               int64
	VolumeCents          int64

	CreatedAt time.Time
	CloseTime sql.NullTime

	OutcomeSort MarketOutcomeSort

	Version int64
}

type Outcome struct {
	ID          int64
	MarketID    int64
	Name        string
	Quantity    int64
	VolumeCents int64
	Position    int64
}

type OutcomeView struct {
	Outcome
	Active bool `json:"active"`
	OddPPH int64
}

type MarketView struct {
	Market
	Categories []Category
	Outcomes   []OutcomeView
}

type Bet struct {
	ID                  uuid.UUID
	LedgerAccountID     uuid.UUID
	OutcomeID           int64
	PayoutCents         int64
	TotalPricePaidCents int64
	FeePaidCents        int64
	FeePPM              int64
	PurchaseTime        time.Time
	IdempotencyKey      string
}

type BetView struct {
	Bet
	UserID      uuid.UUID
	Status      BetStatus
	MarketID    uuid.UUID
	MarketName  string
	OutcomeID   int64
	OutcomeName string
}

type Category struct {
	ID    int64
	Slug  string
	Label string
}

type SortMarket string

const (
	SortHot        SortMarket = "hot"
	SortVolume     SortMarket = "volume"
	SortNewest     SortMarket = "newest"
	SortEndingSoon SortMarket = "ending_soon"
)

var sortSafeMap = map[SortMarket]string{
	SortHot:        "volume_24h DESC",
	SortNewest:     "created_at DESC",
	SortVolume:     "volume_cents DESC",
	SortEndingSoon: "CASE WHEN close_time IS NULL THEN 'infinity'::timestamp ELSE close_time END ASC",
}

type SearchQuery struct {
	Query *string

	CategoryID *int64
	Status     MarketStatus

	Sort SortMarket

	Page     int64
	PageSize int64
}

func (sq *SearchQuery) Limit() int64 {
	return sq.PageSize
}

func (sq *SearchQuery) Offset() int64 {
	return (sq.Page - 1) * sq.PageSize
}

func (sq *SearchQuery) GetOrderBy() string {
	orderBy, ok := sortSafeMap[sq.Sort]
	if !ok {
		panic("unvalid sort option")
	}
	return orderBy
}

// bet search query

type BetStatus string

const (
	BetStatusActive   BetStatus = "active"
	BetStatusWon      BetStatus = "won"
	BetStatusLost     BetStatus = "lost"
	BetStatusResolved BetStatus = "resolved" // won OR lost
	BetStatusRefunded BetStatus = "refunded"
)

type BetSearchQuery struct {
	UserID   *uuid.UUID
	MarketID *uuid.UUID
	Status   *BetStatus
	PageSize int64
	Page     int64
}

func (sq *BetSearchQuery) Limit() int64 {
	return sq.PageSize
}

func (sq *BetSearchQuery) Offset() int64 {
	return (sq.Page - 1) * sq.PageSize
}

// HOT Types (for cache and WS push)

type MarketState struct {
	ID            uuid.UUID `json:"marketId"`
	Version       int64     `json:"marketVersion"`
	AlphaPPM      int64     `json:"alphaPPM"`
	FeePPM        int64     `json:"feePPM"`
	OutcomeIDs    []int64   `json:"outcomeIDs"`
	QVec          []int64   `json:"qVec"`
	OddsPPH       []int64   `json:"oddsPPH"`
	OutcomeActive []bool    `json:"outcomesActive"`
	UpdatedAtUnix int64     `json:"updateAtUnivex"`
}

type WSPayloadOutcomeUpdate struct {
	ID     int64 `json:"id"`
	OddPPH int64 `json:"oddPPH"`
	Active bool  `json:"active"`
}

type WSPayloadMarketUpdate struct {
	ID       uuid.UUID `json:"marketID"`
	Version  int64     `json:"marketVersion"`
	Outcomes []WSPayloadOutcomeUpdate
}

type BetUpdateType string

const (
	Latest     BetUpdateType = "latest"
	HighRoller BetUpdateType = "highRoller"
)

type BetState struct {
	ID             uuid.UUID `json:"id"`
	MarketID       uuid.UUID `json:"marketID"`
	MarketName     string    `json:"marketName"`
	OutcomeID      int64     `json:"outcomeId"`
	OutcomeName    string    `json:"outcomeName"`
	Username       *string   `json:"username"`
	WagerCents     int64     `json:"wageCents"`
	OddsDecimalPPH int64     `json:"oddsDecimalPPH"`
	PlacedAt       time.Time `json:"placedAt"`
}

const WsMarketRoomPrefix = "market:"
const WsBetsLatestRoom = "bets:latest"
const WsBetsHighRoom = "bets:high"
