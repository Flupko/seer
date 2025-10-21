package market

import (
	"database/sql"
	"fmt"
	"seer/internal/finance"
	"seer/internal/numeric"
	"seer/internal/utils/meta"
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
	Currency    finance.Currency
	Status      MarketStatus
	ImgKey      string
	Slug        string

	HouseLedgerAccountID uuid.UUID
	Q0Seeding            *numeric.BigDecimal
	Alpha                *numeric.BigDecimal
	Fee                  *numeric.BigDecimal
	Volume               *numeric.BigDecimal
	CapPrice             *numeric.BigDecimal

	CreatedAt time.Time
	CloseTime sql.NullTime

	OutcomeSort MarketOutcomeSort

	Version int64
}

type Outcome struct {
	ID       int64
	MarketID int64
	Name     string
	Quantity *numeric.BigDecimal
	Volume   *numeric.BigDecimal
	Position int64
}

type MarketView struct {
	Market
	Categories []Category
	Outcomes   []Outcome
}

type MarketSearchResult struct {
	Markets  []*MarketView
	Metadata *meta.Metadata
}

type Bet struct {
	ID               uuid.UUID
	LedgerAccountID  uuid.UUID
	LedgerTransferID uuid.UUID
	OutcomeID        int64
	Payout           *numeric.BigDecimal
	TotalPricePaid   *numeric.BigDecimal
	FeeApplied       *numeric.BigDecimal
	FeePaid          *numeric.BigDecimal
	AvgPrice         *numeric.BigDecimal
	PlacedAt         time.Time
	IdempotencyKey   string
}

type BetCashout struct {
	ID               uuid.UUID
	BetID            uuid.UUID
	LedgerTransferID uuid.UUID
	Payout           *numeric.BigDecimal
	PlacedAt         time.Time
	IdempotencyKey   string
}

type BetView struct {
	Bet
	User struct {
		ID       uuid.UUID
		Username string
		Hidden   bool
	}
	Status      BetStatus
	MarketID    uuid.UUID
	MarketName  string
	OutcomeID   int64
	Currency    finance.Currency
	OutcomeName string
}

type Category struct {
	ID       int64
	Slug     string
	Label    string
	Position int64
	IconUrl  string
	Featured bool
}

type SortMarket string

const (
	SortTrending   SortMarket = "trending"
	SortVolume     SortMarket = "volume"
	SortNewest     SortMarket = "newest"
	SortEndingSoon SortMarket = "endingSoon"
)

var marketSortSafeMap = map[SortMarket]string{
	SortTrending:   "volume_24h DESC",
	SortNewest:     "created_at DESC",
	SortVolume:     "volume DESC",
	SortEndingSoon: "CASE WHEN close_time IS NULL THEN 'infinity'::timestamp ELSE close_time END ASC",
}

type SearchQuery struct {
	Query *string

	CategorySlug *string
	Status       MarketStatus

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
	orderBy, ok := marketSortSafeMap[sq.Sort]
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

	MinPrice *numeric.BigDecimal
	MaxPrice *numeric.BigDecimal

	FromTime *time.Time
	ToTime   *time.Time

	PageSize int64
	Page     int64

	Sort    SortBet
	SortDir string
}

type SortBet string

const (
	SortPlacedAt SortBet = "placedAt"
	SortWager    SortBet = "wager"
	SortPayout   SortBet = "payout"
)

var betSortSafeMap = map[SortBet]string{
	SortPlacedAt: "placed_at",
	SortWager:    "total_price_paid",
	SortPayout:   "payout",
}

func (sq *BetSearchQuery) GetOrderBy() string {
	sortCol, ok := betSortSafeMap[sq.Sort]
	if !ok {
		panic("unvalid sort col")
	}

	if sq.SortDir != "asc" && sq.SortDir != "desc" {
		panic("unvalid sort dir")
	}

	return fmt.Sprintf("%s %s", sortCol, sq.SortDir)

}

func (sq *BetSearchQuery) Limit() int64 {
	return sq.PageSize
}

func (sq *BetSearchQuery) Offset() int64 {
	return (sq.Page - 1) * sq.PageSize
}

// HOT Types (for cache and WS push)
type MarketState struct {
	ID            uuid.UUID             `json:"marketId"`
	Version       int64                 `json:"marketVersion"`
	Alpha         *numeric.BigDecimal   `json:"alpha"`
	Fee           *numeric.BigDecimal   `json:"fee"`
	CapPrice      *numeric.BigDecimal   `json:"capPrice"`
	OutcomeIDs    []int64               `json:"outcomeIDs"`
	QVec          []*numeric.BigDecimal `json:"qVec"`
	Prices        []*numeric.BigDecimal `json:"price"`
	UpdatedAtUnix int64                 `json:"updateAtUnivex"`
}

type BetState struct {
	ID          uuid.UUID `json:"id"`
	MarketID    uuid.UUID `json:"marketID"`
	MarketName  string    `json:"marketName"`
	MarketSlug  string    `json:"marketSlug"`
	OutcomeID   int64     `json:"outcomeId"`
	OutcomeName string    `json:"outcomeName"`
	User        *struct {
		ID              uuid.UUID `json:"id"`
		Username        string    `json:"username"`
		ProfileImageKey string    `json:"profileImageKey"`
	} `json:"user,omitempty"`
	Wager    *numeric.BigDecimal `json:"wager"`
	Payout   *numeric.BigDecimal `json:"payout"`
	AvgPrice *numeric.BigDecimal `json:"avgPrice"`
	PlacedAt time.Time           `json:"placedAt"`
}
