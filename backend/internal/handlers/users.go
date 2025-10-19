package handlers

import (
	"fmt"
	"net/http"
	"seer/internal/repos"
	"seer/internal/utils"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type UserHandler struct {
	validate *validator.Validate
	ur       *repos.UserRepo
}

func NewUserHandler(v *validator.Validate, ur *repos.UserRepo) *UserHandler {
	return &UserHandler{
		validate: v,
		ur:       ur,
	}
}

type userMeRes struct {
	ID              uuid.UUID          `json:"id"`
	ProviderID      repos.AuthProvider `json:"providerId,omitempty"`
	HasPassword     bool               `json:"hasPassword"`
	Email           string             `json:"email"`
	Username        string             `json:"username"`
	ProfileImageKey string             `json:"profileImageKey,omitempty"`
	Status          repos.UserStatus   `json:"status"`
	Balance         int64              `json:"balance"`
}

func (h *UserHandler) UserMe(c echo.Context) error {

	ctx := c.Request().Context()
	user := utils.ContextGetUser(c)

	if user == repos.AnonymousUser {
		return c.JSON(http.StatusOK, nil)
	}

	userView, err := h.ur.GetViewByID(ctx, user.ID)
	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("failed to get user: %w", err)
	}

	userResp := &userMeRes{
		ID:          userView.ID,
		Email:       userView.Email,
		ProviderID:  userView.ProviderID,
		HasPassword: userView.PasswordHash != nil,
		Status:      userView.Status,
		Balance:     userView.Balance,
	}

	if userView.Username.Valid {
		userResp.Username = userView.Username.String
	}

	if userView.ProfileImageKey.Valid {
		userResp.ProfileImageKey = userView.ProfileImageKey.String
	}

	return c.JSON(http.StatusOK, userResp)
}

type preferencesRes struct {
	Hidden                 bool `json:"hidden"`
	ReceiveMarketingEmails bool `json:"receiveMarketingEmails"`
}

func (h *UserHandler) GetPreferences(c echo.Context) error {

	ctx := c.Request().Context()
	userID := utils.ContextGetUser(c).ID

	prefs, err := h.ur.GetPreferences(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	res := &preferencesRes{
		Hidden:                 prefs.Hidden,
		ReceiveMarketingEmails: prefs.ReceiveMarketingEmails,
	}

	return c.JSON(http.StatusOK, res)

}

type updatePreferencesReq struct {
	Hidden                 *bool `json:"hidden" validate:"omitempty"`
	ReceiveMarketingEmails *bool `json:"receiveMarketingEmails" validate:"omitempty"`
}

func (h *UserHandler) UpdatePreferences(c echo.Context) error {

	ctx := c.Request().Context()
	userID := utils.ContextGetUser(c).ID

	r := &updatePreferencesReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, r, h.validate); err != nil {
		return err
	}

	prefs, err := h.ur.GetPreferences(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user preferences: %w", err)
	}

	if r.Hidden != nil {
		prefs.Hidden = *r.Hidden
	}

	if r.ReceiveMarketingEmails != nil {
		prefs.ReceiveMarketingEmails = *r.ReceiveMarketingEmails
	}

	if err := h.ur.UpdatePreferences(ctx, userID, prefs); err != nil {
		return fmt.Errorf("failed to update user preferences: %w", err)
	}

	return c.JSON(http.StatusOK, prefs)

}
