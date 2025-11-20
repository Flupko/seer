package market

import (
	"database/sql"
	"fmt"
	"seer/internal/finance"
	"seer/internal/numeric"
	"seer/internal/repos"
	"seer/internal/utils/meta"
	"time"

	"github.com/google/uuid"
)

type MarketStatus string

const (
	StatusDraft     MarketStatus = "draft"
	StatusOpened    MarketStatus = "opened"
	StatusPaused    MarketStatus = "paused"
	StatusResolved  MarketStatus = "resolved"
	StatusPending   MarketStatus = "pending" // When a market is
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
	IsBinary    bool
	Currency    finance.Currency
	Status      MarketStatus
	ImgKey      string
	Slug        string

	HouseLedgerAccountID uuid.UUID
	Q0Seeding            *numeric.BigDecimal
	Alpha                *numeric.BigDecimal
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

type OutcomeView struct {
	Outcome
	PriceCharts []PriceChart
}

type MarketResolution struct {
	ID               int64
	MarketID         uuid.UUID
	WinningOutcomeID int64
	CreatedAt        time.Time
}

type MarketView struct {
	Market
	Resolution *MarketResolution
	Categories []Category
	Outcomes   []OutcomeView
}

type MarketSearchResult struct {
	Markets  []*MarketView
	Metadata *meta.Metadata
}

type BetSide string

const (
	SideYes BetSide = "y" // buying shares for the outcome
	SideNo  BetSide = "n" // buying shares against the outcome (= all other outcomes)
)

type Bet struct {
	ID               uuid.UUID
	LedgerAccountID  uuid.UUID
	LedgerTransferID uuid.UUID
	OutcomeID        int64
	Side             BetSide
	Payout           *numeric.BigDecimal
	TotalPricePaid   *numeric.BigDecimal
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
	Cashout      *BetCashout
	User         repos.UserView
	Status       BetStatus
	MarketID     uuid.UUID
	MarketName   string
	MarketImgKey string
	OutcomeID    int64
	Currency     finance.Currency
	OutcomeName  string
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
	SortTrending:   "m.volume_24h DESC",
	SortNewest:     "m.created_at DESC",
	SortVolume:     "m.volume DESC",
	SortEndingSoon: "CASE WHEN m.close_time IS NULL THEN 'infinity'::timestamp ELSE m.close_time END ASC",
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
	BetStatusActive    BetStatus = "active"
	BetStatusWon       BetStatus = "won"
	BetStatusLost      BetStatus = "lost"
	BetStatusCashedOut BetStatus = "cashedOut"
	BetStatusResolved  BetStatus = "resolved" // won OR lost OR cashedOut
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
	SortEvent    SortBet = "event"
)

var betSortSafeMap = map[SortBet]string{
	SortPlacedAt: "placed_at",
	SortWager:    "total_price_paid",
	SortPayout:   "payout",
	SortEvent:    "event_at",
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
	Volume        *numeric.BigDecimal   `json:"volume"`
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

type PricesTimeframe string

const (
	Prices24h PricesTimeframe = "24h"
	Prices7d  PricesTimeframe = "7d"
	Prices30d PricesTimeframe = "30d"
	PricesAll PricesTimeframe = "all"
)

var pricesTimeframeSafeMap = map[PricesTimeframe]struct {
	table    string
	duration string
	interval time.Duration
}{
	Prices24h: {"outcome_price_5m", "'5m'", time.Hour * 24},
	Prices7d:  {"outcome_price_1h", "'1h'", time.Hour * 24 * 7},
	Prices30d: {"outcome_price_4h", "'4h'", time.Hour * 24 * 30},
	PricesAll: {"outcome_price_24h", "'24h'", time.Hour * 24 * 365 * 10},
}

type PriceChart struct {
	Timeframe PricesTimeframe       `json:"timeframe"`
	Prices    []PriceChartDataPoint `json:"prices"`
}

type PriceChartDataPoint struct {
	Timestamp int64               `json:"timestamp"`
	Date      time.Time           `json:"date"`
	Price     *numeric.BigDecimal `json:"price"`
}
