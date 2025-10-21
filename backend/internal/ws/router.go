package ws

import (
	"bytes"
	"encoding/json"

	"github.com/go-playground/validator/v10"
)

type Message struct {
	Type    string          `json:"type" validate:"required,max=30"`
	Payload json.RawMessage `json:"payload"`
}

type WsHandlerFunc func(c *Client, payload string)

type SocketRouter struct {
	routes   map[string]WsHandlerFunc
	validate *validator.Validate
}

func NewSocketRouter(v *validator.Validate) *SocketRouter {
	return &SocketRouter{
		routes:   make(map[string]WsHandlerFunc),
		validate: v,
	}
}

func (r *SocketRouter) AddRouteHandler(routeName string, f WsHandlerFunc) {
	r.routes[routeName] = f
}

func (r *SocketRouter) routeMessage(c *Client, rawMessage []byte) {

	dec := json.NewDecoder(bytes.NewReader(rawMessage))
	dec.DisallowUnknownFields()

	for dec.More() {
		var m Message
		if err := dec.Decode(&m); err != nil {
			c.Disconnect()
			return
		}

		err := r.validate.Struct(m)
		if err != nil {
			c.Disconnect()
			return
		}

		handler, ok := r.routes[m.Type]
		if !ok {
			c.Disconnect()
			return
		}

		handler(c, string(m.Payload))

	}

}
