package utils

import (
	"encoding/json"
	"fmt"
	"seer/internal/ws"
)

func WsMessage(msgType string, paylaod any) (ws.Message, error) {

	buf, err := json.Marshal(paylaod)
	if err != nil {
		return ws.Message{}, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return ws.Message{
		Type:    msgType,
		Payload: buf,
	}, nil

}
