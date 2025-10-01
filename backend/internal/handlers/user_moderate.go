package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"seer/internal/repos"
	"seer/internal/utils"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type UserModerateHandler struct {
	validate *validator.Validate
	mr       *repos.ModerateRepo
}

func NewUserModerateHandler(v *validator.Validate, mr *repos.ModerateRepo) *UserModerateHandler {
	return &UserModerateHandler{
		validate: v,
		mr:       mr,
	}
}

type reportCommentRequest struct {
	CommentID int64 `json:"commentId" validate:"required"`
}

func (h *UserModerateHandler) ReportComment(c echo.Context) error {

	r := &reportCommentRequest{}
	if err := utils.ParseAndValidateJSON(c.Request().Body, r, h.validate); err != nil {
		return err
	}

	user := utils.ContextGetUser(c)

	report := &repos.ReportComment{
		ReporterUserID: user.ID,
		CommentID:      r.CommentID,
	}

	if err := h.mr.ReportComment(c.Request().Context(), report); err != nil {
		switch {
		case errors.Is(err, repos.ErrRecordNotFound):
			return echo.NewHTTPError(http.StatusNotFound, "comment not found")
		case errors.Is(err, repos.ErrUniqueViolation):
			return echo.NewHTTPError(http.StatusConflict, "comment already reported by user")
		default:
			return fmt.Errorf("error reporting comment: %w", err)
		}
	}

	return c.JSON(http.StatusCreated, utils.Envelope{"message": "comment reported successfully"})

}
