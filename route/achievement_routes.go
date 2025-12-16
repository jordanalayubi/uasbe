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
		achievementService.CreateAchievementRequest)

	// FR-003: Submit Prestasi - Alternative endpoint for students
	api.Post("/submit", 
		middleware.PermissionMiddleware(authService, "achievements", "create"),
		achievementService.CreateAchievementRequest)

	// Read operations - students can view their own achievements
	api.Get("/", 
		middleware.PermissionMiddleware(authService, "achievements", "read"),
		achievementService.GetStudentAchievementsRequest)

	api.Get("/references", 
		middleware.PermissionMiddleware(authService, "achievements", "read"),
		achievementService.GetStudentAchievementReferencesRequest)

	// FR-006: View Prestasi Mahasiswa Bimbingan - Dosen wali melihat prestasi mahasiswa bimbingannya
	// IMPORTANT: This route must be BEFORE /:id route to avoid route conflict
	api.Get("/advisee", 
		middleware.PermissionMiddleware(authService, "achievements", "view_advisee"),
		achievementService.GetAdviseeAchievementsRequest)

	api.Get("/:id", 
		middleware.PermissionMiddleware(authService, "achievements", "read"),
		achievementService.GetAchievementByIDRequest)

	// Update operations - students can update their own achievements
	api.Put("/:id", 
		middleware.PermissionMiddleware(authService, "achievements", "update"),
		achievementService.UpdateAchievementRequest)

	// Delete operations - students can delete their own achievements
	api.Delete("/:id", 
		middleware.PermissionMiddleware(authService, "achievements", "delete"),
		achievementService.DeleteAchievementRequest)

	// Submit achievement for verification - students can submit their achievements
	api.Post("/:achievement_id/submit", 
		middleware.PermissionMiddleware(authService, "achievements", "update"),
		achievementService.SubmitAchievementRequest)

	// File upload routes - students can upload attachments
	upload := api.Group("/upload")
	upload.Post("/attachment", 
		middleware.PermissionMiddleware(authService, "achievements", "create"),
		achievementService.UploadAttachmentRequest)

	// Lecturer routes for verification
	lecturer := api.Group("/verify")

	// Get pending verifications for lecturer
	lecturer.Get("/pending", 
		middleware.PermissionMiddleware(authService, "achievements", "verify"),
		achievementService.GetPendingVerificationsRequest)

	// FR-007: Get verification detail - dosen review prestasi detail
	lecturer.Get("/:reference_id", 
		middleware.PermissionMiddleware(authService, "achievements", "verify"),
		achievementService.GetVerificationDetailRequest)

	// FR-007: Verify achievement - dosen approve/reject prestasi
	lecturer.Post("/:reference_id", 
		middleware.PermissionMiddleware(authService, "achievements", "verify"),
		achievementService.VerifyAchievementRequest)

	// FR-010: Admin routes - View All Achievements
	admin := api.Group("/admin")
	
	// FR-010: Get all achievements with filters and pagination - admin only
	admin.Get("/all", 
		middleware.AdminOnlyMiddleware(),
		achievementService.GetAllAchievementsRequest)

	// FR-011: Achievement Statistics - Role-based statistics
	stats := api.Group("/statistics")
	
	// FR-011: Get achievement statistics based on user role
	// - Student: own statistics
	// - Lecturer: advisee statistics  
	// - Admin: all statistics
	stats.Get("/", 
		achievementService.GetAchievementStatisticsRequest)
}