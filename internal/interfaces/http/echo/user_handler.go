package echo

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	app "github.com/mohammadpnp/user-import/internal/application/user"
)

type UserHandler struct {
	useCase app.GetUserByID
}

func NewUserHandler(useCase app.GetUserByID) *UserHandler {
	return &UserHandler{useCase: useCase}
}

func (h *UserHandler) GetUserByID(c echo.Context) error {
	out, err := h.useCase.Execute(c.Request().Context(), app.GetUserByIDInput{
		ID: c.Param("id"),
	})
	if err != nil {
		if errors.Is(err, app.ErrInvalidUserID) {
			return c.JSON(http.StatusBadRequest, apiResponse{Error: &errorBody{
				Code:    "invalid_user_id",
				Message: "id must be a valid UUID",
			}})
		}
		if errors.Is(err, app.ErrUserNotFound) {
			return c.JSON(http.StatusNotFound, apiResponse{Error: &errorBody{
				Code:    "not_found",
				Message: "user not found",
			}})
		}

		return c.JSON(http.StatusInternalServerError, apiResponse{Error: &errorBody{
			Code:    "internal_error",
			Message: "failed to get user",
		}})
	}

	return c.JSON(http.StatusOK, apiResponse{Data: out})
}
