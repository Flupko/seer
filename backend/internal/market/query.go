package market

import (
	"context"
	"encoding/json"
	"fmt"
	"seer/internal/utils"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type QueryManager struct {
	db  *pgxpool.Pool
	rdb *redis.Client
}

func NewQueryManager(db *pgxpool.Pool, rdb *redis.Client) *QueryManager {
	return &QueryManager{
		db:  db,
		rdb: rdb,
	}
}

func (qm *QueryManager) GetAllCategories(ctx context.Context) ([]Category, error) {

	rows, err := qm.db.Query(ctx, `SELECT id, slug, label FROM categories`)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve categories: %w", err)
	}

	categories := []Category{}

	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Slug, &c.Label); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, c)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return categories, nil
}

func (qm *QueryManager) SearchMarkets(ctx context.Context, sq *SearchQuery, skipCache bool) ([]*MarketView, error) {

	cacheKey := qm.buildCacheKey(sq)

	if !skipCache {

		cached, err := qm.rdb.Get(ctx, cacheKey).Result()

		// Cache hit. If error either cache miss or redis issue
		if err == nil {
			var res []*MarketView
			if err2 := utils.ReadJson(strings.NewReader(cached), &res); nil == err2 {
				return res, nil
			}
		}
	}

	// Query markets

	var tsQuery string
	if sq.Query != nil {
		tsQuery = strings.Join(strings.Fields(*sq.Query), ":* & ") + ":*"
	}

	query := fmt.Sprintf(`SELECT 
	m.id, m.name, m.description, m.status, 
	m.house_ledger_account_id, m.q0_seeding, m.alpha_ppm, m.fee_ppm, m.volume_cents,
	m.created_at, m.close_time, 
	m.outcome_sort,
	m.version
	FROM markets m
	WHERE ($1 = '' OR to_tsvector('simple', m.name || ' ' || m.description) @@ to_tsquery('simple', $1))
	AND ($2::BIGINT IS NULL OR EXISTS( SELECT 1 FROM categories_market cm WHERE cm.market_id = m.id AND cm.category_id = $2) )
	AND ($3::TEXT IS NULL OR m.status = $3)
	ORDER BY %s
	LIMIT $4 OFFSET $5
	`, sq.GetOrderBy())

	rows, err := qm.db.Query(ctx, query, tsQuery, sq.CategoryID, sq.Status, sq.Limit(), sq.Offset())
	if err != nil {
		return nil, fmt.Errorf("failed to query rows markets: %w", err)
	}
	defer rows.Close()

	markets := []*MarketView{}

	for rows.Next() {
		m := &MarketView{}

		err = rows.Scan(
			&m.ID, &m.Name, &m.Description, &m.Status,
			&m.HouseLedgerAccountID, &m.Q0Seeding, &m.AlphaPPM, &m.FeePPM, &m.VolumeCents,
			&m.CreatedAt, &m.CloseTime, &m.OutcomeSort, &m.Version,
		)

		if err != nil {
			return nil, fmt.Errorf("failed scanning market: %w", err)
		}

		markets = append(markets, m)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	if len(markets) == 0 {
		return []*MarketView{}, nil
	}

	// Build market IDs slice
	marketIDs := make([]uuid.UUID, 0, len(markets))
	for _, m := range markets {
		marketIDs = append(marketIDs, m.ID)
	}

	// Query outcomes for all markets
	outcomesByMarket, err := qm.getOutcomesForMarkets(ctx, marketIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get outcomes: %w", err)
	}

	// Attach slice of outcomes to appropriate market
	// Compute odds for outcomes, and sort them
	for _, m := range markets {

		m.Outcomes = outcomesByMarket[m.ID]

		qVec := make([]int64, len(m.Outcomes))
		for i, o := range m.Outcomes {
			qVec[i] = o.Quantity
		}

		oddsPPH, active, err := OddsPPH(qVec, m.AlphaPPM, m.FeePPM)
		if err != nil {
			return nil, fmt.Errorf("failed to compute odds for market %s", m.ID)
		}

		for i := range m.Outcomes {
			m.Outcomes[i].Active = m.Status == StatusOpened && active[i]
			m.Outcomes[i].OddPPH = oddsPPH[i]
		}

		qm.sortOutcomes(m.Outcomes, m.OutcomeSort)

	}

	// Query categories
	categoriesByMarket, err := qm.getCategoriesForMarkets(ctx, marketIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get outcomes for markets: %w", err)
	}

	// Attach categories to markets
	for _, m := range markets {
		m.Categories = categoriesByMarket[m.ID]
	}

	// Set Redis cache
	if data, err := json.Marshal(markets); nil == err {
		// Set cache, ignore error
		qm.rdb.SetEx(ctx, cacheKey, data, 5*time.Minute)
	}

	return markets, nil

}

func (qm *QueryManager) getOutcomesForMarkets(ctx context.Context, marketIDs []uuid.UUID) (map[uuid.UUID][]OutcomeView, error) {
	query := `
        SELECT o.market_id, o.id, o.name, o.position, o.quantity, o.volume_cents
        FROM outcomes o
        WHERE o.market_id = ANY($1)
        ORDER BY o.id
    `

	rows, err := qm.db.Query(ctx, query, marketIDs)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	outcomesByMarket := make(map[uuid.UUID][]OutcomeView)

	for rows.Next() {
		var o OutcomeView
		var marketID uuid.UUID
		if err := rows.Scan(&marketID, &o.ID, &o.Name, &o.Position, &o.Quantity, &o.VolumeCents); err != nil {
			return nil, fmt.Errorf("failed to scan outcome: %w", err)
		}
		outcomesByMarket[marketID] = append(outcomesByMarket[marketID], o)
	}

	return outcomesByMarket, rows.Err()
}

func (qm *QueryManager) getCategoriesForMarkets(ctx context.Context, marketIDs []uuid.UUID) (map[uuid.UUID][]Category, error) {

	query := `
        SELECT cm.market_id, c.id, c.slug, c.label
        FROM categories_market cm
        JOIN categories c ON cm.category_id = c.id
        WHERE cm.market_id = ANY($1)
        ORDER BY c.position
    `

	rows, err := qm.db.Query(ctx, query, marketIDs)
	if err != nil {
		return nil, err
	}

	categoriesByMarket := make(map[uuid.UUID][]Category)

	defer rows.Close()

	for rows.Next() {
		var c Category
		var marketID uuid.UUID

		err = rows.Scan(&marketID, &c.ID, &c.Slug, &c.Label)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}

		categoriesByMarket[marketID] = append(categoriesByMarket[marketID], c)

	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return categoriesByMarket, nil

}

func (qm *QueryManager) sortOutcomes(outcomes []OutcomeView, outcomeSort MarketOutcomeSort) {

	switch outcomeSort {
	case SortPrice:
		sort.Slice(outcomes, func(i, j int) bool {
			return outcomes[i].OddPPH < outcomes[j].OddPPH
		})
	case SortPosition:
		sort.Slice(outcomes, func(i, j int) bool {
			return outcomes[i].Position < outcomes[j].Position
		})
	// position by default
	default:
		sort.Slice(outcomes, func(i, j int) bool {
			return outcomes[i].Position < outcomes[j].Position
		})
	}
}

func (qm *QueryManager) buildCacheKey(sq *SearchQuery) string {
	queryStr := ""
	if sq.Query != nil {
		queryStr = *sq.Query
	}
	return fmt.Sprintf("market_search:%s:%v:%s:%s:%d:%d",
		queryStr, sq.CategoryID, sq.Status, sq.Sort, sq.Page, sq.PageSize)
}
