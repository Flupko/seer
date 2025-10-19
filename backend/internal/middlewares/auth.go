package middlewares

import (
	"errors"
	"fmt"
	"net/http"
	"seer/internal/repos"
	"seer/internal/utils"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type AuthMiddleware struct {
	sessionRepo *repos.SessionRepo
	validate    *validator.Validate
}

func NewAuthMiddleware(sessionRepo *repos.SessionRepo, validate *validator.Validate) *AuthMiddleware {
	return &AuthMiddleware{
		sessionRepo: sessionRepo,
		validate:    validate,
	}
}

func (am *AuthMiddleware) Authenticate(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {

		sessionCookie, err := c.Cookie("sid")

		if err != nil || sessionCookie.Value == "" {
			c.Set(string(utils.UserContextKey), repos.AnonymousUser)
			return next(c)
		}

		sessionPlain := sessionCookie.Value
		fmt.Println("Session plain:", sessionPlain)
		if err = am.validate.Var(sessionPlain, "token_plain"); err != nil {
			fmt.Println("Session plain:", sessionPlain, "error:", err)
			var validateErrs validator.ValidationErrors
			switch {
			case errors.As(err, &validateErrs):
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid session")
			default:
				return err
			}
		}

		var ip *string
		if ipRaw := c.RealIP(); utils.ValidateIp(ipRaw) {
			ip = &ipRaw
		}

		user, err := am.sessionRepo.GetUserFromPlain(c.Request().Context(), sessionPlain, ip)
		if err != nil {
			switch {
			case errors.Is(err, repos.ErrRecordNotFound):
				c.Set(string(utils.UserContextKey), repos.AnonymousUser)
				return next(c)
			default:
				return err
			}
		}

		c.Set(string(utils.UserContextKey), user)
		return next(c)
	}
}

func (am *AuthMiddleware) RequireAuthentication(next echo.HandlerFunc) echo.HandlerFunc {

	require := func(c echo.Context) error {

		user := utils.ContextGetUser(c)

		if user == repos.AnonymousUser {
			return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
		}

		// if user.Status != repos.Activated {
		// 	return echo.NewHTTPError(http.StatusUnauthorized, "unactivated account")
		// }

		return next(c)
	}

	return am.Authenticate(require)
}

func (am *AuthMiddleware) RequireRole(next echo.HandlerFunc, allowedRole repos.Role) echo.HandlerFunc {

	authorize := func(c echo.Context) error {

		user := utils.ContextGetUser(c)

		if user.Role != allowedRole {
			return echo.NewHTTPError(http.StatusForbidden, "your user account doesn't have the necessary permissions to access this resource")
		}

		return next(c)

	}

	return am.RequireAuthentication(authorize)

}
