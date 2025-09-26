package handlers

import "time"

type AdminModerateHandler struct {
}

func NewAdminModerateHandler() *AdminModerateHandler {
	return &AdminModerateHandler{}
}

type userMuteReq struct {
	UserID       string        `json:"userId"`
	MuteDuration time.Duration `json:"muteDuration"`
}

func (h *AdminModerateHandler) MuteUser() {}
