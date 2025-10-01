package ws

import (
	"bytes"
	"context"
	"encoding/json"
	"seer/internal/repos"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (

	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 30 * time.Second

	// Send pings to peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 10 * 1024 // 10 KB, more than enough

	// Max buffer size
	clientSendBufSize = 256
)

var (
	newline = []byte{'\n'}
)

type Client struct {
	ID     uuid.UUID
	conn   *websocket.Conn
	User   *repos.MinimalUser
	send   chan []byte
	hub    *Hub
	Ctx    context.Context
	cancel context.CancelFunc
}

func NewClient(conn *websocket.Conn, hub *Hub, user *repos.MinimalUser) *Client {

	clientCtx, clientCancel := context.WithCancel(hub.ctx)

	c := &Client{
		ID:     uuid.New(),
		conn:   conn,
		User:   user,
		send:   make(chan []byte, clientSendBufSize),
		hub:    hub,
		Ctx:    clientCtx,
		cancel: clientCancel,
	}

	go func() {
		<-clientCtx.Done()
		hub.Unregister(c) // Unregister the client from the hub
		c.conn.Close()
	}()

	return c
}

func (c *Client) Start(router *SocketRouter) {
	go c.writePump()
	go c.readPump(router)
}

func (c *Client) Send(payload []byte) {
	select {
	case c.send <- payload:
	default:
		c.Disconnect()
	}
}

func (c *Client) SendBatchJSON(vals []any) error {

	var b bytes.Buffer
	for i, v := range vals {
		enc, err := json.Marshal(v)
		if err != nil {
			return err
		}
		b.Write(enc)
		if i < len(vals)-1 {
			b.WriteByte('\n')
		}
	}

	payload := b.Bytes()

	select {
	case c.send <- payload:
	default:
		c.Disconnect()
	}

	return nil
}

func (c *Client) Join(roomID string) {
	c.hub.Subscribe(c, roomID)
}

func (c *Client) Leave(roomID string) {
	c.hub.Unsubscribe(c, roomID)
}

func (c *Client) Disconnect() {
	c.cancel()
}

func (c *Client) readPump(router *SocketRouter) {

	defer c.cancel()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		router.routeMessage(c, message)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.cancel()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			// Closed by the hub
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			if _, err := w.Write(message); err != nil {
				w.Close()
				return
			}
			n := len(c.send)
			for range n {
				msg := <-c.send
				if _, err := w.Write(newline); err != nil {
					w.Close()
					return
				}
				if _, err := w.Write(msg); err != nil {
					w.Close()
					return
				}
			}
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
				return
			}
		}
	}
}
