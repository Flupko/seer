package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"seer/internal/chat"
	"seer/internal/market"
	"seer/internal/utils"
	"seer/internal/ws"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type WsHandler struct {
	bm       *market.BetLiveManager
	msm      *market.StateManager
	cm       *chat.ChatManager
	validate *validator.Validate
}

func NewWsHandler(bm *market.BetLiveManager, msm *market.StateManager, cm *chat.ChatManager, validate *validator.Validate) *WsHandler {
	return &WsHandler{
		bm:       bm,
		msm:      msm,
		cm:       cm,
		validate: validate,
	}
}

const (
	ErrCodeInvalidPayload   = "invalid_payload"
	ErrCodeInvalidMarkets   = "invalid_markets"
	ErrCodeInternalError    = "internal_error"
	ErrCodeValidationError  = "validation_error"
	ErrCodeNotAuthenticated = "not_authenticated_error"
)

type JoinMarketPayload struct {
	MarketIDs []uuid.UUID `json:"market_ids" validate:"required,max=49,min=1"`
}

func (h *WsHandler) HandleJoinMarketRooms(c *ws.Client, reqPayload string) {

	ctx, cancel := context.WithTimeout(c.Ctx, 15*time.Second)
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
		return
	}

	if len(validMarkets) != len(p.MarketIDs) {
		h.sendError(c, ErrCodeInvalidMarkets)
		return
	}

	for _, marketID := range validMarkets {
		c.Join(fmt.Sprintf("%s%s", market.WsMarketRoomPrefix, marketID))
	}

	for _, marketID := range validMarkets {

		ms, err := h.msm.GetMarketState(ctx, marketID)
		if err != nil {
			h.sendError(c, ErrCodeInternalError)
			return
		}

		wsPayload := market.WSPayloadMarketUpdate{
			ID:      ms.ID,
			Version: ms.Version,
		}

		for i := range len(ms.QVec) {
			wsPayload.Outcomes = append(wsPayload.Outcomes,
				market.WSPayloadOutcomeUpdate{
					ID:     ms.OutcomeIDs[i],
					OddPPH: ms.OddsPPH[i],
				})
		}

		buf, err := json.Marshal(wsPayload)
		if err != nil {
			h.sendError(c, ErrCodeInternalError)
			return
		}

		wsMsg := ws.Message{
			Type:    "market_update",
			Payload: buf,
		}

		wsBuf, err := json.Marshal(wsMsg)
		if err != nil {
			h.sendError(c, ErrCodeInternalError)
			return
		}

		c.Send(wsBuf)

	}
}

func (h *WsHandler) HandleJoinBetsRoom(c *ws.Client, reqPayload string) {

	ctx, cancel := context.WithTimeout(c.Ctx, 5*time.Second)
	defer cancel()

	latestBetsState, err := h.bm.GetLatestBets(ctx)
	if err != nil {
		fmt.Println("error getting latest bets:", err)
		h.sendError(c, ErrCodeInternalError)
		return
	}

	highBetsState, err := h.bm.GetHighBets(ctx)
	if err != nil {
		fmt.Println("error getting high bets:", err)
		h.sendError(c, ErrCodeInternalError)
		return
	}

	highBetsAny := make([]any, len(highBetsState))
	latestBetsAny := make([]any, len(latestBetsState))

	for i, bs := range highBetsState {

		wsBuf, err := utils.WsMessage(market.WsBetsHighRoom, bs)
		if err != nil {
			h.sendError(c, ErrCodeInternalError)
			return
		}

		highBetsAny[i] = wsBuf
	}

	for i, bs := range latestBetsState {
		wsBuf, err := utils.WsMessage(market.WsBetsLatestRoom, bs)
		if err != nil {
			h.sendError(c, ErrCodeInternalError)
			return
		}
		latestBetsAny[i] = wsBuf
	}

	err1 := c.SendBatchJSON(latestBetsAny)
	err2 := c.SendBatchJSON(highBetsAny)

	if err := errors.Join(err1, err2); err != nil {
		h.sendError(c, ErrCodeInternalError)
	}

	c.Join(market.WsBetsLatestRoom)
	c.Join(market.WsBetsHighRoom)

}

func (h *WsHandler) HandleJoinGlobalChat(c *ws.Client, reqPayload string) {

	ctx, cancel := context.WithTimeout(c.Ctx, 5*time.Second)
	defer cancel()

	msgs, err := h.cm.GetLastMessages(ctx)
	if err != nil {
		h.sendError(c, ErrCodeInternalError)
		return
	}

	msgsAny := make([]any, len(msgs))

	for i, m := range msgs {

		wsBuf, err := utils.WsMessage(chat.WSChatRoom, m)
		if err != nil {
			h.sendError(c, ErrCodeInternalError)
			return
		}

		msgsAny[i] = wsBuf
	}

	if err := c.SendBatchJSON(msgsAny); err != nil {
		h.sendError(c, ErrCodeInternalError)
		return
	}

	c.Join(chat.WSChatRoom)
}

type SendMessagePayload struct {
	Message string `json:"message" validate:"min=1,max=50"`
}

func (h *WsHandler) HandleSendMessage(c *ws.Client, reqPayload string) {

	// Validate the user is connected
	// if c.User == repos.AnonymousUser {
	// 	h.sendError(c, ErrCodeNotAuthenticated)
	// }

	ctx, cancel := context.WithTimeout(c.Ctx, 5*time.Second)
	defer cancel()

	_ = ctx

	p := &SendMessagePayload{}
	if err := utils.ReadJson(strings.NewReader(reqPayload), p); err != nil {
		switch {
		case errors.Is(err, utils.ErrInvalidJSON):
			h.sendError(c, ErrCodeInvalidPayload)
			c.Disconnect()
		default:
			h.sendError(c, ErrCodeInternalError)
		}
		return
	}

	if err := h.validate.Struct(p); err != nil {
		h.sendError(c, ErrCodeInvalidPayload)
		c.Disconnect()
		return
	}

	sent, err := h.cm.SendMessage(ctx, c.User, p.Message)
	if err != nil {
		fmt.Println("err", err)
		h.sendError(c, ErrCodeInternalError)
		return
	}

	// Rate limited, inform client
	if sent {
		wsAckMsg := ws.Message{Type: chat.WSChatRoom + ":ack"}
		if buf, err := json.Marshal(wsAckMsg); err == nil {
			c.Send(buf)
		} else {
			h.sendError(c, ErrCodeInternalError)
		}
	} else {
		wsRateMsg := ws.Message{Type: chat.WSChatRoom + ":rate"}
		if buf, err := json.Marshal(wsRateMsg); err == nil {
			c.Send(buf)
		} else {
			h.sendError(c, ErrCodeInternalError)
		}
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
