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

func (h *WsHandler) HandleJoinMarketRooms(c *ws.Client, reqPayload string) {
	c.Join(ws.MarketsUpdateRoom)
}

func (h *WsHandler) HandleLeaveMarketRooms(c *ws.Client, reqPayload string) {
	c.Leave(ws.MarketsUpdateRoom)
}

type GetMarketStatePayload struct {
	MarketID uuid.UUID `json:"marketId" validate:"required,max=49,min=1"`
}

func (h WsHandler) HandleGetMarketState(c *ws.Client, reqPayload string) {
	ctx, cancel := context.WithTimeout(c.Ctx, 5*time.Second)
	defer cancel()

	p := &GetMarketStatePayload{}
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

	ms, err := h.msm.GetMarketState(ctx, p.MarketID)
	if err != nil {
		h.sendError(c, ErrCodeInternalError)
		return
	}

	wsPayload := ws.MarketUpdate{
		ID:      ms.ID,
		Version: ms.Version,
	}

	for i := range len(ms.QVec) {
		wsPayload.Outcomes = append(wsPayload.Outcomes,
			ws.OutcomeUpdate{
				ID:       ms.OutcomeIDs[i],
				Quantity: ms.QVec[i],
			})
	}

	wsMsg, err := utils.WsMessage(ws.MarketsUpdateRoom, wsPayload)
	if err != nil {
		h.sendError(c, ErrCodeInternalError)
		return
	}

	wsBuf, err := json.Marshal(wsMsg)
	if err != nil {
		h.sendError(c, ErrCodeInternalError)
		return
	}

	c.Send(wsBuf)

}

func (h *WsHandler) HandleJoinBetsRoom(c *ws.Client, reqPayload string) {

	ctx, cancel := context.WithTimeout(c.Ctx, 5*time.Second)
	defer cancel()

	latestBetsState, err := h.bm.GetLatestBets(ctx)
	if err != nil {
		h.sendError(c, ErrCodeInternalError)
		return
	}

	highBetsState, err := h.bm.GetHighBets(ctx)
	if err != nil {
		h.sendError(c, ErrCodeInternalError)
		return
	}

	highBetsAny := make([]any, 0, len(highBetsState))
	latestBetsAny := make([]any, 0, len(latestBetsState))

	for _, bs := range highBetsState {

		wsPayload := ws.BetUpdate{
			ID:          bs.ID,
			MarketID:    bs.MarketID,
			MarketName:  bs.MarketName,
			OutcomeID:   bs.OutcomeID,
			OutcomeName: bs.OutcomeName,
			Wager:       bs.Wager,
			Payout:      bs.Payout,
			AvgPrice:    bs.AvgPrice,
			PlacedAt:    bs.PlacedAt,
		}

		if bs.User != nil {
			wsPayload.User = &ws.UserState{
				ID:              bs.User.ID,
				Username:        bs.User.Username,
				ProfileImageKey: bs.User.ProfileImageKey,
			}
		}

		wsBuf, err := utils.WsMessage(ws.BetsHighRoom, wsPayload)
		if err != nil {
			h.sendError(c, ErrCodeInternalError)
			return
		}

		highBetsAny = append(highBetsAny, wsBuf)
	}

	for _, bs := range latestBetsState {

		wsPayload := ws.BetUpdate{
			ID:          bs.ID,
			MarketID:    bs.MarketID,
			MarketName:  bs.MarketName,
			OutcomeID:   bs.OutcomeID,
			OutcomeName: bs.OutcomeName,
			Wager:       bs.Wager,
			Payout:      bs.Payout,
			AvgPrice:    bs.AvgPrice,
			PlacedAt:    bs.PlacedAt,
		}

		if bs.User != nil {
			wsPayload.User = &ws.UserState{
				ID:              bs.User.ID,
				Username:        bs.User.Username,
				ProfileImageKey: bs.User.ProfileImageKey,
			}
		}

		wsBuf, err := utils.WsMessage(ws.BetsLatestRoom, wsPayload)
		if err != nil {
			h.sendError(c, ErrCodeInternalError)
			return
		}
		latestBetsAny = append(latestBetsAny, wsBuf)
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

	c.Join(ws.BetsLatestRoom)
	c.Join(ws.BetsHighRoom)

}

func (h *WsHandler) HandleLeaveBetsRoom(c *ws.Client, reqPayload string) {
	c.Leave(ws.BetsLatestRoom)
	c.Leave(ws.BetsHighRoom)
}

type JoinLeaveChatRoomPaylod struct {
	ChatSlug string `json:"chatSlug" validate:"required,lowercase,alphanum,min=3,max=20"`
}

func (h *WsHandler) HandleJoinChatRoom(c *ws.Client, reqPayload string) {

	ctx, cancel := context.WithTimeout(c.Ctx, 5*time.Second)
	defer cancel()

	p := &JoinLeaveChatRoomPaylod{}
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

	msgsAny := make([]any, 0, len(msgs))
	wsChatRoom := chat.BuildWSChatRoom(p.ChatSlug)

	for i, m := range msgs {

		wsPayload := ws.ChatMessage{
			ID:        m.ID,
			ChatSlug:  m.ChatSlug,
			Content:   m.Content,
			CreatedAt: m.CreatedAt,
			Type:      m.Type,
		}
		wsPayload.User = ws.UserState{
			ID:              m.User.ID,
			Username:        m.User.Username,
			ProfileImageKey: m.User.ProfileImageKey,
		}

		wsBuf, err := utils.WsMessage(wsChatRoom, wsPayload)
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

func (h *WsHandler) HandleLeaveChatRoom(c *ws.Client, reqPayload string) {

	p := &JoinLeaveChatRoomPaylod{}
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

	wsChatRoom := chat.BuildWSChatRoom(p.ChatSlug)
	c.Leave(wsChatRoom)
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

	sent, err := h.cm.SendMessage(ctx, c.User, p.Message, p.ChatSlug)
	if err != nil {
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
	c.Join(ws.OnlineRoom)

	wsBuf, err := h.op.GetOnlineWsMsg()
	if err != nil {
		h.sendError(c, ErrCodeInternalError)
		return
	}

	c.Send(wsBuf)
}

func (h *WsHandler) HandleLeaveOnlineRoom(c *ws.Client, reqPayload string) {
	c.Leave(ws.OnlineRoom)
}

func (h *WsHandler) sendError(c *ws.Client, message string) {
	wsPayload := ws.WSError{Error: message}

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
