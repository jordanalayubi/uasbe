package route

import (
	"UASBE/app/service"
	"UASBE/middleware"

	"github.com/gofiber/fiber/v2"
)

func SetupAchievementRoutes(app *fiber.App, achievementService *service.AchievementService, authService *service.AuthService) {
	api := app.Group("/api/achievements")
	
	// Apply auth middleware to all achievement routes
	api.Use(middleware.AuthMiddleware(authService))

	// CRUD Operations for Students - Full access to their own achievements
	// Create achievement - students can create their own achievements
	api.Post("/", 
		middleware.PermissionMiddleware(authService, "achievements", "create"),
		achievementService.HandleCreateAchievementRequest)

	// FR-003: Submit Prestasi - Alternative endpoint for students
	api.Post("/submit", 
		middleware.PermissionMiddleware(authService, "achievements", "create"),
		achievementService.HandleCreateAchievementRequest)

	// Read operations - students can view their own achievements
	api.Get("/", 
		middleware.PermissionMiddleware(authService, "achievements", "read"),
		achievementService.HandleGetStudentAchievements)

	api.Get("/references", 
		middleware.PermissionMiddleware(authService, "achievements", "read"),
		achievementService.HandleGetStudentAchievementReferences)

	api.Get("/:id", 
		middleware.PermissionMiddleware(authService, "achievements", "read"),
		achievementService.HandleGetAchievementByID)

	// Update operations - students can update their own achievements
	api.Put("/:id", 
		middleware.PermissionMiddleware(authService, "achievements", "update"),
		achievementService.HandleUpdateAchievementRequest)

	// Delete operations - students can delete their own achievements
	api.Delete("/:id", 
		middleware.PermissionMiddleware(authService, "achievements", "delete"),
		achievementService.HandleDeleteAchievementRequest)

	// Submit achievement for verification - students can submit their achievements
	api.Post("/:achievement_id/submit", 
		middleware.PermissionMiddleware(authService, "achievements", "update"),
		achievementService.HandleSubmitAchievementRequest)

	// File upload routes - students can upload attachments
	upload := api.Group("/upload")
	upload.Post("/attachment", 
		middleware.PermissionMiddleware(authService, "achievements", "create"),
		achievementService.HandleUploadAttachment)

	// Lecturer routes for verification
	lecturer := api.Group("/verify")

	// Get pending verifications for lecturer
	lecturer.Get("/pending", 
		middleware.PermissionMiddleware(authService, "achievements", "verify"),
		achievementService.HandleGetPendingVerifications)

	// Verify achievement
	lecturer.Post("/:reference_id", 
		middleware.PermissionMiddleware(authService, "achievements", "verify"),
		achievementService.HandleVerifyAchievementRequest)
}