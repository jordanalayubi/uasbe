package route

import (
	"UASBE/app/service"
	"UASBE/middleware"

	"github.com/gofiber/fiber/v2"
)

func SetupAdminRoutes(app *fiber.App, authService *service.AuthService) {
	admin := app.Group("/api/admin")
	
	// Apply auth middleware and admin-only middleware
	admin.Use(middleware.AuthMiddleware(authService))
	admin.Use(middleware.AdminOnlyMiddleware())

	// Admin dashboard
	admin.Get("/dashboard", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"success": true,
			"message": "Welcome to admin dashboard",
			"user": c.Locals("username"),
			"role": c.Locals("role"),
		})
	})

	// Get all users (admin only)
	admin.Get("/users", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"success": true,
			"message": "List of all users",
			"data": []string{"This would contain user list"},
		})
	})

	// System statistics (admin only)
	admin.Get("/stats", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"success": true,
			"data": fiber.Map{
				"total_users": 100,
				"total_achievements": 250,
				"pending_verifications": 15,
			},
		})
	})
}