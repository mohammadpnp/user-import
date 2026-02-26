package echo

import e "github.com/labstack/echo/v4"

func RegisterRoutes(server *e.Echo, importHandler *ImportHandler) {
	server.POST("/api/v1/imports/users", importHandler.ImportUsers)
}
