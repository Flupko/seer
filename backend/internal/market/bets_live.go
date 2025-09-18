package market

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"seer/internal/utils"
	"seer/internal/ws"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	betUpdateChannel = "bet:update"
	betUpdateTimeout = 5 * time.Second
	latestBetsKey    = "bets:latest" // latest bets
	highBetsKey      = "bets:high"   // high bets
	highBetsTreshold = 1_00          // 100 USDT is a high bet
	nbBetsKept       = 10
)

type BetUpdate struct {
	BetID uuid.UUID `json:"betId"`
}

type BetLiveManager struct {
	ctx context.Context
	db  *pgxpool.Pool

	rdb    *redis.Client
	script *redis.Script

	logger *slog.Logger
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
	}
}

func (blm *BetLiveManager) Start() {
	go blm.start()
}

func (blm *BetLiveManager) start() {

	pubsub := blm.rdb.Subscribe(blm.ctx, betUpdateChannel)
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

			fmt.Println("received bet update")

			err := blm.updateBetLive(msg.Payload)
			if err != nil {
				blm.logger.Error("could not update bet live", "error", err)
			}

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

	u := &BetUpdate{}
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

	err1 = blm.updateCacheAndPub(updateCtx, bs, latestBetsKey, WsBetsLatestRoom)

	// Check if high bet
	if bs.WagerCents >= highBetsTreshold {
		err2 = blm.updateCacheAndPub(updateCtx, bs, highBetsKey, WsBetsHighRoom)
	}

	return errors.Join(err1, err2)

}

func (blm *BetLiveManager) retrieveBetStateDB(ctx context.Context, betID uuid.UUID) (*BetState, error) {

	bs := &BetState{}

	query := `SELECT b.id, m.id, m.name, 
	o.id, o.name, 
	u.username, u.hidden, 
	b.total_price_paid_cents AS wager_cents,
	b.payout_cents,
	b.purchase_time AS placed_at
	FROM bets b
	JOIN outcomes o ON o.id = b.outcome_id
	JOIN markets m ON m.id = o.market_id
	JOIN ledger_accounts la ON la.id = b.ledger_account_id
	JOIN users u ON u.id = la.user_id
	WHERE b.id = $1
	`

	var payoutCents int64
	var username string
	var userHidden bool
	err := blm.db.QueryRow(ctx, query, betID).Scan(&bs.ID, &bs.MarketID, &bs.MarketName,
		&bs.OutcomeID, &bs.OutcomeName,
		&username, &userHidden,
		&bs.WagerCents, &payoutCents,
		&bs.PlacedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to query bet state: %w", err)
	}

	if !userHidden {
		bs.Username = &username
	}

	odds, err := ComputeOddDecPPH(bs.WagerCents, payoutCents)

	if err != nil {
		return nil, fmt.Errorf("failed to compute odd for bet : %w", err)
	}

	bs.OddsDecimalPPH = odds

	return bs, nil
}

func (blm *BetLiveManager) PrepopulateLatestBets(ctx context.Context) error {

	query := `SELECT b.id, m.id, m.name, 
        o.id, o.name, 
        u.username, u.hidden, 
        b.total_price_paid_cents AS wager_cents,
        b.payout_cents,
		b.purchase_time AS placed_at
        FROM bets b
		JOIN outcomes o ON o.id = b.outcome_id
		JOIN markets m ON m.id = o.market_id
        JOIN ledger_accounts la ON la.id = b.ledger_account_id
        JOIN users u ON u.id = la.user_id
        ORDER BY b.purchase_time DESC
        LIMIT $1`

	return blm.prepopulateBets(ctx, latestBetsKey, query, nbBetsKept)
}

func (blm *BetLiveManager) PrepopulateHighBets(ctx context.Context) error {

	query := `SELECT b.id, m.id, m.name, 
        o.id, o.name, 
        u.username, u.hidden, 
        b.total_price_paid_cents AS wager_cents,
        b.payout_cents,
		b.purchase_time AS placed_at
        FROM bets b
		JOIN outcomes o ON o.id = b.outcome_id
		JOIN markets m ON m.id = o.market_id
        JOIN ledger_accounts la ON la.id = b.ledger_account_id
        JOIN users u ON u.id = la.user_id
		WHERE b.total_price_paid_cents >= $1
        ORDER BY b.purchase_time DESC
        LIMIT $2`

	return blm.prepopulateBets(ctx, highBetsKey, query, highBetsTreshold, nbBetsKept)
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
		var payoutCents int64
		var username string
		var userHidden bool

		err := rows.Scan(
			&bs.ID, &bs.MarketID, &bs.MarketName,
			&bs.OutcomeID, &bs.OutcomeName,
			&username, &userHidden,
			&bs.WagerCents, &payoutCents,
			&bs.PlacedAt)

		if err != nil {
			return fmt.Errorf("failed to scan bet: %w", err)
		}

		if !userHidden {
			bs.Username = &username
		}

		odds, err := ComputeOddDecPPH(bs.WagerCents, payoutCents)
		if err != nil {
			blm.logger.Warn("failed to compute odds for bet")
			continue
		}

		bs.OddsDecimalPPH = odds
		bets = append(bets, bs)
	}

	if len(bets) == 0 {
		return nil
	}

	// Clear existing cache and populate
	if err = blm.rdb.Del(ctx, cacheKey).Err(); err != nil {
		return fmt.Errorf("failed to delete current redis cache")
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

	data, err := json.Marshal(bs)
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
