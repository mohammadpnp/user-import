package echo

import e "github.com/labstack/echo/v4"

func RegisterRoutes(server *e.Echo, importHandler *ImportHandler, userHandler *UserHandler) {
	if importHandler != nil {
		server.POST("/api/v1/imports/users", importHandler.ImportUsers)
	}
	if userHandler != nil {
		server.GET("/api/v1/users/:id", userHandler.GetUserByID)
	}
}
