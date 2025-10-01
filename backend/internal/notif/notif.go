package notif

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"seer/internal/ps"
	"seer/internal/utils"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type NotificationType string

const (
	BetWon NotificationType = "bet_won"
)

const (
	MarketResolvedNotifTimeout = 30 * time.Second
)

type NotificationManager struct {
	ctx    context.Context
	logger *slog.Logger
	rdb    *redis.Client
	db     *pgxpool.Pool
}

type BetWonNotification struct {
	MarketID           uuid.UUID `json:"marketId"`
	MarketName         string    `json:"marketName"`
	BetID              uuid.UUID `json:"betId"`
	WinningOutcomeID   int64     `json:"winningOutcomeId"`
	WinningOutcomeName string    `json:"winningOutcomeName"`
	PricePaidCents     int64     `json:"pricePaidCents"`
	PayoutCents        int64     `json:"payoutCents"`
}

func (n BetWonNotification) Value() (driver.Value, error) {
	return json.Marshal(n)
}

// Make BetWonNotification struct implement the sql.Scanner interface. This method
// simply decodes a JSON-encoded value into the struct fields
func (n *BetWonNotification) Scan(value any) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &n)
}

func NewNotificationManager(ctx context.Context, logger *slog.Logger, rdb *redis.Client, db *pgxpool.Pool) *NotificationManager {
	return &NotificationManager{
		ctx:    ctx,
		logger: logger,
		rdb:    rdb,
		db:     db,
	}
}

type Notification struct {
	ID        int64
	UserID    uuid.UUID
	Type      NotificationType
	Data      any // JSON data
	IsRead    bool
	CreatedAt time.Time
}

func (nm *NotificationManager) Start() {
	go nm.start()
}

func (nm *NotificationManager) start() {

	pubsub := nm.rdb.Subscribe(nm.ctx, ps.MarketResolvedChannel)

	defer func() {
		if err := pubsub.Close(); err != nil {
			nm.logger.Error("failed to close pubsub", "error", err)
		}
	}()

	ch := pubsub.Channel()

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				nm.logger.Warn("pubsub channel closed")
				return
			}

			fmt.Println("received market resolved notification")

			switch msg.Channel {
			case ps.MarketResolvedChannel:
				if err := nm.NotifyMarketWinningUsers(msg.Payload); err != nil {
					nm.logger.Error("could not send notification to winning users market", "error", err)
				}
			}

		case <-nm.ctx.Done():
			nm.logger.Info("notification manager shutting down", "reason", nm.ctx.Err())
			return
		}

	}

}

func (nm *NotificationManager) NotifyMarketWinningUsers(payload string) error {

	if payload == "" {
		return nil
	}

	u := &ps.MarketResolvedUpdate{}
	err := utils.ReadJson(strings.NewReader(payload), u)
	if err != nil {
		return fmt.Errorf("failed to parse pubsub payload %q: %w", payload, err)
	}

	updateCtx, cancel := context.WithTimeout(nm.ctx, MarketResolvedNotifTimeout)
	defer cancel()

	// Retrieve winning bets and user info

	query := `
	SELECT u.id, m.id, m.name, b.id, o.id, o.name, b.total_price_paid_cents, b.payout_cents
	FROM bets b
	JOIN outcomes o ON b.outcome_id = o.id
	JOIN markets m ON o.market_id = m.id
	JOIN ledger_accounts la ON b.ledger_account_id = la.id
	JOIN users u ON la.user_id = u.id
	WHERE b.outcome_id = $1`

	rows, err := nm.db.Query(updateCtx, query, u.WinningOutcomeID)
	if err != nil {
		return fmt.Errorf("failed to query winning bets: %w", err)
	}

	defer rows.Close()

	var notifs []*Notification

	for rows.Next() {
		bn := &BetWonNotification{}
		var userID uuid.UUID
		err = rows.Scan(&userID, &bn.MarketID, &bn.MarketName, &bn.BetID, &bn.WinningOutcomeID, &bn.WinningOutcomeName, &bn.PricePaidCents, &bn.PayoutCents)
		if err != nil {
			return fmt.Errorf("failed to scan winning bet: %w", err)
		}

		n := &Notification{
			UserID: userID,
			Type:   BetWon,
			Data:   bn,
		}

		notifs = append(notifs, n)

	}

	if rows.Err() != nil {
		return fmt.Errorf("error iterating winning bets rows: %w", rows.Err())
	}

	// Insert notifications
	tx, err := nm.db.Begin(updateCtx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer tx.Rollback(updateCtx)

	query = `INSERT INTO notifications (user_id, type, data) VALUES ($1, $2, $3)`
	for _, n := range notifs {
		_, err = tx.Exec(updateCtx, query, n.UserID, n.Type, n.Data)
		if err != nil {
			return fmt.Errorf("failed to insert notification: %w", err)
		}
	}

	return tx.Commit(updateCtx)

}

func (nm *NotificationManager) GetUnreadNotifications(ctx context.Context, userID uuid.UUID) ([]*Notification, error) {
	query := `SELECT id, user_id, type, data, created_at
	FROM notifications
	WHERE user_id = $1 AND is_read = FALSE
	ORDER BY created_at DESC`

	rows, err := nm.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query unread notifications: %w", err)
	}
	defer rows.Close()

	var notifications []*Notification
	for rows.Next() {
		n := &Notification{}
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Data, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}
		notifications = append(notifications, n)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating notifications rows: %w", rows.Err())
	}

	return notifications, nil

}

func (nm *NotificationManager) MarkAsRead(ctx context.Context, userID uuid.UUID, IDs []int64) error {
	query := `UPDATE notifications SET is_read = TRUE 
	WHERE user_id = $1 
	AND id = ANY($2)
	AND is_read = FALSE`
	_, err := nm.db.Exec(ctx, query, userID, IDs)
	if err != nil {
		return fmt.Errorf("failed to mark notifications as read: %w", err)
	}
	return nil
}

func (nm *NotificationManager) CreateNotification(ctx context.Context, n *Notification) error {
	query := `INSERT INTO notifications (user_id, type, data) VALUES ($1, $2, $3) RETURNING id, created_at`
	err := nm.db.QueryRow(ctx, query, n.UserID, n.Type, n.Data).Scan(&n.ID, &n.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}
	return nil
}
