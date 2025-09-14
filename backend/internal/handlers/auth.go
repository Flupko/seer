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
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Picture       string `json:"picture"`
	Sub           string `json:"sub"`
}

type AuthHandler struct {
	validate    *validator.Validate
	logger      *slog.Logger
	providers   map[repos.AuthProvider]*ProviderConfig
	userRepo    UserRepository
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
	userRepo UserRepository,
	sessionRepo *repos.SessionRepo,
	tokenRepo *repos.TokenRepo,
	geoService *geo.GeoService) (*AuthHandler, error) {

	// Setup Google provider
	googleProvider, err := oidc.NewProvider(ctx, "https://accounts.google.com")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Google provider: %w", err)
	}

	// Setup Twitch provider
	twitchProvider, err := oidc.NewProvider(ctx, "https://id.twitch.tv/oauth2")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Twitch provider: %w", err)
	}

	providers := map[repos.AuthProvider]*ProviderConfig{
		repos.GoogleProvider: {
			oauth2: &oauth2.Config{
				ClientID:     os.Getenv("GOOGLE_OAUTH_CLIENT_ID"),
				ClientSecret: os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
				RedirectURL:  fmt.Sprintf("%s/auth/%s/callback", os.Getenv("API_BASE_URL"), repos.GoogleProvider),
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
				RedirectURL:  fmt.Sprintf("%s/auth/%s/callback", os.Getenv("API_BASE_URL"), repos.TwitchProvider),
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
	return c.Redirect(http.StatusFound, authURL)
}

func (h *AuthHandler) GetAuthCallback(c echo.Context) error {
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
	storedState, err := h.getAndClearCookie(c, cookieName)
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
		h.setSecureCookie(c, "profile_completion_token", tokenPlain, 5*time.Minute)

		redirectURL := fmt.Sprintf("%s/?SHOW=AUTH_COMPLETION", os.Getenv("FRONTEND_URL"))
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
	if err := utils.BindAndValidate(c, u, h.validate); err != nil {
		return err
	}

	profileToken, err := h.getAndClearCookie(c, "profile_completion_token")
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid profile token")
	}

	user, err := h.tokenRepo.GetUserForToken(ctx, repos.ScopeProfileCompletion, profileToken)
	if err != nil {
		switch {
		case errors.Is(err, repos.ErrRecordNotFound):
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid profile token")
		default:
			return err
		}
	}

	if user.Status != repos.PendingProfile {
		return echo.NewHTTPError(http.StatusConflict, "profile already completed")
	}

	err = h.userRepo.CompleteProfile(ctx, user.ID, u.Username, user.Version)
	if err != nil {
		switch {
		case errors.Is(err, repos.ErrEditConflict):
			return echo.NewHTTPError(http.StatusConflict, "profile modified by another request")
		case errors.Is(err, repos.ErrUniqueViolation):
			return echo.NewHTTPError(http.StatusConflict, "username already taken")
		default:
			return err
		}

	}

	// Authenticate token
	if err = h.createSession(c, user.ID); err != nil {
		return err
	}

	return c.Redirect(http.StatusFound, os.Getenv("FRONTEND_URL"))

}

type registerUserReq struct {
	Email    string `json:"email" validate:"required,email"`
	Username string `json:"username" validate:"required,alphanum"`
	Password string `json:"password" validate:"required,min=8,max=49"`
}

func (h *AuthHandler) RegisterUserByEmail(c echo.Context) error {

	ctx := c.Request().Context()

	r := &registerUserReq{}
	if err := utils.BindAndValidate(c, r, h.validate); err != nil {
		return err
	}

	// Check availability for username and email
	usernameTaken, err := h.userRepo.UsernameTaken(ctx, r.Username)
	if err != nil {
		return err
	}

	if usernameTaken {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "username already taken")
	}

	emailTaken, err := h.userRepo.EmailTaken(ctx, r.Email)
	if err != nil {
		return err
	}

	if emailTaken {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, "email already taken")
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

	return c.JSON(http.StatusCreated, utils.Envelope{"message": "user successfully registered, pending email verification"})

}

type authenticationUserReq struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

func (h *AuthHandler) LoginUserByEmail(c echo.Context) error {

	ctx := c.Request().Context()

	r := &authenticationUserReq{}
	if err := utils.BindAndValidate(c, r, h.validate); err != nil {
		return err
	}

	user, err := h.userRepo.GetByEmail(ctx, r.Email)
	if err != nil {
		switch {
		case errors.Is(err, repos.ErrRecordNotFound):
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
		default:
			return err
		}
	}

	match, err := repos.MatchPassword(user.PasswordHash, r.Password)
	if err != nil {
		return err
	}

	if !match {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}

	if user.Status != repos.Activated {
		return echo.NewHTTPError(http.StatusUnauthorized, "unactivated account")
	}

	if err = h.createSession(c, user.ID); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, utils.Envelope{"message": "successfully authenticated"})

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
			user.ProfileImageKey = sql.NullString{String: userData.Picture, Valid: true}

			if err := h.userRepo.Insert(ctx, user); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return user, nil
}

func (h *AuthHandler) setSecureCookie(c echo.Context, name, value string, maxAge time.Duration) {
	c.SetCookie(&http.Cookie{
		Name:     name,
		Value:    value,
		HttpOnly: true,
		Secure:   config.IsProduction, // Must be true in production (HTTPS)
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		MaxAge:   int(maxAge.Seconds()),
	})
}

func (h *AuthHandler) getAndClearCookie(c echo.Context, cookieName string) (string, error) {

	cookie, err := c.Cookie(cookieName)

	if err != nil {
		return "", err
	}

	// Clear the state cookie
	c.SetCookie(&http.Cookie{
		Name:     cookieName,
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	return cookie.Value, nil
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

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := h.sessionRepo.LimitSessionsUser(bgCtx, userID, 5); err != nil {
			h.logger.Error("failed to limit sessions for user", "error", err)
		}
	}()

	// Set session cookie
	h.setSecureCookie(c, "sid", sessionPlain, 14*24*time.Hour)
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
