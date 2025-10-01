package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"seer/internal/chat"
	"seer/internal/market"
	"seer/internal/repos"
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
	op       *ws.OnlinePusher
	validate *validator.Validate
}

func NewWsHandler(bm *market.BetLiveManager, msm *market.StateManager, cm *chat.ChatManager, op *ws.OnlinePusher, validate *validator.Validate) *WsHandler {
	return &WsHandler{
		bm:       bm,
		msm:      msm,
		cm:       cm,
		op:       op,
		validate: validate,
	}
}

const (
	ErrCodeInvalidPayload   = "invalid_payload"
	ErrCodeInvalidMarkets   = "invalid_markets"
	ErrCodeInternalError    = "internal_error"
	ErrCodeValidationError  = "validation_error"
	ErrCodeNotAuthenticated = "not_authenticated_error"
	ErrCodeNotAuthorized    = "not_authorized"
	ErrCodeChatRoomNotFound = "chat_room_not_found"
)

type JoinMarketPayload struct {
	MarketIDs []uuid.UUID `json:"marketIds" validate:"required,max=49,min=1"`
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
					ID:      ms.OutcomeIDs[i],
					ProbPPM: ms.PricesPPM[i],
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

	var err1, err2 error
	if len(latestBetsAny) > 0 {
		err1 = c.SendBatchJSON(latestBetsAny)
	}
	if len(highBetsAny) > 0 {
		err2 = c.SendBatchJSON(highBetsAny)
	}

	if err := errors.Join(err1, err2); err != nil {
		h.sendError(c, ErrCodeInternalError)
	}

	c.Join(market.WsBetsLatestRoom)
	c.Join(market.WsBetsHighRoom)

}

type JoinChatRoomPaylod struct {
	ChatSlug string `json:"chatSlug" validate:"required,lowercase,alphanum,min=3,max=20"`
}

func (h *WsHandler) HandleJoinChatRoom(c *ws.Client, reqPayload string) {

	ctx, cancel := context.WithTimeout(c.Ctx, 5*time.Second)
	defer cancel()

	p := &JoinChatRoomPaylod{}
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

	msgs, err := h.cm.GetLastMessagesChat(ctx, p.ChatSlug)
	if err != nil {
		fmt.Println("error getting last messages:", err)
		switch {
		case errors.Is(err, chat.ErrChatRoomNotFound):
			h.sendError(c, ErrCodeChatRoomNotFound)
		default:
			h.sendError(c, ErrCodeInternalError)
		}
		return
	}

	msgsAny := make([]any, len(msgs))
	wsChatRoom := chat.BuildWSChatRoom(p.ChatSlug)

	for i, m := range msgs {

		wsBuf, err := utils.WsMessage(wsChatRoom, m)
		if err != nil {
			h.sendError(c, ErrCodeInternalError)
			return
		}

		msgsAny[i] = wsBuf
	}

	if len(msgsAny) > 0 {
		if err := c.SendBatchJSON(msgsAny); err != nil {
			h.sendError(c, ErrCodeInternalError)
			return
		}
	}

	c.Join(wsChatRoom)
}

type SendMessagePayload struct {
	Message  string `json:"message" validate:"min=1,max=50"`
	ChatSlug string `json:"chatSlug" validate:"required,lowercase,alphanum,min=3,max=20"`
}

func (h *WsHandler) HandleSendMessage(c *ws.Client, reqPayload string) {

	if c.User.MutedUntil.Valid && c.User.MutedUntil.Time.After(time.Now()) {
		h.sendError(c, ErrCodeNotAuthorized)
		return
	}

	ctx, cancel := context.WithTimeout(c.Ctx, 5*time.Second)
	defer cancel()

	_ = ctx

	p := &SendMessagePayload{}
	if err := utils.ReadJson(strings.NewReader(reqPayload), p); err != nil {
		fmt.Println("error reading json:", err)
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
		fmt.Println("validation error:", err)
		h.sendError(c, ErrCodeInvalidPayload)
		c.Disconnect()
		return
	}

	sent, err := h.cm.SendMessage(ctx, c.User, p.Message, p.ChatSlug)
	if err != nil {
		fmt.Println("error sending message:", err)
		switch {
		case errors.Is(err, chat.ErrChatRoomNotFound):
			h.sendError(c, ErrCodeChatRoomNotFound)
		default:
			h.sendError(c, ErrCodeInternalError)
		}
		return
	}

	wsChatRoom := chat.BuildWSChatRoom(p.ChatSlug)
	if sent {
		wsAckMsg := ws.Message{Type: wsChatRoom + ":ack"}
		if buf, err := json.Marshal(wsAckMsg); err == nil {
			c.Send(buf)
		} else {
			h.sendError(c, ErrCodeInternalError)
		}
	} else {
		// Rate limited, inform client
		wsRateMsg := ws.Message{Type: wsChatRoom + ":rate"}
		if buf, err := json.Marshal(wsRateMsg); err == nil {
			c.Send(buf)
		} else {
			h.sendError(c, ErrCodeInternalError)
		}
	}

}

func (h *WsHandler) RequireAuthentication(next ws.WsHandlerFunc) ws.WsHandlerFunc {
	return func(c *ws.Client, reqPayload string) {

		// Validate the user is connected
		if c.User == repos.AnonymousUser {
			h.sendError(c, ErrCodeNotAuthenticated)
			return
		}

		// Validate the user is activated
		if c.User.Status != repos.Activated {
			h.sendError(c, ErrCodeNotAuthorized)
			return
		}

		// Call  next
		next(c, reqPayload)
	}
}

func (h *WsHandler) HandleJoinOnlineRoom(c *ws.Client, reqPayload string) {
	c.Join(ws.WsOnlineRoom)

	wsBuf, err := h.op.GetOnlineWsMsg()
	if err != nil {
		h.sendError(c, ErrCodeInternalError)
		return
	}

	c.Send(wsBuf)
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
