package market

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"seer/internal/numeric"
	"seer/internal/ps"
	"seer/internal/utils"
	"seer/internal/ws"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type StateManager struct {
	rdb    *redis.Client
	db     *pgxpool.Pool
	logger *slog.Logger
	script *redis.Script
	ctx    context.Context

	sem chan struct{} // Use a semaphore to limit concurrent executions
}

const (
	marketCacheKeyPrefix = "market_state:"
	MarketupdateTimeout  = 5 * time.Second
	cacheTTL             = 30 // 30 seconds
)

func NewStateManager(ctx context.Context, rdb *redis.Client, db *pgxpool.Pool, logger *slog.Logger) *StateManager {

	// Redis lua script which updates the cache only if the version on which the prices were computed
	// is higher than the currently stored version
	const lua = `
        local current_version = tonumber(redis.call('HGET', KEYS[1], 'version'))
        local new_version = tonumber(ARGV[1])
		local ttl = tonumber(ARGV[3])
        
        if not current_version or current_version < new_version then
          redis.call('HSET', KEYS[1], 'version', ARGV[1], 'payload', ARGV[2])
		  redis.call('EXPIRE', KEYS[1], ttl)
          return 1
        else
          return 0
        end
    `

	return &StateManager{
		rdb:    rdb,
		db:     db,
		logger: logger,
		script: redis.NewScript(lua),
		ctx:    ctx,

		sem: make(chan struct{}, 20), // max concurrent executions
	}
}

func (sm *StateManager) Start() {
	go sm.start()
}

func (sm *StateManager) start() {

	pubsub := sm.rdb.Subscribe(sm.ctx, ps.MarketUpdateChannel)
	defer func() {
		if err := pubsub.Close(); err != nil {
			sm.logger.Error("failed to close pubsub", "error", err)
		}
	}()

	ch := pubsub.Channel()

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				sm.logger.Warn("pubsub channel closed")
				return
			}

			go func() {

				sm.sem <- struct{}{}
				defer func() { <-sm.sem }()

				err := sm.updateMarketState(msg.Payload)
				if err != nil {
					sm.logger.Error("could not update market prices", "error", err)
				}

			}()

		case <-sm.ctx.Done():
			sm.logger.Info("market state manager shutting down", "reason", sm.ctx.Err())
			return
		}

	}
}

func (sm *StateManager) updateMarketState(payload string) error {

	if payload == "" {
		return errors.New("payload cannot be empty")
	}

	u := &ps.MarketUpdate{}
	err := utils.ReadJson(strings.NewReader(payload), u)
	if err != nil {
		return fmt.Errorf("failed to parse pubsub payload %q: %w", payload, err)
	}

	if u.MarketID == uuid.Nil {
		return fmt.Errorf("invalid market ID in payload: %s", payload)
	}

	updateCtx, cancel := context.WithTimeout(sm.ctx, MarketupdateTimeout)
	defer cancel()

	ms, err := sm.retrieveMarketStateDB(updateCtx, u.MarketID)
	if err != nil {
		return fmt.Errorf("failed to retrieve market state from db: %w", err)
	}

	changed, err := sm.setRedisCacheMarketState(updateCtx, ms)

	// Don't return on error, can function without redis
	if err != nil {
		sm.logger.Error("failed to set market state in redis cache", "error", err)
	}

	// Stale version, abort
	if err == nil && changed == 0 {
		return nil
	}

	wsPayload := ws.MarketUpdate{
		ID:      ms.ID,
		Version: ms.Version,
	}

	for i := range len(ms.QVec) {
		wsPayload.Outcomes = append(wsPayload.Outcomes,
			ws.OutcomeUpdate{
				ID:       ms.OutcomeIDs[i],
				Quantity: ms.QVec[i],
			})
	}

	wsMsg, err := utils.WsMessage(ws.MarketsUpdateRoom, wsPayload)
	if err != nil {
		return fmt.Errorf("failed to build websocket message: %w", err)
	}

	wsBuf, err := json.Marshal(wsMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal websocket message: %w", err)
	}

	fmt.Println("Publishing market update for market:", u.MarketID)

	if err := sm.rdb.Publish(updateCtx, fmt.Sprintf("%s%s", ws.RoomPubSubPrefix, ws.MarketsUpdateRoom), string(wsBuf)).Err(); err != nil {
		fmt.Println("Failed to publish market update:", err)
		return fmt.Errorf("publish prices: %w", err)
	}
	return nil
}

// Returns (gainCents, avgPrice, err)
func (sm *StateManager) GetQuoteForBet(ctx context.Context, betAmount *numeric.BigDecimal, marketID uuid.UUID, outcomeID int64) (*numeric.BigDecimal, *numeric.BigDecimal, error) {

	ms, err := sm.GetMarketState(ctx, marketID)

	if err != nil {
		return nil, nil, err
	}

	idx := slices.Index(ms.OutcomeIDs, outcomeID)

	if idx == -1 {
		return nil, nil, ErrOutcomeNotFound
	}

	_, gain, _, avgPrice, err := PossibleGainFeePriceForBuy(ms.QVec, idx, ms.Alpha, ms.Fee, betAmount, ms.CapPrice)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to compute gain: %w", err)
	}

	return gain, avgPrice, nil
}

func (sm *StateManager) GetMarketState(ctx context.Context, marketID uuid.UUID) (*MarketState, error) {

	if marketID == uuid.Nil {
		return nil, errors.New("market ID cannot be nil")
	}

	// Try to hit the cache first
	cacheKey := buildCacheKeyRedis(marketID)

	cache, err := sm.rdb.HGet(ctx, cacheKey, "payload").Result()
	// If there's an error (= cache not set, or redis has issues) simply fallback to the DB
	if err == nil {
		m := &MarketState{}
		err := utils.ReadJson(strings.NewReader(cache), m)
		if err == nil {
			return m, nil
		}
		sm.logger.Warn("cache data corrupted, falling back to DB", "marketID", marketID, "error", err)
	}

	// DB Fallback
	ms, err := sm.retrieveMarketStateDB(ctx, marketID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve market state from db: %w", err)
	}

	// Set redis cache in a separate goroutine
	go func() {
		cacheCtx, cancel := context.WithTimeout(sm.ctx, 5*time.Second)
		defer cancel()
		_, err := sm.setRedisCacheMarketState(cacheCtx, ms)
		if err != nil {
			sm.logger.Error("failed to set market state in redis cache", "error", err)
		}
	}()

	// Return MarketState
	return ms, nil
}

func (sm *StateManager) retrieveMarketStateDB(ctx context.Context, marketID uuid.UUID) (*MarketState, error) {
	ms := &MarketState{}

	query := `
	SELECT m.id, m.version, m.alpha, m.fee, m.cap_price,
	array_agg(o.quantity ORDER BY o.id) AS q_vec,
	array_agg(o.id ORDER BY o.id) AS outcome_ids
	FROM markets m
	JOIN outcomes o ON o.market_id = m.id
	WHERE m.id = $1 AND status = 'opened' AND (close_time IS NULL OR close_time > NOW())
	GROUP BY m.id, m.version, m.alpha, m.fee`

	if err := sm.db.QueryRow(ctx, query, marketID).Scan(&ms.ID, &ms.Version, &ms.Alpha, &ms.Fee, &ms.CapPrice, &ms.QVec, &ms.OutcomeIDs); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMarketNotFound
		}
		return nil, fmt.Errorf("failed to query market's current state: %w", err)
	}

	if len(ms.QVec) != len(ms.OutcomeIDs) || len(ms.QVec) == 0 {
		return nil, errors.New("inconsistent outcomes for market")
	}

	prices, err := PricesBD(ms.QVec, ms.Alpha, ms.Fee)
	if err != nil {
		return nil, fmt.Errorf("failed to compute prices for market %s: %w", ms.ID, err)
	}

	ms.Prices = prices

	return ms, nil
}

func (sm *StateManager) setRedisCacheMarketState(ctx context.Context, ms *MarketState) (int, error) {

	data, err := json.Marshal(ms)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal market state: %w", err)
	}

	cacheKey := buildCacheKeyRedis(ms.ID)
	changed, err := sm.script.Run(ctx, sm.rdb, []string{cacheKey}, ms.Version, data, cacheTTL).Int()
	if err != nil {
		return 0, fmt.Errorf("failed to execute redis lua script: %w", err)
	}

	return changed, nil

}

func (sm *StateManager) GetValidMarkets(ctx context.Context, marketIDs []uuid.UUID) ([]uuid.UUID, error) {

	var validMarkets []uuid.UUID
	query := `SELECT id FROM markets WHERE id = ANY($1)`

	rows, err := sm.db.Query(ctx, query, marketIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var marketID uuid.UUID
		err = rows.Scan(&marketID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan: %w", err)
		}
		validMarkets = append(validMarkets, marketID)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating markets rows: %w", rows.Err())
	}

	return validMarkets, nil
}

func buildCacheKeyRedis(marketID uuid.UUID) string {
	return fmt.Sprintf("%s%s", marketCacheKeyPrefix, marketID)
}
