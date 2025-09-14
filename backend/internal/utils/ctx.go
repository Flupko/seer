package utils

import (
	"seer/internal/repos"

	"github.com/labstack/echo/v4"
)

type ContextKey string

const (
	UserContextKey ContextKey = "user"
)

func ContextGetUser(c echo.Context) *repos.MinimalUser {

	user, ok := c.Get(string(UserContextKey)).(*repos.MinimalUser)

	if !ok {
		panic("invalid value pair")
	}

	return user
}
