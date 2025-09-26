package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"seer/internal/repos"
	"seer/internal/utils"
	"seer/internal/ws"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var (
	ErrChatRoomNotFound = errors.New("chat room not found")
)

type ChatManager struct {
	rdb    *redis.Client
	script *redis.Script
	db     *pgxpool.Pool
	logger *slog.Logger
}

const (
	cacheKeyPrefixRate     = "rate:user:"
	cacheKeyPrefixChat     = "chat:"
	cacheKeyPrefixChatList = "chat:list:"
	WSChatRoomPrefix       = "chat:"
	maxLenChat             = 20
	bucketCapacity         = 10 // Burst of 10 messages
	ratePerMin             = 30 // 1 message every 2 seconds
	expireRateSec          = 10 * 60
)

type Chat struct {
	ID    uuid.UUID
	Label string
	Slug  string
}

type ChatMessageView struct {
	ID        uuid.UUID `json:"id"`
	ChatID    uuid.UUID `json:"chatId"`
	ChatSlug  string    `json:"chatSlug"`
	Content   string    `json:"message"`
	CreatedAt time.Time `json:"createdAt"`
	Type      string    `json:"type"` // "user" or "system"
	User      struct {
		ID       uuid.UUID `json:"id"`
		Username string    `json:"username"`
	}
}

func NewChatManager(rdb *redis.Client, db *pgxpool.Pool, logger *slog.Logger) *ChatManager {

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
		logger: logger,
	}
}

func (cm *ChatManager) PrepopulateChatRooms(ctx context.Context) error {
	// First retrieve all chat rooms

	rows, err := cm.db.Query(ctx, `SELECT id, label, slug FROM chat_rooms`)
	if err != nil {
		return fmt.Errorf("failed to query chat rooms: %w", err)
	}

	defer rows.Close()

	var chats []*Chat

	for rows.Next() {
		c := &Chat{}
		if err = rows.Scan(&c.ID, &c.Label, &c.Slug); err != nil {
			return fmt.Errorf("failed to scan chat: %w", err)
		}
		chats = append(chats, c)
	}

	if rows.Err() != nil {
		return fmt.Errorf("error iterating chats rows: %w", rows.Err())
	}

	for _, c := range chats {
		if err = cm.prepopulateChatRoom(ctx, c); err != nil {
			return fmt.Errorf("failed to prepopulate chat room %s: %w", c, err)
		}
	}

	return nil

}

func (cm *ChatManager) prepopulateChatRoom(ctx context.Context, c *Chat) error {

	query := `SELECT cm.id, cm.chat_id, cr.slug, cm.content, cm.created_at, u.id, u.username
	FROM chat_messages cm
	JOIN users u ON cm.user_id = u.id
	JOIN chat_rooms cr ON cm.chat_id = cr.id
	WHERE cm.chat_id = $1
	ORDER BY created_at DESC
	LIMIT $2`

	rows, err := cm.db.Query(ctx, query, c.ID, maxLenChat)
	if err != nil {
		return fmt.Errorf("failed to query chat messages: %w", err)
	}

	defer rows.Close()

	var messages []*ChatMessageView

	for rows.Next() {
		m := &ChatMessageView{}
		if err = rows.Scan(&m.ID, &m.ChatID, &m.ChatSlug, &m.Content, &m.CreatedAt, &m.User.ID, &m.User.Username); err != nil {
			return fmt.Errorf("failed to scan message: %w", err)
		}
		m.Type = "user"
		messages = append(messages, m)
	}

	// Clear existing cache and prepulate
	cacheKeyChatList := buildCacheKeyChatList(c.Slug)

	if err = cm.rdb.Del(ctx, cacheKeyChatList).Err(); err != nil {
		return fmt.Errorf("failed to delete current redis chat cache: %w", err)
	}

	// Insert bets in reverse order (oldest first) so newest is at head
	for i := len(messages) - 1; i >= 0; i-- {
		data, err := json.Marshal(messages[i])
		if err != nil {
			return fmt.Errorf("failed to marshal bet state: %w", err)
		}
		cm.rdb.LPush(ctx, cacheKeyChatList, data)
	}

	// Set cache for chat existence
	cacheKeyChat := buildCacheKeyChat(c.Slug)
	cm.rdb.Set(ctx, cacheKeyChat, c.ID.String(), 0)

	return nil

}

// SendMessage sends message and returns wether message was rate limited
func (cm *ChatManager) SendMessage(ctx context.Context, user *repos.MinimalUser, content string, chatSlug string) (bool, error) {

	cacheKeyChat := buildCacheKeyChat(chatSlug)
	chatIDStr, err := cm.rdb.Get(ctx, cacheKeyChat).Result()
	if err != nil {
		switch {
		case errors.Is(err, redis.Nil):
			return false, ErrChatRoomNotFound
		default:
			return false, fmt.Errorf("failed to check chat existence")
		}
	}

	chatID, err := uuid.Parse(chatIDStr)
	if err != nil {
		return false, fmt.Errorf("failed to parse chat id from redis: %w", err)
	}

	cacheKeyChatList := buildCacheKeyChatList(chatSlug)

	chatMsg := &ChatMessageView{
		ID:        uuid.New(),
		ChatID:    chatID,
		ChatSlug:  chatSlug,
		Content:   content,
		CreatedAt: time.Now().UTC(),
		Type:      "user",
	}

	chatMsg.User.ID = user.ID
	chatMsg.User.Username = user.Username

	data, err := json.Marshal(chatMsg)
	if err != nil {
		return false, fmt.Errorf("failed to marshal chat msg: %w", err)
	}

	// Redis cache
	cacheKeyRateUser := buildCacheKeyRateUser(user.ID)
	inserted, err := cm.script.Run(ctx, cm.rdb, []string{cacheKeyRateUser, cacheKeyChatList},
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

	wsChatRoom := BuildWSChatRoom(chatSlug)

	wsMsg := ws.Message{
		Type:    wsChatRoom,
		Payload: data,
	}

	wsBuf, err := json.Marshal(wsMsg)
	if err != nil {
		return false, fmt.Errorf("failed to marshall websocket message: %w", err)
	}

	if err := cm.rdb.Publish(ctx, fmt.Sprintf("%s%s", ws.RoomPubSubPrefix, wsChatRoom), wsBuf).Err(); err != nil {
		return false, fmt.Errorf("failed to publish latest bet: %w", err)
	}

	go func() {
		persistCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := cm.persistMessageDB(persistCtx, chatMsg); err != nil {
			cm.logger.Error("failed to persist chat message", "errror", err)
		}
	}()

	return true, nil

}

func (cm *ChatManager) persistMessageDB(ctx context.Context, m *ChatMessageView) error {

	query := `INSERT INTO chat_messages(id, user_id, chat_id, content, created_at)
	VALUES($1, $2, $3, $4, $5)
	`

	if _, err := cm.db.Exec(ctx, query, m.ID, m.User.ID, m.ChatID, m.Content, m.CreatedAt); err != nil {
		fmt.Println("error inserting chat message:", err)
		return fmt.Errorf("failed to insert chat message: %w", err)
	}

	return nil
}

func (cm *ChatManager) GetLastMessagesChat(ctx context.Context, chatSlug string) ([]*ChatMessageView, error) {

	cacheKeyChat := buildCacheKeyChat(chatSlug)
	_, err := cm.rdb.Get(ctx, cacheKeyChat).Result()
	if err != nil {
		switch {
		case errors.Is(err, redis.Nil):
			return nil, ErrChatRoomNotFound
		default:
			return nil, fmt.Errorf("failed to check chat existence")
		}
	}

	cacheKeyChatList := buildCacheKeyChatList(chatSlug)

	vals, err := cm.rdb.LRange(ctx, cacheKeyChatList, 0, -1).Result()

	if err != nil {
		return nil, fmt.Errorf("failed to get latest messages from redis: %w", err)
	}

	msgs := make([]*ChatMessageView, 0, len(vals))

	for _, v := range vals {
		m := &ChatMessageView{}
		err := utils.ReadJson(strings.NewReader(v), m)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshall messages json: %w", err)
		}
		msgs = append(msgs, m)
	}

	return msgs, nil

}

func buildCacheKeyChatList(chatSlug string) string {
	return fmt.Sprintf("%s%s", cacheKeyPrefixChatList, chatSlug)
}

func buildCacheKeyChat(chatSlug string) string {
	return fmt.Sprintf("%s%s", cacheKeyPrefixChat, chatSlug)
}

func buildCacheKeyRateUser(userID uuid.UUID) string {
	return fmt.Sprintf("%s%s", cacheKeyPrefixRate, userID)
}

func BuildWSChatRoom(chatSlug string) string {
	return fmt.Sprintf("%s%s", WSChatRoomPrefix, chatSlug)
}
