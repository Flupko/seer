package balance

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"seer/internal/ps"
	"seer/internal/utils"
	"seer/internal/ws"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	WsBalanceRoom = "balance"
)

const (
	marketBalancePushTimeout = 20 * time.Second
	userBalancePushTimeout   = 5 * time.Second
)

type WsPayloadBalanceUpdate struct {
	UserID   uuid.UUID `json:"-"`
	Currency string    `json:"currency"`
	Balance  int64     `json:"balance"`
	Version  int64     `json:"version"`
}

type BalancePusher struct {
	ctx    context.Context
	rdb    *redis.Client
	db     *pgxpool.Pool
	logger *slog.Logger
	sem    chan struct{} // Use a semaphore to limit concurrent executions
}

func NewBalancePusher(ctx context.Context, rdb *redis.Client, db *pgxpool.Pool, logger *slog.Logger) *BalancePusher {
	return &BalancePusher{
		ctx:    ctx,
		rdb:    rdb,
		db:     db,
		logger: logger,
		sem:    make(chan struct{}, 5), // max concurrent executions
	}
}

func (bp *BalancePusher) Start() {
	go bp.pushBalances()
}

func (bp *BalancePusher) pushBalances() {

	pubsub := bp.rdb.Subscribe(bp.ctx, ps.MarketResolvedChannel, ps.BalanceUpdateChannel)
	defer func() {
		if err := pubsub.Close(); err != nil {
			bp.logger.Error("failed to close pubsub", "error", err)
		}
	}()

	ch := pubsub.Channel()

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				bp.logger.Warn("pubsub channel closed")
				return
			}

			switch msg.Channel {
			case ps.MarketResolvedChannel:

				go func() {

					bp.sem <- struct{}{}
					defer func() { <-bp.sem }()

					err := bp.pushBalancesForResolvedMarket(msg.Payload)
					if err != nil {
						bp.logger.Error("failed to push balances for resolved market", "error", err, "marketId", msg.Payload)
					}
				}()

			case ps.BalanceUpdateChannel:

				go func() {

					bp.sem <- struct{}{}
					defer func() { <-bp.sem }()

					err := bp.pushBalanceUpdate(msg.Payload)
					if err != nil {
						bp.logger.Error("failed to push balance update", "error", err)
					}

				}()

			}

		case <-bp.ctx.Done():
			bp.logger.Info("bet live manager shutting down", "reason", bp.ctx.Err())
			return
		}

	}
}

func (bp *BalancePusher) pushBalancesForResolvedMarket(payload string) error {
	if payload == "" {
		return nil
	}

	u := &ps.MarketResolvedUpdate{}
	err := utils.ReadJson(strings.NewReader(payload), u)
	if err != nil {
		return fmt.Errorf("failed to parse pubsub payload %q: %w", payload, err)
	}

	pushCtx, cancel := context.WithTimeout(bp.ctx, marketBalancePushTimeout)
	defer cancel()

	// Retrieve all user IDs curently connected through websockets

	bp.logger.Info("pushing balances for resolved market", "winningOutcomeId", u.WinningOutcomeID)

	userOnlineIDs, err := bp.rdb.HKeys(pushCtx, ws.LoggedInUsersKey).Result()
	if err != nil {
		return fmt.Errorf("failed to retrieve online users from redis: %w", err)
	}

	query := `SELECT DISTINCT la.user_id, la.balance, la.currency, la.version
	FROM bets b
	JOIN ledger_accounts la ON la.id = b.ledger_account_id
	WHERE b.outcome_id = $1 AND la.user_id = ANY($2)`

	rows, err := bp.db.Query(pushCtx, query, u.WinningOutcomeID, userOnlineIDs)
	if err != nil {
		return fmt.Errorf("failed to query user balances for resolved market: %w", err)
	}

	defer rows.Close()

	balanceUpdates := make([]*WsPayloadBalanceUpdate, 0)

	for rows.Next() {
		bu := &WsPayloadBalanceUpdate{}
		err = rows.Scan(&bu.UserID, &bu.Balance, &bu.Currency, &bu.Version)
		if err != nil {
			return fmt.Errorf("failed to scan balance update: %w", err)
		}
		balanceUpdates = append(balanceUpdates, bu)
	}

	if rows.Err() != nil {
		return fmt.Errorf("error iterating balance update rows: %w", rows.Err())
	}

	for _, bu := range balanceUpdates {

		err = bp.pushBalanceToUserWs(pushCtx, bu)
		if err != nil {
			bp.logger.Error("failed to push balance to user", "error", err, "userId", bu.UserID, "currency", bu.Currency)
		}
	}

	return nil

}

func (bp *BalancePusher) pushBalanceUpdate(payload string) error {
	if payload == "" {
		return nil
	}

	u := &ps.BalanceUpdate{}
	err := utils.ReadJson(strings.NewReader(payload), u)
	if err != nil {
		return fmt.Errorf("failed to parse pubsub payload %q: %w", payload, err)
	}

	if u.LedgerAccountID == uuid.Nil {
		return fmt.Errorf("invalid ledger account ID in payload: %s", payload)
	}

	pushCtx, cancel := context.WithTimeout(bp.ctx, userBalancePushTimeout)
	defer cancel()

	query := `SELECT la.user_id, la.balance, la.currency, la.version
	FROM ledger_accounts la
	WHERE la.id = $1`

	bu := &WsPayloadBalanceUpdate{}

	err = bp.db.QueryRow(pushCtx, query, u.LedgerAccountID).Scan(&bu.UserID, &bu.Balance, &bu.Currency, &bu.Version)
	if err != nil {
		return fmt.Errorf("failed to query user balance for ledger account %s: %w", u.LedgerAccountID, err)
	}

	err = bp.pushBalanceToUserWs(pushCtx, bu)
	if err != nil {
		return fmt.Errorf("failed to push balance to user %s: %w", bu.UserID, err)
	}

	return nil

}

func (bp *BalancePusher) pushBalanceToUserWs(ctx context.Context, bu *WsPayloadBalanceUpdate) error {

	data, err := json.Marshal(bu)
	if err != nil {
		return fmt.Errorf("failed to marshal bet state: %w", err)
	}

	wsMsg := ws.Message{
		Type:    WsBalanceRoom,
		Payload: data,
	}

	wsBuf, err := json.Marshal(wsMsg)
	if err != nil {
		return fmt.Errorf("failed to marshall websocket message: %w", err)
	}

	if err := bp.rdb.Publish(ctx, fmt.Sprintf("%suser:%s", ws.RoomPubSubPrefix, bu.UserID.String()), wsBuf).Err(); err != nil {
		return fmt.Errorf("failed to publish latest bet: %w", err)
	}

	return nil
}
