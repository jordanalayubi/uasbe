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
	
	// Get available advisors (lecturers)
	api.Get("/advisors", 
		middleware.AdminOnlyMiddleware(),
		userService.GetAvailableAdvisorsRequest)
	
	// Create default lecturer for testing
	api.Post("/create-default-lecturer", 
		middleware.AdminOnlyMiddleware(),
		userService.CreateDefaultLecturerRequest)
	
	// Debug users
	api.Get("/debug", 
		middleware.AdminOnlyMiddleware(),
		userService.DebugUsersRequest)
	
	// Debug specific user role
	api.Get("/debug-role/:user_id", 
		middleware.AdminOnlyMiddleware(),
		userService.DebugUserRoleRequest)
	
	// Fix database constraints
	api.Post("/fix-constraints", 
		middleware.AdminOnlyMiddleware(),
		userService.FixDatabaseConstraintsRequest)
	
	// Clean invalid data
	api.Post("/clean-invalid-data", 
		middleware.AdminOnlyMiddleware(),
		userService.CleanInvalidDataRequest)
	
	// Clean achievement references
	api.Post("/clean-achievement-references", 
		middleware.AdminOnlyMiddleware(),
		userService.CleanAchievementReferencesRequest)
	
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