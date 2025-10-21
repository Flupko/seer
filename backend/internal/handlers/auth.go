package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"seer/internal/config"
	"seer/internal/geo"
	"seer/internal/repos"
	"seer/internal/utils"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/mileusna/useragent"
	"golang.org/x/oauth2"
)

type UserData struct {
	Email         string  `json:"email"`
	EmailVerified bool    `json:"email_verified"`
	Picture       *string `json:"picture"`
	Sub           string  `json:"sub"`
}

type AuthHandler struct {
	validate    *validator.Validate
	logger      *slog.Logger
	providers   map[repos.AuthProvider]*ProviderConfig
	userRepo    *repos.UserRepo
	sessionRepo *repos.SessionRepo
	tokenRepo   *repos.TokenRepo
	geoService  *geo.GeoService
}

type ProviderConfig struct {
	oauth2   *oauth2.Config
	verifier *oidc.IDTokenVerifier
}

func NewAuthHandler(ctx context.Context,
	validate *validator.Validate,
	logger *slog.Logger,
	userRepo *repos.UserRepo,
	sessionRepo *repos.SessionRepo,
	tokenRepo *repos.TokenRepo,
	geoService *geo.GeoService) (*AuthHandler, error) {

	// Setup Google provider
	googleProvider, err := oidc.NewProvider(ctx, "https://accounts.google.com")
	_ = err
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Google provider: %w", err)
	}

	// Setup Twitch provider
	twitchProvider, err := oidc.NewProvider(ctx, "https://id.twitch.tv/oauth2")
	_ = err
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Twitch provider: %w", err)
	}

	providers := map[repos.AuthProvider]*ProviderConfig{
		repos.GoogleProvider: {
			oauth2: &oauth2.Config{
				ClientID:     os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
				ClientSecret: os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
				RedirectURL:  fmt.Sprintf("%s/auth/provider/%s/callback", os.Getenv("API_BASE_URL"), repos.GoogleProvider),
				Scopes:       []string{"openid", "email", "profile"},
				Endpoint:     googleProvider.Endpoint(),
			},
			verifier: googleProvider.Verifier(&oidc.Config{
				ClientID: os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
			}),
		},
		repos.TwitchProvider: {
			oauth2: &oauth2.Config{
				ClientID:     os.Getenv("TWITCH_OAUTH_CLIENT_ID"),
				ClientSecret: os.Getenv("TWITCH_OAUTH_CLIENT_SECRET"),
				RedirectURL:  fmt.Sprintf("%s/auth/provider/%s/callback", os.Getenv("API_BASE_URL"), repos.TwitchProvider),
				Scopes:       []string{"openid", "user:read:email"},
				Endpoint:     twitchProvider.Endpoint(),
			},
			verifier: twitchProvider.Verifier(&oidc.Config{
				ClientID: os.Getenv("TWITCH_OAUTH_CLIENT_ID"),
			}),
		},
	}

	return &AuthHandler{
		validate:    validate,
		logger:      logger,
		providers:   providers,
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		tokenRepo:   tokenRepo,
		geoService:  geoService,
	}, nil
}

func (h *AuthHandler) ProviderLogin(c echo.Context) error {

	authProvider, err := validateProvider(c.Param("provider"))
	if err != nil {
		return h.redirectWithError(c, false)
	}

	providerCfg, exists := h.providers[authProvider]
	if !exists {
		return h.redirectWithError(c, false)
	}

	state, err := generateSecureToken()
	if err != nil {
		return h.redirectWithError(c, true)
	}

	cookieName := fmt.Sprintf("oauth_state_%s", authProvider)

	c.SetCookie(&http.Cookie{
		Name:     cookieName,
		Value:    state,
		HttpOnly: true,
		Secure:   config.IsProduction,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		MaxAge:   int(5 * time.Minute.Seconds()),
	})

	authURL := providerCfg.oauth2.AuthCodeURL(state, oauth2.AccessTypeOnline)
	return c.JSON(http.StatusOK, utils.Envelope{"url": authURL})
}

func (h *AuthHandler) GetAuthCallback(c echo.Context) (test error) {
	ctx := c.Request().Context()

	authProvider, err := validateProvider(c.Param("provider"))
	if err != nil {

		return h.redirectWithError(c, false)
	}

	providerCfg, exists := h.providers[authProvider]
	if !exists {
		return h.redirectWithError(c, false)
	}

	code := c.QueryParam("code")
	if code == "" || len(code) > 4096 {
		return h.redirectWithError(c, false)
	}

	// Verify state parameter (CSRF protection)
	cookieName := fmt.Sprintf("oauth_state_%s", authProvider)
	storedState, err := utils.GetAndClearCookie(c, cookieName)
	if err != nil {
		return h.redirectWithError(c, false)
	}

	receivedState := c.QueryParam("state")
	if storedState != receivedState {
		return h.redirectWithError(c, false)
	}

	// Exchange code for tokens
	oauthToken, err := providerCfg.oauth2.Exchange(ctx, code)
	if err != nil {
		return h.redirectWithError(c, false)
	}

	// Extract and verify ID token
	rawIDToken, ok := oauthToken.Extra("id_token").(string)
	if !ok {
		return h.redirectWithError(c, false)
	}

	idToken, err := providerCfg.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return h.redirectWithError(c, false)
	}

	// Parse user claims
	var userData UserData
	if err := idToken.Claims(&userData); err != nil {
		return h.redirectWithError(c, false)
	}

	// If user is not activated, abort early
	if !userData.EmailVerified {
		return h.redirectWithError(c, true)
	}

	// Get or create user
	user, err := h.getOrCreateUser(ctx, authProvider, &userData)
	if err != nil {
		return h.redirectWithError(c, true)
	}

	// Handle user state and create appropriate tokens
	switch user.Status {
	case repos.PendingProfile:
		// Clean up existing profile completion tokens
		h.tokenRepo.DeleteAllForUser(ctx, repos.ScopeProfileCompletion, user.ID)

		// Generate profile completion token
		tokenPlain, token, err := repos.GenerateToken(user.ID, repos.ScopeProfileCompletion, 5*time.Minute)
		if err != nil {
			return h.redirectWithError(c, true)
		}

		if err := h.tokenRepo.Insert(ctx, token); err != nil {
			return h.redirectWithError(c, true)
		}

		// Set profile completion cookie
		utils.SetSecureCookie(c, "profile_completion_token", tokenPlain, 5*time.Minute)

		redirectURL := fmt.Sprintf("%s/?show=profile_completion", os.Getenv("FRONTEND_URL"))
		return c.Redirect(http.StatusFound, redirectURL)

	case repos.Activated:

		if err := h.createSession(c, user.ID); err != nil {
			return h.redirectWithError(c, true)
		}

		return c.Redirect(http.StatusFound, os.Getenv("FRONTEND_URL"))

	default:
		return h.redirectWithError(c, false)
	}
}

type usernameReq struct {
	Username string `json:"username" validate:"required,min=3,max=15,alphanum"`
}

func (h *AuthHandler) CompleteProfile(c echo.Context) error {

	ctx := c.Request().Context()

	u := &usernameReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, u, h.validate); err != nil {
		return err
	}

	profileToken, err := utils.GetCookie(c, "profile_completion_token")
	if err != nil {
		utils.ClearCookie(c, "profile_completion_token")
		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid profile token")
	}

	user, err := h.tokenRepo.GetUserForToken(ctx, repos.ScopeProfileCompletion, profileToken)
	if err != nil {
		switch {
		case errors.Is(err, repos.ErrRecordNotFound):
			utils.ClearCookie(c, "profile_completion_token")
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid profile token")
		default:
			return err
		}
	}

	if user.Status != repos.PendingProfile {
		utils.ClearCookie(c, "profile_completion_token")
		return echo.NewHTTPError(http.StatusConflict, "Profile already completed")
	}

	err = h.userRepo.CompleteProfile(ctx, user.ID, u.Username, user.Version)
	if err != nil {
		switch {
		case errors.Is(err, repos.ErrEditConflict):
			utils.ClearCookie(c, "profile_completion_token")
			return echo.NewHTTPError(http.StatusConflict, "Profile modified by another request")
		case errors.Is(err, repos.ErrUniqueViolation):
			return c.JSON(http.StatusUnprocessableEntity, utils.ErrorResponse{
				Errors: []utils.ValidationError{{Field: "username", Message: "Username already taken"}},
			})
		default:
			return err
		}
	}

	// Authenticate token
	if err = h.createSession(c, user.ID); err != nil {
		return err
	}

	utils.ClearCookie(c, "profile_completion_token")
	return c.JSON(http.StatusOK, utils.Envelope{"message": "successfully authenticated"})

}

type registerUserReq struct {
	Email    string `json:"email" validate:"required,email"`
	Username string `json:"username" validate:"required,min=3,alphanum"`
	Password string `json:"password" validate:"required,min=8,max=49"`
}

func (h *AuthHandler) RegisterUserByEmail(c echo.Context) error {

	ctx := c.Request().Context()
	r := &registerUserReq{}

	if err := utils.ParseAndValidateJSON(c.Request().Body, r, h.validate); err != nil {
		return err
	}

	emailTaken, err := h.userRepo.EmailTaken(ctx, r.Email)
	if err != nil {
		return err
	}

	if emailTaken {
		return c.JSON(http.StatusUnprocessableEntity, utils.ErrorResponse{
			Errors: []utils.ValidationError{{Field: "email", Message: "Email already taken"}},
		})
	}

	// Check availability for username and email
	usernameTaken, err := h.userRepo.UsernameTaken(ctx, r.Username)
	if err != nil {
		return err
	}

	if usernameTaken {
		return c.JSON(http.StatusUnprocessableEntity, utils.ErrorResponse{
			Errors: []utils.ValidationError{{Field: "username", Message: "Username already taken"}},
		})
	}

	// Hash password
	passwordHash, err := repos.GetHashedPassword(r.Password)
	if err != nil {
		return fmt.Errorf("failed to hash password:%w", err)
	}

	// Try to insert

	user := &repos.User{
		Email:          r.Email,
		Status:         repos.PendingEmailVerification,
		PasswordHash:   passwordHash,
		ProviderID:     repos.CredentialsProvider,
		ProviderUserID: r.Email,
	}

	user.Username = sql.NullString{String: r.Username, Valid: true}

	if err = h.userRepo.Insert(ctx, user); err != nil {
		return fmt.Errorf("failed to insert user:%w", err)
	}

	if err = h.createSession(c, user.ID); err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, utils.Envelope{"message": "user successfully registered, pending email verification"})

}

type authenticationUserReq struct {
	Login    string `json:"login" validate:"required"` // can be email or username
	Password string `json:"password" validate:"required"`
}

var dummyHash = []byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcA/3Qz3hC5bDjq8s9tcRfWxE7.")

func (h *AuthHandler) LoginUserByEmailOrUsername(c echo.Context) error {

	ctx := c.Request().Context()

	r := &authenticationUserReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, r, h.validate); err != nil {
		return err
	}

	user, err := h.userRepo.GetByEmailOrUsername(ctx, r.Login)

	hashToCompare := dummyHash
	var userID uuid.UUID

	if err == nil {
		hashToCompare = user.PasswordHash
		userID = user.ID
	} else if !errors.Is(err, repos.ErrRecordNotFound) {
		return err
	}

	match, err := repos.MatchPassword(hashToCompare, r.Password)
	if err != nil {
		return err
	}

	if userID == uuid.Nil || !match {
		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid Login or Password")
	}

	if err = h.createSession(c, userID); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, utils.Envelope{"message": "successfully authenticated"})

}

func (h *AuthHandler) Logout(c echo.Context) error {

	ctx := c.Request().Context()
	user := utils.ContextGetUser(c)
	err := h.sessionRepo.RevokeSession(ctx, user.SessionID, user.ID)
	if err != nil && !errors.Is(err, repos.ErrRecordNotFound) {
		fmt.Println("failed to revoke session:", err)
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	utils.ClearCookie(c, "sid")
	return c.JSON(http.StatusOK, utils.Envelope{"message": "successfully logged out"})

}

type sessionsReq struct {
	ShowInactive bool `query:"showInactive"`
}

type sessionRes struct {
	ID         uuid.UUID `json:"id"`
	LastUsedAt time.Time `json:"lastUsedAt"`
	OS         string    `json:"os,omitempty"`
	Broswer    string    `json:"broswer,omitempty"`
	Device     string    `json:"device,omitempty"`
	IP         string    `json:"ip,omitempty"`
	Country    string    `json:"country,omitempty"`
	City       string    `json:"city,omitempty"`
	Active     bool      `json:"active"`
	Current    bool      `json:"current"`
}

func (h *AuthHandler) GetSessions(c echo.Context) error {

	r := &sessionsReq{}
	if err := utils.ParseAndValidateQueryParams(c, r, h.validate); err != nil {
		return err
	}

	ctx := c.Request().Context()
	user := utils.ContextGetUser(c)

	sessions, err := h.sessionRepo.GetForUser(ctx, user.ID, r.ShowInactive)
	if err != nil {
		return fmt.Errorf("failed to get active sessions: %w", err)
	}

	currentSessionID := user.SessionID
	sessResp := make([]*sessionRes, 0, len(sessions))
	for _, s := range sessions {
		rs := &sessionRes{
			ID:         s.ID,
			LastUsedAt: s.LastUsedAt,
		}
		if s.ClientOS.Valid {
			rs.OS = s.ClientOS.String
		}
		if s.ClientDevice.Valid {
			rs.Device = s.ClientDevice.String
		}
		if s.ClientBrowser.Valid {
			rs.Broswer = s.ClientBrowser.String
		}
		if s.IPLast.Valid {
			rs.IP = s.IPLast.String
		}

		if s.GeoCountry.Valid {
			rs.Country = s.GeoCountry.String
		}
		if s.GeoCity.Valid {
			rs.City = s.GeoCity.String
		}
		if s.ID == currentSessionID {
			rs.Current = true
		}

		rs.Active = !s.RevokedAt.Valid && s.ExpiresAt.After(time.Now())

		sessResp = append(sessResp, rs)

	}

	return c.JSON(http.StatusOK, sessResp)

}

type revokeSessionReq struct {
	SessionID uuid.UUID `param:"id" validate:"required"`
}

func (h *AuthHandler) RevokeSession(c echo.Context) error {

	r := &revokeSessionReq{}
	if err := utils.ParseAndValidadePathParams(c, r, h.validate); err != nil {
		return err
	}

	ctx := c.Request().Context()
	user := utils.ContextGetUser(c)

	err := h.sessionRepo.RevokeSession(ctx, r.SessionID, user.ID)
	if err != nil {
		switch {
		case errors.Is(err, repos.ErrRecordNotFound):
			return c.JSON(http.StatusNotFound, utils.ErrorResponse{Message: "Session not found"})
		default:
			return fmt.Errorf("failed to revoke session: %w", err)
		}
	}

	return c.JSON(http.StatusOK, utils.Envelope{"message": "session succesfully revoked"})

}

type passwordChangeReq struct {
	CurrentPassword string `json:"currentPassword" validate:"required"`
	NewPassword     string `json:"newPassword" validate:"required,min=8,max=49"`
}

func (h *AuthHandler) ChangePassword(c echo.Context) error {

	r := &passwordChangeReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, r, h.validate); err != nil {
		return err
	}

	userID := utils.ContextGetUser(c).ID
	ctx := c.Request().Context()

	user, err := h.userRepo.GetByID(ctx, userID)

	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	match, err := repos.MatchPassword(user.PasswordHash, r.CurrentPassword)
	if err != nil {
		return err
	}

	if !match {
		return c.JSON(http.StatusUnauthorized, utils.ErrorResponse{
			Errors: []utils.ValidationError{{Field: "currentPassword", Message: "Password is incorrect"}},
		})
	}

	newPasswordHash, err := repos.GetHashedPassword(r.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password:%w", err)
	}

	user.PasswordHash = newPasswordHash

	if err := h.userRepo.ChangePassword(ctx, user); err != nil {
		switch {
		case errors.Is(err, repos.ErrEditConflict):
			return echo.NewHTTPError(http.StatusConflict, "User updated while attempting to change password.")
		default:
			return fmt.Errorf("failed to change password: %w", err)
		}
	}

	return c.JSON(http.StatusAccepted, utils.Envelope{"message": "password successfully updated"})

}

type passwordSetReq struct {
	Password string `json:"password" validate:"required,min=8,max=49"`
}

func (h *AuthHandler) SetPassword(c echo.Context) error {

	r := &passwordSetReq{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, r, h.validate); err != nil {
		return err
	}

	userID := utils.ContextGetUser(c).ID
	ctx := c.Request().Context()

	user, err := h.userRepo.GetByID(ctx, userID)

	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	passwordHash, err := repos.GetHashedPassword(r.Password)
	if err != nil {
		return fmt.Errorf("failed to hash password:%w", err)
	}

	user.PasswordHash = passwordHash

	if err := h.userRepo.ChangePassword(ctx, user); err != nil {
		switch {
		case errors.Is(err, repos.ErrEditConflict):
			return echo.NewHTTPError(http.StatusConflict, "User updated while attempting to set password.")
		default:
			return fmt.Errorf("failed to set password: %w", err)
		}
	}

	return c.JSON(http.StatusAccepted, utils.Envelope{"message": "password successfully set"})

}

func (h *AuthHandler) getOrCreateUser(ctx context.Context, provider repos.AuthProvider, userData *UserData) (*repos.User, error) {
	user, err := h.userRepo.GetBySubProvider(ctx, userData.Sub, provider)
	if err != nil {
		if errors.Is(err, repos.ErrRecordNotFound) {
			// Create new user
			user = &repos.User{
				Email:          userData.Email,
				ProviderID:     provider,
				ProviderUserID: userData.Sub,
				Status:         repos.PendingProfile,
			}

			if userData.Picture != nil {
				user.ProfileImageKey = sql.NullString{String: *userData.Picture, Valid: true}
			}

			if err := h.userRepo.Insert(ctx, user); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return user, nil
}

func (h *AuthHandler) redirectWithError(c echo.Context, showError bool) error {
	baseURL := os.Getenv("FRONTEND_URL")
	if showError {
		return c.Redirect(http.StatusTemporaryRedirect, baseURL+"?error=authentication_failed")
	}
	return c.Redirect(http.StatusTemporaryRedirect, baseURL)
}

func (h *AuthHandler) createSession(c echo.Context, userID uuid.UUID) error {

	ctx := c.Request().Context()

	ipRaw := c.RealIP()
	var ip, country, city string
	if geoData, err := h.geoService.GetGeoDataFromIp(ipRaw); err == nil {
		ip = ipRaw
		country = geoData.Country
		city = geoData.City
	}

	uaRaw := c.Request().UserAgent()
	if len(uaRaw) > 512 {
		uaRaw = uaRaw[:512]
	}

	// Parse userAgent
	ua := useragent.Parse(uaRaw)
	// TODO GEO location from IP
	sessionPlain, session, err := repos.GenerateSession(userID, 14*24*time.Hour, ip, uaRaw, ua.OS, ua.Name, ua.Device, country, city)
	if err != nil {
		return fmt.Errorf("failed to generate session: %w", err)
	}

	// Store session in DB
	if err := h.sessionRepo.CreateSession(ctx, session); err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// go func() {
	// 	bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// 	defer cancel()
	// 	if err := h.sessionRepo.LimitSessionsUser(bgCtx, userID, 5); err != nil {
	// 		h.logger.Error("failed to limit sessions for user", "error", err)
	// 	}
	// }()

	// Set session cookie
	utils.SetSecureCookie(c, "sid", sessionPlain, 14*24*time.Hour)
	return nil
}

func generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func validateProvider(provider string) (repos.AuthProvider, error) {
	switch repos.AuthProvider(provider) {
	case repos.GoogleProvider, repos.TwitchProvider:
		return repos.AuthProvider(provider), nil
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}
}
