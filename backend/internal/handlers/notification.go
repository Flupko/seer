package handlers

import (
	"fmt"
	"net/http"
	"seer/internal/notif"
	"seer/internal/utils"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type NotificationHandler struct {
	validate *validator.Validate
	nm       *notif.NotificationManager
}

func NewNotificationHandler(v *validator.Validate, nm *notif.NotificationManager) *NotificationHandler {
	return &NotificationHandler{
		validate: v,
		nm:       nm,
	}
}

type notifRes struct {
	Type notif.NotificationType
	Data any
}

func (h *NotificationHandler) GetUnreadNotifications(c echo.Context) error {

	user := utils.ContextGetUser(c)
	ctx := c.Request().Context()

	notifs, err := h.nm.GetUnreadNotifications(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("failed to get unread notifications: %w", err)
	}

	resps := make([]*notifRes, 0, len(notifs))
	for _, n := range notifs {
		nr := &notifRes{
			Type: n.Type,
			Data: n.Data,
		}
		resps = append(resps, nr)
	}

	return c.JSON(http.StatusOK, utils.Envelope{"notifications": resps})

}

type readNotifReq struct {
	NotifIDs []int64 `json:"notifIds" validate:"required,min=1,max=10,dive,required"`
}

func (h *NotificationHandler) ReadNotifications(c echo.Context) error {
	r := &readNotifReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, r, h.validate); err != nil {
		return err
	}

	user := utils.ContextGetUser(c)
	ctx := c.Request().Context()

	if err := h.nm.MarkAsRead(ctx, user.ID, r.NotifIDs); err != nil {
		return fmt.Errorf("failed to read notifcations")
	}

	return c.JSON(http.StatusOK, utils.Envelope{"message": "notifications read succesfully"})
}
