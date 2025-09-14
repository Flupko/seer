package market

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"seer/internal/utils"
	"seer/internal/ws"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

const wsMarketRoomPrefix = "market:"

type WsHandler struct {
	msm      *StateManager
	validate *validator.Validate
}

func NewWsHandler(msm *StateManager, validate *validator.Validate) *WsHandler {
	return &WsHandler{
		msm:      msm,
		validate: validate,
	}
}

const (
	ErrCodeInvalidPayload  = "invalid_payload"
	ErrCodeInvalidMarkets  = "invalid_markets"
	ErrCodeInternalError   = "internal_error"
	ErrCodeValidationError = "validation_error"
)

type JoinMarketPayload struct {
	MarketIDs []uuid.UUID `json:"market_ids" validate:"required,max=49,min=1"`
}

func (h *WsHandler) HandleJoinMarketRooms(c *ws.Client, reqPayload string) {

	if reqPayload == "" {
		c.Disconnect()
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	p := &JoinMarketPayload{}
	if err := utils.ReadJson(strings.NewReader(reqPayload), p); err != nil {
		switch {
		case errors.Is(err, utils.ErrInvalidJSON):
			h.sendError(c, ErrCodeInvalidPayload)
		default:
			h.sendError(c, ErrCodeInternalError)
		}
		c.Disconnect()
		return
	}

	if err := h.validate.Struct(p); err != nil {
		h.sendError(c, ErrCodeInvalidPayload)
		c.Disconnect()
		return
	}

	validMarkets, err := h.msm.GetValidMarkets(ctx, p.MarketIDs)
	if err != nil {
		h.sendError(c, ErrCodeInternalError)
		c.Disconnect()
		return
	}

	if len(validMarkets) != len(p.MarketIDs) {
		h.sendError(c, ErrCodeInvalidMarkets)
		return
	}

	for _, marketID := range validMarkets {
		c.Join(fmt.Sprintf("%s%s", wsMarketRoomPrefix, marketID))
	}

	for _, marketID := range validMarkets {

		ms, err := h.msm.getMarketState(ctx, marketID)
		if err != nil {
			h.sendError(c, ErrCodeInternalError)
			c.Disconnect()
			return
		}

		wsPayload := WSPayloadMarketUpdate{
			ID:      ms.ID,
			Version: ms.Version,
		}

		for i := range len(ms.QVec) {
			wsPayload.Outcomes = append(wsPayload.Outcomes,
				WSPayloadOutcomeUpdate{
					ID:     ms.OutcomeIDs[i],
					OddPPH: ms.OddsPPH[i],
				})
		}

		buf, err := json.Marshal(wsPayload)
		if err != nil {
			h.sendError(c, ErrCodeInternalError)
			c.Disconnect()
			return
		}

		wsMsg := ws.Message{
			Type:    "market_update",
			Payload: buf,
		}

		wsBuf, err := json.Marshal(wsMsg)
		if err != nil {
			h.sendError(c, ErrCodeInternalError)
			c.Disconnect()
			return
		}

		c.Send(wsBuf)

	}
}

type WSError struct {
	Error string `json:"error"`
}

func (h *WsHandler) sendError(c *ws.Client, message string) {
	wsPayload := WSError{Error: message}

	buf, err := json.Marshal(wsPayload)
	if err != nil {
		return
	}
	wsMsg := ws.Message{
		Type:    "error",
		Payload: buf,
	}

	wsBuf, err := json.Marshal(wsMsg)
	if err != nil {
		return
	}

	c.Send(wsBuf)
}
