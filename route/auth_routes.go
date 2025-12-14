package route

import (
	"UASBE/app/service"

	"github.com/gofiber/fiber/v2"
)

func SetupAuthRoutes(app *fiber.App, authService *service.AuthService) {
	auth := app.Group("/api/auth")

	// Login
	auth.Post("/login", authService.HandleLoginRequest)

	// Register
	auth.Post("/register", authService.HandleRegisterRequest)
}