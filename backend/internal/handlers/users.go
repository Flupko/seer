package handlers

import (
	"fmt"
	"net/http"
	"seer/internal/repos"
	"seer/internal/utils"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type UserHandler struct {
	ur *repos.UserRepo
}

func NewUserHandler(ur *repos.UserRepo) *UserHandler {
	return &UserHandler{
		ur: ur,
	}
}

type userMeRes struct {
	ID              uuid.UUID        `json:"id"`
	Email           string           `json:"email"`
	Username        string           `json:"username"`
	ProfileImageKey string           `json:"profileImageKey,omitempty"`
	Status          repos.UserStatus `json:"status"`
	Balance         int64            `json:"balance"`
}

func (h *UserHandler) UserMe(c echo.Context) error {

	ctx := c.Request().Context()
	user := utils.ContextGetUser(c)

	if user == repos.AnonymousUser {
		return c.JSON(http.StatusOK, utils.Envelope{"user": nil})
	}

	userView, err := h.ur.GetByID(ctx, user.ID)
	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("failed to get user: %w", err)
	}

	userResp := &userMeRes{
		ID:      userView.ID,
		Email:   userView.Email,
		Status:  userView.Status,
		Balance: userView.Balance,
	}

	if userView.Username.Valid {
		userResp.Username = userView.Username.String
	}

	if userView.ProfileImageKey.Valid {
		userResp.ProfileImageKey = userView.ProfileImageKey.String
	}

	return c.JSON(http.StatusOK, utils.Envelope{"user": userResp})
}
