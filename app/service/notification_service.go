package service

import (
	"UASBE/app/model"
	"UASBE/app/repository"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type NotificationService struct {
	notificationRepo *repository.NotificationRepository
}

func NewNotificationService(notificationRepo *repository.NotificationRepository) *NotificationService {
	return &NotificationService{
		notificationRepo: notificationRepo,
	}
}

// CreateNotification - Create a new notification
func (s *NotificationService) CreateNotification(userID, notificationType, title, message string, data map[string]interface{}) error {
	notification := &model.Notification{
		UserID:  userID,
		Type:    notificationType,
		Title:   title,
		Message: message,
		Data:    data,
	}

	return s.notificationRepo.Create(notification)
}

// CreateAchievementSubmittedNotification - Create notification when achievement is submitted
func (s *NotificationService) CreateAchievementSubmittedNotification(lecturerID, studentName, achievementTitle string) error {
	title := "New Achievement Submitted"
	message := studentName + " has submitted an achievement: " + achievementTitle
	data := map[string]interface{}{
		"student_name":      studentName,
		"achievement_title": achievementTitle,
		"action_required":   "verification",
	}

	return s.CreateNotification(lecturerID, "achievement_submitted", title, message, data)
}

// GetNotificationsRequest - Get user notifications
func (s *NotificationService) GetNotificationsRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	notifications, err := s.notificationRepo.GetByUserID(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get notifications",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    notifications,
	})
}

// MarkAsReadRequest - Mark notification as read
func (s *NotificationService) MarkAsReadRequest(c *fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid notification ID",
		})
	}

	err = s.notificationRepo.MarkAsRead(id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to mark notification as read",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Notification marked as read",
	})
}

// GetUnreadCountRequest - Get unread notification count
func (s *NotificationService) GetUnreadCountRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	count, err := s.notificationRepo.GetUnreadCount(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get unread count",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"unread_count": count,
		},
	})
}