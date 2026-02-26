package bootstrap

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	app "github.com/mohammadpnp/user-import/internal/application/user"
	"github.com/mohammadpnp/user-import/internal/infrastructure/repository"
	httpecho "github.com/mohammadpnp/user-import/internal/interfaces/http/echo"
	"gorm.io/gorm"
)

func NewHTTPServer(db *gorm.DB) *echo.Echo {
	server := echo.New()
	server.HideBanner = true

	server.Use(middleware.Recover())
	server.Use(middleware.RequestID())
	server.Use(middleware.BodyLimit("10M"))

	importJobRepo := repository.NewImportJobRepository(db)
	startImport := app.NewStartImportUsersFromJSON(importJobRepo)
	importHandler := httpecho.NewImportHandler(startImport)
	userQueryRepo := repository.NewUserQueryRepository(db)
	getUserByID := app.NewGetUserByID(userQueryRepo)
	userHandler := httpecho.NewUserHandler(getUserByID)

	httpecho.RegisterRoutes(server, importHandler, userHandler)

	server.GET("/healthz", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	return server
}
