package main

import (
	"context"
	"fmt"
	"onepixel_backend/src/config"
	"onepixel_backend/src/db"
	"onepixel_backend/src/server"
	"onepixel_backend/src/utils/applogger"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/samber/lo"
)

func main() {
	// Initialize the database
	appDb := lo.Must(db.GetDB())

	// Create the app
	adminApp := server.CreateAdminApp(appDb)
	mainApp := server.CreateMainApp(appDb)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		host := strings.Split(c.Hostname(), ":")[0]
		switch host {
		case config.AdminHost:
			adminApp.Handler()(c.Context())
			return nil
		case config.MainHost:
			mainApp.Handler()(c.Context())
			return nil
		default:
			c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"message": "called via unsupported host",
			})
			return nil
		}
	})

	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 30 seconds.
	// Use a buffered channel to avoid missing signals as recommended for signal.Notify
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM)
	signal.Notify(quit, syscall.SIGINT)

	go func() {
		<-quit
		applogger.Info("Server is shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		lo.Must0(lo.Must(appDb.DB()).Close())
		lo.Must0(adminApp.ShutdownWithContext(ctx))
		lo.Must0(mainApp.ShutdownWithContext(ctx))
		lo.Must0(app.ShutdownWithContext(ctx))
	}()

	applogger.Fatal(app.Listen(fmt.Sprintf(":%s", config.Port)))
}
