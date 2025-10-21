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
	"strings"
	"time"

	"github.com/ericlagergren/decimal"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	betUpdateTimeout = 5 * time.Second
	latestBetsKey    = "bets:latest" // latest bets
	highBetsKey      = "bets:high"   // high bets
	nbBetsKept       = 10
)

var (
	highBetsTresholdCents = numeric.BigDecimal{Big: *decimal.New(100, 0)} // 100 USDT is a high bet
)

type BetLiveManager struct {
	ctx context.Context
	db  *pgxpool.Pool

	rdb    *redis.Client
	script *redis.Script

	logger *slog.Logger

	sem chan struct{} // Use a semaphore to limit concurrent executions
}

func NewBetLiveManager(ctx context.Context, rdb *redis.Client, db *pgxpool.Pool, logger *slog.Logger) *BetLiveManager {

	// Redis lua script for maintaining list with N elements
	const lua = `
		redis.call('LPUSH', KEYS[1], ARGV[1])
		redis.call('LTRIM', KEYS[1], 0, tonumber(ARGV[2]) - 1)
		return 1
    `

	return &BetLiveManager{
		ctx:    ctx,
		db:     db,
		rdb:    rdb,
		script: redis.NewScript(lua),
		logger: logger,

		sem: make(chan struct{}, 5), // max concurrent executions
	}
}

func (blm *BetLiveManager) Start() {
	go blm.start()
}

func (blm *BetLiveManager) start() {

	pubsub := blm.rdb.Subscribe(blm.ctx, ps.BetUpdateChannel)
	defer func() {
		if err := pubsub.Close(); err != nil {
			blm.logger.Error("failed to close pubsub", "error", err)
		}
	}()

	ch := pubsub.Channel()

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				blm.logger.Warn("pubsub channel closed")
				return
			}

			go func() {

				blm.sem <- struct{}{}
				defer func() { <-blm.sem }()

				err := blm.updateBetLive(msg.Payload)
				if err != nil {
					blm.logger.Error("could not update bet live", "error", err)
				}

			}()

		case <-blm.ctx.Done():
			blm.logger.Info("bet live manager shutting down", "reason", blm.ctx.Err())
			return
		}

	}

}

func (blm *BetLiveManager) updateBetLive(payload string) error {

	if payload == "" {
		return nil
	}

	u := &ps.BetUpdate{}
	err := utils.ReadJson(strings.NewReader(payload), u)
	if err != nil {
		return fmt.Errorf("failed to parse pubsub payload %q: %w", payload, err)
	}

	if u.BetID == uuid.Nil {
		return fmt.Errorf("invalid bet ID in payload: %s", payload)
	}

	updateCtx, cancel := context.WithTimeout(blm.ctx, betUpdateTimeout)
	defer cancel()

	// Retrieve bet state from DB

	bs, err := blm.retrieveBetStateDB(updateCtx, u.BetID)
	if err != nil {
		return fmt.Errorf("failed to retrieve bet state from db: %w", err)
	}

	// Latest
	var err1, err2 error

	err1 = blm.updateCacheAndPub(updateCtx, bs, latestBetsKey, ws.BetsLatestRoom)

	// Check if high bet
	if bs.Wager.Cmp(&highBetsTresholdCents.Big) >= 0 {
		err2 = blm.updateCacheAndPub(updateCtx, bs, highBetsKey, ws.BetsHighRoom)
	}

	return errors.Join(err1, err2)

}

func (blm *BetLiveManager) retrieveBetStateDB(ctx context.Context, betID uuid.UUID) (*BetState, error) {

	bs := &BetState{}

	query := `SELECT b.id, m.id, m.name, m.slug,
	o.id, o.name, 
	u.id, u.username, COALESCE(u.profile_image_key, ''),
	u.hidden, 
	b.total_price_paid AS wager,
	b.payout,
	b.avg_price,
	b.placed_at
	FROM bets b
	JOIN outcomes o ON o.id = b.outcome_id
	JOIN markets m ON m.id = o.market_id
	JOIN ledger_accounts la ON la.id = b.ledger_account_id
	JOIN users u ON u.id = la.user_id
	WHERE b.id = $1
	`

	user := &struct {
		ID              uuid.UUID `json:"id"`
		Username        string    `json:"username"`
		ProfileImageKey string    `json:"profileImageKey"`
	}{}

	var userHidden bool
	err := blm.db.QueryRow(ctx, query, betID).Scan(&bs.ID, &bs.MarketID, &bs.MarketName, &bs.MarketSlug,
		&bs.OutcomeID, &bs.OutcomeName,
		&user.ID, &user.Username, &user.ProfileImageKey,
		&userHidden,
		&bs.Wager, &bs.Payout, &bs.AvgPrice,
		&bs.PlacedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to query bet state: %w", err)
	}

	if !userHidden {
		bs.User = user
	}

	return bs, nil
}

func (blm *BetLiveManager) PrepopulateLatestBets(ctx context.Context) error {

	query := `SELECT b.id, m.id, m.name, m.slug,
		o.id, o.name, 
		u.id, u.username, COALESCE(u.profile_image_key, ''),
		u.hidden, 
		b.total_price_paid AS wager,
		b.payout,
		b.avg_price,
		b.placed_at
        FROM bets b
		JOIN outcomes o ON o.id = b.outcome_id
		JOIN markets m ON m.id = o.market_id
        JOIN ledger_accounts la ON la.id = b.ledger_account_id
        JOIN users u ON u.id = la.user_id
        ORDER BY b.placed_at DESC
        LIMIT $1`

	return blm.prepopulateBets(ctx, latestBetsKey, query, nbBetsKept)
}

func (blm *BetLiveManager) PrepopulateHighBets(ctx context.Context) error {

	query := `SELECT b.id, m.id, m.name, m.slug,
		o.id, o.name, 
		u.id, u.username, COALESCE(u.profile_image_key, ''),
		u.hidden, 
		b.total_price_paid AS wager,
		b.payout,
		b.avg_price,
		b.placed_at
        FROM bets b
		JOIN outcomes o ON o.id = b.outcome_id
		JOIN markets m ON m.id = o.market_id
        JOIN ledger_accounts la ON la.id = b.ledger_account_id
        JOIN users u ON u.id = la.user_id
		WHERE b.total_price_paid >= $1
        ORDER BY b.placed_at DESC
        LIMIT $2`

	return blm.prepopulateBets(ctx, highBetsKey, query, highBetsTresholdCents, nbBetsKept)
}

func (blm *BetLiveManager) prepopulateBets(ctx context.Context, cacheKey string, query string, args ...any) error {

	rows, err := blm.db.Query(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to query bets from db: %w", err)
	}

	defer rows.Close()

	var bets []*BetState

	for rows.Next() {
		bs := &BetState{}
		user := &struct {
			ID              uuid.UUID `json:"id"`
			Username        string    `json:"username"`
			ProfileImageKey string    `json:"profileImageKey"`
		}{}
		var userHidden bool

		err := rows.Scan(
			&bs.ID, &bs.MarketID, &bs.MarketName, &bs.MarketSlug,
			&bs.OutcomeID, &bs.OutcomeName,
			&user.ID, &user.Username, &user.ProfileImageKey,
			&userHidden,
			&bs.Wager, &bs.Payout, &bs.AvgPrice,
			&bs.PlacedAt)

		if err != nil {
			return fmt.Errorf("failed to scan bet: %w", err)
		}

		if !userHidden {
			bs.User = user
		}

		bets = append(bets, bs)
	}

	if rows.Err() != nil {
		return fmt.Errorf("error iterating bets rows: %w", rows.Err())
	}

	if len(bets) == 0 {
		return nil
	}

	// Clear existing cache and populate
	if err = blm.rdb.Del(ctx, cacheKey).Err(); err != nil {
		return fmt.Errorf("failed to delete current redis bets cache %w", err)
	}

	// Insert bets in reverse order (oldest first) so newest is at head
	for i := len(bets) - 1; i >= 0; i-- {
		data, err := json.Marshal(bets[i])
		if err != nil {
			return fmt.Errorf("failed to marshal bet state: %w", err)
		}
		blm.rdb.LPush(ctx, cacheKey, data)
	}

	return nil
}

func (blm *BetLiveManager) GetLatestBets(ctx context.Context) ([]*BetState, error) {

	vals, err := blm.rdb.LRange(ctx, latestBetsKey, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get latest bets from redis: %w", err)
	}

	bets := make([]*BetState, 0, len(vals))

	for _, v := range vals {
		b := &BetState{}
		err := utils.ReadJson(strings.NewReader(v), b)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshall bet json: %w", err)
		}
		bets = append(bets, b)
	}

	return bets, nil

}

func (blm *BetLiveManager) GetHighBets(ctx context.Context) ([]*BetState, error) {

	vals, err := blm.rdb.LRange(ctx, highBetsKey, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get latest bets from redis: %w", err)
	}

	bets := make([]*BetState, 0, len(vals))

	for _, v := range vals {
		b := &BetState{}
		err := utils.ReadJson(strings.NewReader(v), b)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshall bet json: %w", err)
		}
		bets = append(bets, b)
	}

	return bets, nil

}

func (blm *BetLiveManager) updateCacheAndPub(ctx context.Context, bs *BetState, listKey, wsRoom string) error {

	wsPayload := ws.BetUpdate{
		ID:          bs.ID,
		MarketID:    bs.MarketID,
		MarketName:  bs.MarketName,
		MarketSlug:  bs.MarketSlug,
		OutcomeID:   bs.OutcomeID,
		OutcomeName: bs.OutcomeName,
		Wager:       bs.Wager,
		Payout:      bs.Payout,
		AvgPrice:    bs.AvgPrice,
		PlacedAt:    bs.PlacedAt,
	}

	if bs.User != nil {
		wsPayload.User = &ws.UserState{
			ID:              bs.User.ID,
			Username:        bs.User.Username,
			ProfileImageKey: bs.User.ProfileImageKey,
		}
	}

	data, err := json.Marshal(wsPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal bet state: %w", err)
	}

	err = blm.script.Run(ctx, blm.rdb, []string{listKey}, data, nbBetsKept).Err()
	if err != nil {
		return fmt.Errorf("failed to execute redis lua script: %w", err)
	}

	wsMsg := ws.Message{
		Type:    wsRoom,
		Payload: data,
	}

	wsBuf, err := json.Marshal(wsMsg)
	if err != nil {
		return fmt.Errorf("failed to marshall websocket message: %w", err)
	}

	if err := blm.rdb.Publish(ctx, fmt.Sprintf("%s%s", ws.RoomPubSubPrefix, wsRoom), wsBuf).Err(); err != nil {
		return fmt.Errorf("failed to publish latest bet: %w", err)
	}

	return nil

}
