package route

import (
	"UASBE/app/service"
	"UASBE/middleware"

	"github.com/gofiber/fiber/v2"
)

func SetupUserRoutes(app *fiber.App, userService *service.UserService, authService *service.AuthService) {
	api := app.Group("/api/users")
	
	// Apply auth middleware to all user routes
	api.Use(middleware.AuthMiddleware(authService))

	// FR-009: Manage Users - Admin only endpoints
	
	// Get all users
	api.Get("/", 
		middleware.AdminOnlyMiddleware(),
		userService.GetAllUsersRequest)

	// Get user by ID
	api.Get("/:id", 
		middleware.AdminOnlyMiddleware(),
		userService.GetUserByIDRequest)

	// Create new user
	api.Post("/", 
		middleware.AdminOnlyMiddleware(),
		userService.CreateUserRequest)

	// Update user
	api.Put("/:id", 
		middleware.AdminOnlyMiddleware(),
		userService.UpdateUserRequest)

	// Delete user
	api.Delete("/:id", 
		middleware.AdminOnlyMiddleware(),
		userService.DeleteUserRequest)
}