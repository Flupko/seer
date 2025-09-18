package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"seer/internal/repos"
	"seer/internal/utils"
	"seer/internal/ws"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type ChatManager struct {
	rdb    *redis.Client
	script *redis.Script
	db     *pgxpool.Pool
}

const (
	cacheKeyRateUser = "rate:user:"
	cacheKeyList     = "chat:global"
	WSChatRoom       = "chat:global"
	maxLenChat       = 20
	bucketCapacity   = 10 // Burst of 10 messages
	ratePerMin       = 30 // 1 message every 2 seconds
	expireRateSec    = 10 * 60
)

type ChatMessage struct {
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"createdAt"`
	Type      string    `json:"type"` // "user" or "system"
	User      struct {
		ID       uuid.UUID `json:"id"`
		Username string    `json:"username"`
	}
}

func NewChatManager(rdb *redis.Client, db *pgxpool.Pool) *ChatManager {

	const lua = `
	local message = ARGV[1]
	local max_len = tonumber(ARGV[2])

	local now_time_ms = tonumber(ARGV[3])

	local capacity_bucket = tonumber(ARGV[4])
	local rate_per_min = tonumber(ARGV[5])
	local rate_per_ms = rate_per_min / 60000.0

	local rate_user_key = KEYS[1]
	local list_messages_key = KEYS[2]

	local expire_rate_sec = tonumber(ARGV[6])
	redis.call("EXPIRE", rate_user_key, expire_rate_sec)

	local last_refill_time = tonumber(redis.call("HGET", rate_user_key, "last_refill_time") or tostring(now_time_ms))
	local current_tokens = tonumber(redis.call("HGET", rate_user_key, "current_tokens") or tostring(capacity_bucket))

	local elapsed_time_ms = now_time_ms - last_refill_time
	local new_tokens = math.floor(elapsed_time_ms * rate_per_ms)

	local tokens = math.min(capacity_bucket, current_tokens + new_tokens)

	if tokens < 1 then
		return 0
	end

	tokens = tokens - 1

	redis.call("HSET", rate_user_key, "last_refill_time", now_time_ms, "current_tokens", tokens)

	-- add the new message
	redis.call("LPUSH", list_messages_key, message)
	redis.call("LTRIM", list_messages_key, 0, max_len - 1)

	return 1
	`
	return &ChatManager{
		rdb:    rdb,
		script: redis.NewScript(lua),
		db:     db,
	}
}

// SendMessage sends message and returns wether message was rate limited
func (cm *ChatManager) SendMessage(ctx context.Context, user *repos.MinimalUser, message string) (bool, error) {

	chatMsg := &ChatMessage{
		Message:   message,
		CreatedAt: time.Now().UTC(),
		Type:      "user",
	}

	chatMsg.User.ID = user.ID
	chatMsg.User.Username = user.Username

	data, err := json.Marshal(chatMsg)
	if err != nil {
		return false, fmt.Errorf("failed to marshal chat msg: %w", err)
	}

	// Redis first
	inserted, err := cm.script.Run(ctx, cm.rdb, []string{cacheKeyRateUser, cacheKeyList},
		data, maxLenChat,
		time.Now().UTC().UnixMilli(),
		bucketCapacity, ratePerMin,
		expireRateSec,
	).Bool()

	if err != nil {
		return false, fmt.Errorf("failed to execute redis lua script: %w", err)
	}

	if !inserted {
		return false, nil
	}

	wsMsg := ws.Message{
		Type:    WSChatRoom,
		Payload: data,
	}

	wsBuf, err := json.Marshal(wsMsg)
	if err != nil {
		return false, fmt.Errorf("failed to marshall websocket message: %w", err)
	}

	if err := cm.rdb.Publish(ctx, fmt.Sprintf("%s%s", ws.RoomPubSubPrefix, WSChatRoom), wsBuf).Err(); err != nil {
		return false, fmt.Errorf("failed to publish latest bet: %w", err)
	}

	return true, nil

}

func (cm *ChatManager) GetLastMessages(ctx context.Context) ([]*ChatMessage, error) {

	vals, err := cm.rdb.LRange(ctx, cacheKeyList, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get latest messages from redis: %w", err)
	}

	msgs := make([]*ChatMessage, 0, len(vals))

	for _, v := range vals {
		m := &ChatMessage{}
		err := utils.ReadJson(strings.NewReader(v), m)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshall messages json: %w", err)
		}
		msgs = append(msgs, m)
	}

	return msgs, nil

}
