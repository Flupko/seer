package market

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"seer/internal/ps"
	"seer/internal/utils"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type StatManager struct {
	rdb    *redis.Client
	db     *pgxpool.Pool
	logger *slog.Logger
	ctx    context.Context
}

func NewStatManager(ctx context.Context, rdb *redis.Client, db *pgxpool.Pool, logger *slog.Logger) *StatManager {
	return &StatManager{
		ctx:    ctx,
		rdb:    rdb,
		db:     db,
		logger: logger,
	}
}

func (sm *StatManager) Start(ctx context.Context) {
	go sm.start(ctx)
}

func (sm *StatManager) start(ctx context.Context) {

	// Track bet updates to update prices
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

			err := sm.addMarketPriceHistory(msg.Payload)
			if err != nil {
				sm.logger.Error("could not add market price history", "error", err)
			}

		case <-sm.ctx.Done():
			sm.logger.Info("market state manager shutting down", "reason", sm.ctx.Err())
			return
		}

	}

}

func (sm *StatManager) addMarketPriceHistory(payload string) error {

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

	addCtx, cancel := context.WithTimeout(sm.ctx, MarketupdateTimeout)
	defer cancel()

	// For each outcome of the market, recompute corresponding price, store it as ppm
	// No serializable level, as readed market state doesn't need to be exactly precise
	// No business level code logic depends on this computed price, purely indicative for admins to manage the market
	tx, err := sm.db.Begin(addCtx)

	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}

	defer tx.Rollback(addCtx)

	var alphaPPM, feePPM int64
	var qVec, outcomeIds []int64

	query := `
	SELECT m.alpha_ppm, m.fee_ppm, 
	array_agg(o.quantity ORDER BY o.id) AS q_vec,
	array_agg(o.id ORDER BY o.id) AS outcome_ids
	FROM markets m
	JOIN outcomes o ON o.market_id = m.id
	WHERE m.id = $1
	GROUP BY m.id, m.alpha_ppm, m.fee_ppm`

	if err := tx.QueryRow(addCtx, query, u.MarketID).Scan(&alphaPPM, &feePPM, &qVec, &outcomeIds); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMarketNotFound
		}
		return fmt.Errorf("failed to query market's current state: %w", err)
	}

	if len(qVec) != len(outcomeIds) || len(qVec) == 0 {
		return errors.New("inconsistent outcomes for market")
	}

	pricesPPM, _, err := PricesPPM(qVec, alphaPPM, feePPM)
	if err != nil {
		return fmt.Errorf("failed to compute prices for market %s: %w", u.MarketID, err)
	}

	query = `INSERT INTO outcome_price_history(outcome_id, price_ppm) VALUES ($1, $2)`
	for i := range len(pricesPPM) {
		if _, err := tx.Exec(addCtx, query, outcomeIds[i], pricesPPM[i]); err != nil {
			return fmt.Errorf("failed to insert outcome price history: %w", err)
		}
	}

	return tx.Commit(addCtx)

}

type TimeInterval string

const (
	IntervalHour  TimeInterval = "hour"
	IntervalDay   TimeInterval = "day"
	IntervalWeek  TimeInterval = "week"
	IntervalMonth TimeInterval = "month"
)

type OutcomeBoughtStat struct {
	OutcomeID int64 `json:"outcomeId"`
	BetsCents int64 `json:"betsCents"`
	Quantity  int64 `json:"quantity"`
	From      time.Time
	To        time.Time
	Interval  TimeInterval
}

func (sm *StatManager) GetOutcomeBoughtInterval(ctx context.Context, outcomeID int64, from, to time.Time) ([]OutcomeBoughtStat, error) {

	return nil, nil

}
