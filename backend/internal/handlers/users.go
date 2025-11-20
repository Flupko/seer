package handlers

import (
	"fmt"
	"net/http"
	"seer/internal/numeric"
	"seer/internal/repos"
	"seer/internal/utils"
	"time"

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
	TotalWagered    numeric.BigDecimal `json:"totalWagered"`
	CreatedAt       time.Time          `json:"createdAt"`
	Status          repos.UserStatus   `json:"status"`
}

func (h *UserHandler) UserMe(c echo.Context) error {

	ctx := c.Request().Context()
	user := utils.ContextGetUser(c)

	if user == repos.AnonymousUser {
		return c.JSON(http.StatusOK, nil)
	}

	userView, err := h.ur.GetViewByIDOrUsername(ctx, user.ID, "")
	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("failed to get user: %w", err)
	}

	userResp := &userMeRes{
		ID:           userView.ID,
		Email:        userView.Email,
		ProviderID:   userView.ProviderID,
		HasPassword:  userView.PasswordHash != nil,
		TotalWagered: userView.TotalWagered,
		CreatedAt:    userView.CreatedAt,
		Status:       userView.Status,
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

type userProfileReq struct {
	Username string `param:"username" validate:"required,max=30"`
}

type PublicUserRes struct {
	ID              uuid.UUID          `json:"id"`
	Username        string             `json:"username"`
	ProfileImageKey string             `json:"profileImageKey,omitempty"`
	TotalWagered    numeric.BigDecimal `json:"totalWagered"`
	CreatedAt       time.Time          `json:"createdAt"`
}

func UserViewToPublicRes(userView *repos.UserView) *PublicUserRes {
	r := &PublicUserRes{
		ID:           userView.ID,
		Username:     userView.Username.String,
		TotalWagered: userView.TotalWagered,
		CreatedAt:    userView.CreatedAt,
	}

	if userView.ProfileImageKey.Valid {
		r.ProfileImageKey = userView.ProfileImageKey.String
	}

	return r
}

func (h *UserHandler) GetUserProfile(c echo.Context) error {

	r := &userProfileReq{}
	if err := utils.ParseAndValidadePathParams(c, r, h.validate); err != nil {
		return err
	}

	ctx := c.Request().Context()

	userView, err := h.ur.GetViewByIDOrUsername(ctx, uuid.Nil, r.Username)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if userView.Hidden {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	res := UserViewToPublicRes(userView)

	return c.JSON(http.StatusOK, res)

}
