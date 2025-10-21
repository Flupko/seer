package market

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"seer/internal/numeric"
	"seer/internal/utils"
	"seer/internal/utils/meta"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

func (qm *QueryManager) GetAllFeaturedCategories(ctx context.Context) ([]Category, error) {

	query := `SELECT id, slug, label, iconUrl 
	FROM categories
	WHERE featured = TRUE
	ORDER BY position`

	rows, err := qm.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve categories: %w", err)
	}

	categories := []Category{}

	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Slug, &c.Label, &c.IconUrl); err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}
		categories = append(categories, c)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating categories rows: %w", rows.Err())
	}

	return categories, nil
}

func (qm *QueryManager) SearchMarkets(ctx context.Context, sq *SearchQuery, skipCache bool) ([]*MarketView, *meta.Metadata, error) {

	cacheKey := qm.buildCacheKey(sq)

	if !skipCache {

		cached, err := qm.rdb.Get(ctx, cacheKey).Result()

		fmt.Println("err checking market search cache:", err)

		// Cache hit. If error either cache miss or redis issue
		if err == nil {
			fmt.Println("Market search cache hit:", cacheKey)
			var searchRes MarketSearchResult
			if err2 := utils.ReadJson(strings.NewReader(cached), &searchRes); nil == err2 {
				return searchRes.Markets, searchRes.Metadata, nil
			}
		}
	}

	// Query markets

	var tsQuery string
	if sq.Query != nil {
		tsQuery = strings.Join(strings.Fields(*sq.Query), ":* & ") + ":*"
	}

	query := fmt.Sprintf(`SELECT count(*) OVER() AS total_count,
	m.id, m.name, m.description, m.img_key, m.slug,
	m.status,
	m.house_ledger_account_id, m.q0_seeding, m.alpha, m.fee, m.cap_price,
	m.volume,
	m.created_at, m.close_time, 
	m.outcome_sort,
	m.version
	FROM markets m
	WHERE ($1 = '' OR to_tsvector('simple', m.name || ' ' || m.description) @@ to_tsquery('simple', $1))
	AND ($2::TEXT IS NULL OR EXISTS(SELECT 1 FROM categories_market cm JOIN categories c ON cm.category_id = c.id WHERE cm.market_id = m.id AND c.slug = $2))
	AND ($3::TEXT IS NULL OR m.status = $3)
	ORDER BY %s
	LIMIT $4 OFFSET $5
	`, sq.GetOrderBy())

	rows, err := qm.db.Query(ctx, query, tsQuery, sq.CategorySlug, sq.Status, sq.Limit(), sq.Offset())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query rows markets: %w", err)
	}
	defer rows.Close()

	markets := []*MarketView{}
	var totalCount int64

	for rows.Next() {
		m := &MarketView{}

		err = rows.Scan(
			&totalCount,
			&m.ID, &m.Name, &m.Description, &m.ImgKey, &m.Slug,
			&m.Status,
			&m.HouseLedgerAccountID, &m.Q0Seeding, &m.Alpha, &m.Fee, &m.CapPrice,
			&m.Volume,
			&m.CreatedAt, &m.CloseTime, &m.OutcomeSort, &m.Version,
		)

		if err != nil {
			return nil, nil, fmt.Errorf("failed scanning market: %w", err)
		}

		markets = append(markets, m)
	}

	if rows.Err() != nil {
		return nil, nil, fmt.Errorf("error iterating markets rows: %w", rows.Err())
	}

	if len(markets) == 0 {
		return []*MarketView{}, meta.CalculateMetadata(0, sq.Page, sq.PageSize), nil
	}

	// Build market IDs slice
	marketIDs := make([]uuid.UUID, 0, len(markets))
	for _, m := range markets {
		marketIDs = append(marketIDs, m.ID)
	}

	// Query outcomes for all markets
	outcomesByMarket, err := qm.getOutcomesForMarkets(ctx, marketIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get outcomes: %w", err)
	}

	// Attach slice of outcomes to appropriate market
	// Compute pries for outcomes, and sort them
	for _, m := range markets {

		m.Outcomes = outcomesByMarket[m.ID]

		qVec := make([]*numeric.BigDecimal, len(m.Outcomes))
		for i, o := range m.Outcomes {
			qVec[i] = o.Quantity
		}

	}

	// Query categories
	categoriesByMarket, err := qm.getCategoriesForMarkets(ctx, marketIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get outcomes for markets: %w", err)
	}

	// Attach categories to markets
	for _, m := range markets {
		m.Categories = categoriesByMarket[m.ID]
	}

	metadata := meta.CalculateMetadata(totalCount, sq.Page, sq.PageSize)

	// Set Redis cache
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if data, err := json.Marshal(MarketSearchResult{Markets: markets, Metadata: metadata}); nil == err {
			fmt.Println("Setting market search cache:", cacheKey)
			// Set cache, ignore error
			qm.rdb.SetEx(cacheCtx, cacheKey, data, 5*time.Minute)
		}
	}()

	return markets, metadata, nil

}

func (qm *QueryManager) getOutcomesForMarkets(ctx context.Context, marketIDs []uuid.UUID) (map[uuid.UUID][]Outcome, error) {
	query := `
        SELECT o.market_id, o.id, o.name, o.position
        FROM outcomes o
        WHERE o.market_id = ANY($1)
        ORDER BY o.id
    `

	rows, err := qm.db.Query(ctx, query, marketIDs)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	outcomesByMarket := make(map[uuid.UUID][]Outcome)

	for rows.Next() {
		var o Outcome
		var marketID uuid.UUID
		if err := rows.Scan(&marketID, &o.ID, &o.Name, &o.Position); err != nil {
			return nil, fmt.Errorf("failed to scan outcome: %w", err)
		}
		outcomesByMarket[marketID] = append(outcomesByMarket[marketID], o)
	}

	return outcomesByMarket, rows.Err()
}

func (qm *QueryManager) getCategoriesForMarkets(ctx context.Context, marketIDs []uuid.UUID) (map[uuid.UUID][]Category, error) {

	query := `
        SELECT cm.market_id, c.id, c.slug, c.label, c.iconUrl
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

		err = rows.Scan(&marketID, &c.ID, &c.Slug, &c.Label, &c.IconUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category: %w", err)
		}

		categoriesByMarket[marketID] = append(categoriesByMarket[marketID], c)

	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating categories rows: %w", rows.Err())
	}

	return categoriesByMarket, nil

}

func (qm *QueryManager) GetMarketByID(ctx context.Context, marketID uuid.UUID) (*MarketView, error) {
	query := `SELECT m.id, m.name, m.description, m.img_key, slug,
	m.status,
	m.house_ledger_account_id, m.q0_seeding, m.alpha, m.fee, m.cap_price,
	m.volume,
	m.created_at, m.close_time, 
	m.outcome_sort,
	m.version
	FROM markets m
	WHERE m.id = $1`

	m := &MarketView{}
	err := qm.db.QueryRow(ctx, query, marketID).Scan(
		m.ID, &m.Name, &m.Description, &m.ImgKey, &m.Slug,
		&m.Status,
		&m.HouseLedgerAccountID, &m.Q0Seeding, &m.Alpha, &m.Fee, &m.CapPrice,
		&m.Volume,
		&m.CreatedAt, &m.CloseTime, &m.OutcomeSort, &m.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, ErrMarketNotFound
		default:
			return nil, fmt.Errorf("failed to query market by id: %w", err)
		}
	}

	// Get outcomes
	outcomesByMarket, err := qm.getOutcomesForMarkets(ctx, []uuid.UUID{marketID})
	if err != nil {
		return nil, fmt.Errorf("failed to get outcomes: %w", err)
	}
	m.Outcomes = outcomesByMarket[m.ID]

	// Get categories
	categoriesByMarket, err := qm.getCategoriesForMarkets(ctx, []uuid.UUID{marketID})
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}
	m.Categories = categoriesByMarket[m.ID]

	return m, nil
}

func (qm *QueryManager) buildCacheKey(sq *SearchQuery) string {
	queryStr := ""
	if sq.Query != nil {
		queryStr = *sq.Query
	}

	categoryStr := ""
	if sq.CategorySlug != nil {
		categoryStr = *sq.CategorySlug
	}

	return fmt.Sprintf("market_search:%s:%s:%s:%s:%d:%d",
		queryStr, categoryStr, sq.Status, sq.Sort, sq.Page, sq.PageSize)
}
