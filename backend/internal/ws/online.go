package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

type OnlinePusher struct {
	ctx    context.Context
	rdb    *redis.Client
	logger *slog.Logger
}

const (
	onlinePushDelay = 10 * time.Second
)

func NewOnlinePusher(ctx context.Context, rdb *redis.Client, logger *slog.Logger) *OnlinePusher {
	return &OnlinePusher{
		ctx:    ctx,
		rdb:    rdb,
		logger: logger,
	}
}

func (op *OnlinePusher) Start() {
	go op.pushOnlineCount()
}

func (op *OnlinePusher) pushOnlineCount() {
	ticker := time.NewTicker(onlinePushDelay)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:

			wsBuf, err := op.GetOnlineWsMsg()
			if err != nil {
				op.logger.Error("failed to get online users count ws message", "error", err)
				continue
			}

			err = op.rdb.Publish(op.ctx, fmt.Sprintf("%s%s", RoomPubSubPrefix, OnlineRoom), wsBuf).Err()
			if err != nil {
				op.logger.Error("failed to publish online users count", "error", err)
			}
		case <-op.ctx.Done():
			op.logger.Info("shutting down online count pusher")
			return
		}
	}
}

func (op *OnlinePusher) GetOnlineWsMsg() ([]byte, error) {
	count, err := op.rdb.HLen(op.ctx, OnlineUsersKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve online users count from redis: %w", err)
	}

	update := OnlineUpdate{
		UsersOnlineCount: count,
	}

	data, err := json.Marshal(update)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal online users count: %w", err)
	}

	wsMsg := Message{
		Type:    OnlineRoom,
		Payload: data,
	}

	wsBuf, err := json.Marshal(wsMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshall websocket message: %w", err)
	}

	return wsBuf, nil
}
