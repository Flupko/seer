package handlers

import (
	"log"
	"seer/internal/ws"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

type SocketHandler struct {
	hub      *ws.Hub
	router   *ws.SocketRouter
	upgrader *websocket.Upgrader
}

func NewSocketHandler(hub *ws.Hub, router *ws.SocketRouter, upgrader *websocket.Upgrader) *SocketHandler {
	return &SocketHandler{
		hub:      hub,
		router:   router,
		upgrader: upgrader,
	}
}

func (h *SocketHandler) ServeWS(c echo.Context) error {
	conn, err := h.upgrader.Upgrade(c.Response().Writer, c.Request(), nil)
	if err != nil {
		log.Println(err)
		return nil
	}

	client := ws.NewClient(conn, h.hub, uuid.Nil)
	h.hub.Register(client)
	client.Start(h.router)

	return nil
}
