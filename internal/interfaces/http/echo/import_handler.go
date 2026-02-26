package echo

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	app "github.com/mohammadpnp/user-import/internal/application/user"
)

type ImportHandler struct {
	useCase app.StartImportUsersFromJSON
}

type importUsersRequest struct {
	SourcePath string `json:"source_path"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type apiResponse struct {
	Data  any        `json:"data,omitempty"`
	Error *errorBody `json:"error,omitempty"`
}

func NewImportHandler(useCase app.StartImportUsersFromJSON) *ImportHandler {
	return &ImportHandler{useCase: useCase}
}

func (h *ImportHandler) ImportUsers(c echo.Context) error {
	var req importUsersRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, apiResponse{Error: &errorBody{
			Code:    "bad_request",
			Message: "invalid request body",
		}})
	}

	out, err := h.useCase.Execute(c.Request().Context(), app.StartImportUsersFromJSONInput{
		SourcePath: req.SourcePath,
	})
	if err != nil {
		if errors.Is(err, app.ErrInvalidImportSource) {
			return c.JSON(http.StatusBadRequest, apiResponse{Error: &errorBody{
				Code:    "invalid_source",
				Message: "source_path must be a .json file",
			}})
		}
		return c.JSON(http.StatusInternalServerError, apiResponse{Error: &errorBody{
			Code:    "internal_error",
			Message: "failed to enqueue import job",
		}})
	}

	return c.JSON(http.StatusAccepted, apiResponse{Data: out})
}
