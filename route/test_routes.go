package route

import (
	"UASBE/app/service"
	"UASBE/middleware"

	"github.com/gofiber/fiber/v2"
)

func SetupTestRoutes(app *fiber.App, authService *service.AuthService) {
	test := app.Group("/api/test")
	
	// Apply auth middleware
	test.Use(middleware.AuthMiddleware(authService))

	// Test endpoint for students only
	test.Get("/student-only", 
		middleware.RoleMiddleware("student"),
		func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{
				"success": true,
				"message": "This endpoint is for students only",
				"user": c.Locals("username"),
				"role": c.Locals("role"),
			})
		})

	// Test endpoint for lecturers only
	test.Get("/lecturer-only", 
		middleware.RoleMiddleware("lecturer"),
		func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{
				"success": true,
				"message": "This endpoint is for lecturers only",
				"user": c.Locals("username"),
				"role": c.Locals("role"),
			})
		})

	// Test endpoint for lecturers and admins
	test.Get("/lecturer-or-admin", 
		middleware.LecturerOrAdminMiddleware(),
		func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{
				"success": true,
				"message": "This endpoint is for lecturers and admins",
				"user": c.Locals("username"),
				"role": c.Locals("role"),
			})
		})

	// Test endpoint with specific permission
	test.Get("/manage-users", 
		middleware.PermissionMiddleware(authService, "users", "manage"),
		func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{
				"success": true,
				"message": "This endpoint requires manage users permission",
				"user": c.Locals("username"),
				"role": c.Locals("role"),
				"permissions": c.Locals("permissions"),
			})
		})

	// Test endpoint to show user info
	test.Get("/me", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"success": true,
			"user_id": c.Locals("user_id"),
			"username": c.Locals("username"),
			"role": c.Locals("role"),
			"role_id": c.Locals("role_id"),
			"permissions": c.Locals("permissions"),
		})
	})
}