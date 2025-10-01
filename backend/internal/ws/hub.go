package ws

import (
	"context"
	"fmt"
	"log/slog"
	"seer/internal/repos"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Client can connect
// Join a room
// Leave a room
//

const RoomPubSubPrefix = "ws:room:"

const (
	onlineUsersHeartbeat = 10 * time.Second
	LoggedInUsersKey     = "ws:loogeded_in_online_users"
	OnlineUsersKey       = "ws:all_online_users"
	onlineUserTTL        = 20 * time.Second
)

type subscriptionReq struct {
	client *Client
	roomID string
}

type broadcastReq struct {
	roomID  string
	payload []byte
}

type Hub struct {
	logger *slog.Logger

	rdb *redis.Client

	ctx    context.Context
	cancel context.CancelFunc

	register   chan *Client
	unregister chan *Client

	subscribe   chan subscriptionReq
	unsubscribe chan subscriptionReq
	broadcast   chan broadcastReq

	rooms map[string]*roomState

	clients        map[*Client]bool
	clientRooms    map[*Client]map[*roomState]bool
	userRegistered map[uuid.UUID]int32

	wg sync.WaitGroup
}

type roomState struct {
	id          string
	clients     map[*Client]bool
	cancel      context.CancelFunc
	redisPubSub *redis.PubSub
}

func NewHub(parent context.Context, logger *slog.Logger, rdb *redis.Client) *Hub {
	ctx, cancel := context.WithCancel(parent)
	h := &Hub{
		logger: logger,

		rdb:    rdb,
		ctx:    ctx,
		cancel: cancel,

		register:    make(chan *Client, 256),
		unregister:  make(chan *Client, 256),
		subscribe:   make(chan subscriptionReq, 1024),
		unsubscribe: make(chan subscriptionReq, 1024),
		broadcast:   make(chan broadcastReq, 4096),

		clients:        make(map[*Client]bool),
		rooms:          make(map[string]*roomState),
		clientRooms:    make(map[*Client]map[*roomState]bool),
		userRegistered: make(map[uuid.UUID]int32),
	}
	go h.start()
	return h
}

func (h *Hub) Close() {
	h.cancel()
}

func (h *Hub) start() {

	ticker := time.NewTicker(onlineUsersHeartbeat)
	for {
		select {
		case <-h.ctx.Done():
			h.cleanupOnShutdown()
			h.stop()
			return
		case c := <-h.register:
			h.handleRegister(c)
		case c := <-h.unregister:
			h.handleUnregister(c)

		case s := <-h.subscribe:
			h.handleSubscribe(s)

		case s := <-h.unsubscribe:
			h.handleUnsubscribe(s)

		case br := <-h.broadcast:
			h.handleBroadcast(br)
		case <-ticker.C:
			h.updateOnlineUsers()
		}
	}
}

func (h *Hub) cleanupOnShutdown() {
	for c := range h.clients {
		delete(h.clientRooms, c)
		delete(h.clients, c)
		close(c.send)
	}
}

func (h *Hub) Register(c *Client) {

	if c == nil {
		return
	}

	select {
	case h.register <- c:
	case <-h.ctx.Done():
	}

}

func (h *Hub) handleRegister(c *Client) {

	if c == nil {
		return
	}

	if _, ok := h.clients[c]; ok {
		return
	}

	h.clients[c] = true
	h.clientRooms[c] = make(map[*roomState]bool)

	if c.User != repos.AnonymousUser {
		h.userRegistered[c.User.ID]++
	}

	h.handleSubscribe(subscriptionReq{client: c, roomID: fmt.Sprintf("user:%s", c.User.ID)})

}

func (h *Hub) Unregister(c *Client) {

	if c == nil {
		return
	}

	select {
	case h.unregister <- c:
	case <-h.ctx.Done():
	}
}

func (h *Hub) handleUnregister(c *Client) {

	if c == nil {
		return
	}

	// If client is already unregistered, skip
	if _, ok := h.clients[c]; !ok {
		return
	}

	if c.User != repos.AnonymousUser {
		h.userRegistered[c.User.ID]--
		if h.userRegistered[c.User.ID] <= 0 {
			delete(h.userRegistered, c.User.ID)
		}
	}

	rooms := h.clientRooms[c]
	// Loop over all rooms the client is subscribed to
	for room := range rooms {
		delete(room.clients, c)
		// delete the room if no one left in it
		if len(room.clients) == 0 {
			h.deleteRoom(room)
		}
	}

	delete(h.clientRooms, c)
	delete(h.clients, c)
	close(c.send)
}

func (h *Hub) Subscribe(c *Client, roomID string) {
	h.subscribe <- subscriptionReq{client: c, roomID: roomID}
}

func (h *Hub) handleSubscribe(s subscriptionReq) {

	if s.client == nil {
		return
	}

	// If client is not yet registered, add it
	if _, ok := h.clients[s.client]; !ok {
		h.clients[s.client] = true
		h.clientRooms[s.client] = make(map[*roomState]bool)
	}

	// If the room doesn't already exist, create it
	room, ok := h.rooms[s.roomID]
	if !ok {
		room = h.createRoom(s.roomID)
		h.rooms[s.roomID] = room
	}

	// Verify client has a subscription map
	if _, ok := h.clientRooms[s.client]; !ok {
		h.clientRooms[s.client] = make(map[*roomState]bool)
	}
	// Add the room to the client's subscriptions
	h.clientRooms[s.client][room] = true
	room.clients[s.client] = true
}

func (h *Hub) Unsubscribe(c *Client, id string) {
	select {
	case h.unsubscribe <- subscriptionReq{client: c, roomID: id}:
	case <-h.ctx.Done():
	}
}

func (h *Hub) handleUnsubscribe(s subscriptionReq) {

	if s.client == nil {
		return
	}

	// If client is not registered, skip
	if _, ok := h.clients[s.client]; !ok {
		return
	}

	room, ok := h.rooms[s.roomID]
	if !ok {
		return
	}

	// delete the room from the client's subscriptions
	if rooms, ok := h.clientRooms[s.client]; ok {
		delete(rooms, room)
	}

	delete(room.clients, s.client)
	// delete the room if no one left in it
	if len(room.clients) == 0 {
		h.deleteRoom(room)
	}

}

func (h *Hub) Broadcast(r broadcastReq) {
	select {
	case h.broadcast <- r:
	case <-h.ctx.Done():
	}
}

func (h *Hub) handleBroadcast(br broadcastReq) {

	room, ok := h.rooms[br.roomID]
	if !ok {
		return
	}

	dropped := make([]*Client, 0)

	for c := range room.clients {
		select {
		case c.send <- br.payload:
		default:
			dropped = append(dropped, c)
		}
	}

	// Drop slow clients
	for _, c := range dropped {
		c.cancel()
	}
}

func (h *Hub) createRoom(roomID string) *roomState {

	// If the room already exists, return early
	if room, ok := h.rooms[roomID]; ok {
		return room
	}

	roomCtx, roomCancel := context.WithCancel(h.ctx)
	roomPubSub := h.rdb.Subscribe(roomCtx, fmt.Sprintf("ws:room:%s", roomID))
	room := &roomState{
		id:          roomID,
		clients:     make(map[*Client]bool),
		cancel:      roomCancel,
		redisPubSub: roomPubSub,
	}

	ch := room.redisPubSub.Channel()

	h.wg.Add(1)
	go func() {

		defer h.wg.Done()

		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					return
				}
				h.Broadcast(broadcastReq{roomID: room.id, payload: []byte(msg.Payload)})
			case <-roomCtx.Done():
				return
			}
		}

	}()

	return room

}

func (h *Hub) deleteRoom(room *roomState) {

	if room == nil {
		return
	}

	// If the room is already deleted, return early
	if _, ok := h.rooms[room.id]; !ok {
		return
	}

	// Cancel the context to stop the redis goroutine
	room.cancel()

	if room.redisPubSub != nil {
		room.redisPubSub.Close()
	}

	// Delete all remaining clients's subscription to the room
	for c := range room.clients {
		if rooms, ok := h.clientRooms[c]; ok {
			delete(rooms, room)
		}
	}

	// Delete roomID entry
	delete(h.rooms, room.id)

}

func (h *Hub) updateOnlineUsers() {

	// Collect users, to avoid blocking the main loop on Redis operations (network = slow)
	loggedInUsers := make([]uuid.UUID, 0, len(h.userRegistered))
	for userID := range h.userRegistered {
		loggedInUsers = append(loggedInUsers, userID)
	}

	allUsers := make([]uuid.UUID, 0, len(h.clients))
	for c := range h.clients {
		if c.User != repos.AnonymousUser {
			allUsers = append(allUsers, c.ID)
		}
	}

	go h.UpdateCacheUserConnected(loggedInUsers, LoggedInUsersKey)
	go h.UpdateCacheUserConnected(allUsers, OnlineUsersKey)
}

func (h *Hub) UpdateCacheUserConnected(ids []uuid.UUID, key string) {
	ctx, cancel := context.WithTimeout(h.ctx, 2*time.Second)
	defer cancel()

	pipe := h.rdb.Pipeline()

	for _, userID := range ids {
		pipe.HSet(ctx, key, userID.String(), "1")
		pipe.HExpire(ctx, key, onlineUserTTL, userID.String())
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		h.logger.Error("failed to update online users", "key", key, "error", err)
	}
}

func (h *Hub) stop() {
	h.wg.Wait()
}
