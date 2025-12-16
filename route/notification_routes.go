package route

import (
	"UASBE/app/service"
	"UASBE/middleware"

	"github.com/gofiber/fiber/v2"
)

func SetupNotificationRoutes(app *fiber.App, notificationService *service.NotificationService, authService *service.AuthService) {
	api := app.Group("/api/notifications")
	
	// Apply auth middleware to all notification routes
	api.Use(middleware.AuthMiddleware(authService))

	// Get user notifications
	api.Get("/", notificationService.GetNotificationsRequest)

	// Get unread notification count
	api.Get("/unread-count", notificationService.GetUnreadCountRequest)

	// Mark notification as read
	api.Put("/:id/read", notificationService.MarkAsReadRequest)
}