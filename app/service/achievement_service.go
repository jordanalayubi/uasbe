package service

import (
	"UASBE/app/model"
	"UASBE/app/repository"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AchievementService struct {
	achievementRepo   *repository.AchievementRepository
	studentRepo       *repository.StudentRepository
	lecturerRepo      *repository.LecturerRepository
	notificationService *NotificationService
}

type CreateAchievementRequest struct {
	Category     string                 `json:"category"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	Details      map[string]interface{} `json:"details"`
	CustomFields []model.CustomField    `json:"custom_fields,omitempty"`
	Attachments  []model.Attachment     `json:"attachments,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
}

type VerifyAchievementRequest struct {
	Status        string `json:"status"` // "verified" or "rejected"
	RejectionNote string `json:"rejection_note,omitempty"`
}

func NewAchievementService(achievementRepo *repository.AchievementRepository, studentRepo *repository.StudentRepository, lecturerRepo *repository.LecturerRepository, notificationService *NotificationService) *AchievementService {
	return &AchievementService{
		achievementRepo:     achievementRepo,
		studentRepo:         studentRepo,
		lecturerRepo:        lecturerRepo,
		notificationService: notificationService,
	}
}

// FR-010: View All Achievements - Admin dapat melihat semua prestasi
// GetAllAchievementsRequest handles admin view of all achievements
// @Summary Get All Achievements (Admin)
// @Description Admin can view all achievements with filters and pagination (FR-010)
// @Tags Achievements
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param status query string false "Filter by status" Enums(draft, submitted, verified, rejected)
// @Param category query string false "Filter by category" Enums(competition, research, community_service, academic, organization)
// @Param student_id query string false "Filter by student ID"
// @Param sort_by query string false "Sort field" Enums(created_at, updated_at, title, category) default(created_at)
// @Param sort_order query string false "Sort order" Enums(asc, desc) default(desc)
// @Success 200 {object} map[string]interface{} "All achievements retrieved successfully"
// @Failure 403 {object} map[string]interface{} "Forbidden - Admin only"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /achievements/admin/all [get]
func (s *AchievementService) GetAllAchievementsRequest(c *fiber.Ctx) error {
	userRole := c.Locals("role").(string)
	
	// Only admin can view all achievements
	if userRole != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only admin can view all achievements",
		})
	}

	// Get query parameters for filtering and pagination
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	status := c.Query("status", "")
	category := c.Query("category", "")
	studentID := c.Query("student_id", "")
	sortBy := c.Query("sort_by", "created_at")
	sortOrder := c.Query("sort_order", "desc")

	// Validate pagination
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// FR-010 Step 1: Get all achievement references with filters
	references, total, err := s.achievementRepo.GetAllReferencesWithFilters(page, limit, status, studentID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get achievement references",
		})
	}

	if len(references) == 0 {
		return c.JSON(fiber.Map{
			"success": true,
			"data":    []interface{}{},
			"pagination": fiber.Map{
				"page":        page,
				"limit":       limit,
				"total":       0,
				"total_pages": 0,
			},
		})
	}

	// Extract achievement IDs
	var achievementIDs []string
	for _, ref := range references {
		achievementIDs = append(achievementIDs, ref.AchievementID)
	}

	// FR-010 Step 2: Fetch details dari MongoDB with filters and sorting
	achievements, err := s.achievementRepo.GetAchievementsByIDsWithFilters(achievementIDs, category, sortBy, sortOrder)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get achievement details",
		})
	}

	// Create map for quick lookup
	achievementMap := make(map[string]*model.Achievement)
	for i, achievement := range achievements {
		achievementMap[achievement.ID.Hex()] = &achievements[i]
	}

	// Get all student IDs for batch lookup (filter out empty strings)
	studentIDs := make(map[string]bool)
	for _, ref := range references {
		if ref.StudentID != "" && len(ref.StudentID) > 0 {
			studentIDs[ref.StudentID] = true
		}
	}

	// Get student info batch
	var studentIDList []string
	for id := range studentIDs {
		if id != "" && len(id) > 0 {
			studentIDList = append(studentIDList, id)
		}
	}

	students, err := s.studentRepo.GetStudentsByUserIDs(studentIDList)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": "Failed to get student information",
			"message": err.Error(),
			"debug_info": fiber.Map{
				"student_ids_requested": studentIDList,
				"total_references": len(references),
			},
		})
	}

	// Create student map for quick lookup
	studentMap := make(map[string]*model.Student)
	for i, student := range students {
		studentMap[student.UserID] = &students[i]
	}

	// FR-010 Step 4: Combine data and return dengan pagination
	var result []fiber.Map
	for _, ref := range references {
		achievement := achievementMap[ref.AchievementID]
		if achievement == nil {
			continue
		}

		student := studentMap[ref.StudentID]
		var studentInfo fiber.Map
		if student != nil {
			studentInfo = fiber.Map{
				"student_id":    student.StudentID,
				"full_name":     "Student Name", // Will be enhanced with user data
				"program_study": student.ProgramStudy,
				"academic_year": student.AcademicYear,
				"user_id":       student.UserID,
			}
		}

		// Calculate processing time if verified
		var processingTime string
		if ref.Status == "verified" && !ref.SubmittedAt.IsZero() && !ref.VerifiedAt.IsZero() {
			duration := ref.VerifiedAt.Sub(ref.SubmittedAt)
			if duration.Hours() < 1 {
				processingTime = fmt.Sprintf("%.0f minutes", duration.Minutes())
			} else if duration.Hours() < 24 {
				processingTime = fmt.Sprintf("%.1f hours", duration.Hours())
			} else {
				processingTime = fmt.Sprintf("%.1f days", duration.Hours()/24)
			}
		}

		// Enhanced reference info
		referenceInfo := fiber.Map{
			"id":           ref.ID.Hex(),
			"status":       ref.Status,
			"submitted_at": ref.SubmittedAt,
			"verified_at":  ref.VerifiedAt,
			"verified_by":  ref.VerifiedBy,
			"created_at":   ref.CreatedAt,
			"updated_at":   ref.UpdatedAt,
		}

		if processingTime != "" {
			referenceInfo["processing_time"] = processingTime
		}

		if ref.RejectionNote != "" {
			referenceInfo["rejection_note"] = ref.RejectionNote
		}

		result = append(result, fiber.Map{
			"achievement":  achievement,
			"reference":    referenceInfo,
			"student_info": studentInfo,
		})
	}

	// Calculate pagination
	totalPages := (total + limit - 1) / limit
	hasNextPage := page < totalPages
	hasPreviousPage := page > 1

	// Build next/previous URLs
	var nextPageURL, previousPageURL *string
	if hasNextPage {
		nextURL := fmt.Sprintf("/api/achievements/admin/all?page=%d&limit=%d", page+1, limit)
		if status != "" {
			nextURL += "&status=" + status
		}
		if category != "" {
			nextURL += "&category=" + category
		}
		nextPageURL = &nextURL
	}
	if hasPreviousPage {
		prevURL := fmt.Sprintf("/api/achievements/admin/all?page=%d&limit=%d", page-1, limit)
		if status != "" {
			prevURL += "&status=" + status
		}
		if category != "" {
			prevURL += "&category=" + category
		}
		previousPageURL = &prevURL
	}

	// Get statistics
	stats, _ := s.achievementRepo.GetAchievementStatistics()

	return c.JSON(fiber.Map{
		"success": true,
		"message": "All achievements retrieved successfully",
		"summary": fiber.Map{
			"total_achievements": total,
			"filtered_results":   len(result),
			"page_results":       len(result),
			"filters_applied": fiber.Map{
				"status":     status,
				"category":   category,
				"student_id": studentID,
			},
		},
		"data": result,
		"pagination": fiber.Map{
			"current_page":       page,
			"per_page":           limit,
			"total_items":        total,
			"total_pages":        totalPages,
			"has_next_page":      hasNextPage,
			"has_previous_page":  hasPreviousPage,
			"next_page_url":      nextPageURL,
			"previous_page_url":  previousPageURL,
		},
		"filters": fiber.Map{
			"applied": fiber.Map{
				"status":     status,
				"category":   category,
				"student_id": studentID,
				"sort_by":    sortBy,
				"sort_order": sortOrder,
			},
			"available": fiber.Map{
				"statuses":    []string{"draft", "submitted", "verified", "rejected", "deleted"},
				"categories":  []string{"competition", "research", "community_service", "academic", "organization"},
				"sort_fields": []string{"created_at", "updated_at", "title", "category"},
			},
		},
		"statistics": stats,
	})
}

// FR-003: Submit Prestasi - Service method untuk mahasiswa submit prestasi
// CreateAchievementRequest handles achievement creation
// @Summary Create Achievement
// @Description Student creates a new achievement (FR-003)
// @Tags Achievements
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param achievement body CreateAchievementRequest true "Achievement data"
// @Success 201 {object} map[string]interface{} "Achievement created successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Router /achievements [post]
func (s *AchievementService) CreateAchievementRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	userRole := c.Locals("role").(string)
	username := c.Locals("username").(string)
	
	// FR-003 Precondition: User terautentikasi sebagai mahasiswa
	if userRole != "student" && userRole != "Mahasiswa" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error": "Access denied",
			"message": "Only students can submit achievements",
			"code": "INSUFFICIENT_PERMISSIONS",
			"user_role": userRole,
			"required_role": "student",
		})
	}
	
	var req CreateAchievementRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": "Invalid request body",
			"message": "Please provide valid JSON data",
			"code": "INVALID_REQUEST_BODY",
		})
	}

	// FR-003 Step 1: Mahasiswa mengisi data prestasi - Enhanced validation
	validationErrors := make(map[string]string)
	if req.Category == "" {
		validationErrors["category"] = "Category is required"
	}
	if req.Title == "" {
		validationErrors["title"] = "Title is required"
	}
	if req.Description == "" {
		validationErrors["description"] = "Description is required"
	}
	
	// Validate category values
	validCategories := []string{"competition", "research", "community_service", "academic", "organization"}
	isValidCategory := false
	for _, cat := range validCategories {
		if req.Category == cat {
			isValidCategory = true
			break
		}
	}
	if req.Category != "" && !isValidCategory {
		validationErrors["category"] = "Invalid category. Valid options: " + fmt.Sprintf("%v", validCategories)
	}

	if len(validationErrors) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": "Validation failed",
			"message": "Please correct the following errors",
			"code": "VALIDATION_ERROR",
			"details": validationErrors,
			"valid_categories": validCategories,
		})
	}

	// Get student info for response
	student, err := s.studentRepo.GetByUserID(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": "Student profile not found",
			"message": "Unable to find student profile for this user",
			"code": "STUDENT_PROFILE_NOT_FOUND",
		})
	}

	// FR-003 Step 2: Mahasiswa upload dokumen pendukung (handled in request)
	// FR-003 Step 3: Sistem simpan ke MongoDB (achievement) dan PostgreSQL (reference)
	// FR-003 Step 4: Status awal: 'draft'
	achievement, reference, err := s.SubmitAchievement(userID, &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": "Failed to create achievement",
			"message": err.Error(),
			"code": "CREATION_FAILED",
		})
	}

	// FR-003 Step 5: Enhanced return achievement data
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Achievement created successfully",
		"code": "ACHIEVEMENT_CREATED",
		"data": fiber.Map{
			"achievement": fiber.Map{
				"id": achievement.ID.Hex(),
				"student_id": achievement.StudentID,
				"category": achievement.Category,
				"title": achievement.Title,
				"description": achievement.Description,
				"details": achievement.Details,
				"custom_fields": achievement.CustomFields,
				"attachments": achievement.Attachments,
				"tags": achievement.Tags,
				"created_at": achievement.CreatedAt,
				"updated_at": achievement.UpdatedAt,
			},
			"reference": fiber.Map{
				"id": reference.ID.Hex(),
				"student_id": reference.StudentID,
				"achievement_id": reference.AchievementID,
				"status": reference.Status,
				"created_at": reference.CreatedAt,
				"updated_at": reference.UpdatedAt,
			},
			"student_info": fiber.Map{
				"student_id": student.StudentID,
				"full_name": username,
				"program_study": student.ProgramStudy,
				"academic_year": student.AcademicYear,
			},
		},
		"next_steps": []string{
			"Your achievement has been saved as draft",
			"You can edit this achievement anytime while it's in draft status",
			"Submit for verification when you're ready: POST /api/achievements/" + achievement.ID.Hex() + "/submit",
			"Upload attachments if needed: POST /api/achievements/upload/attachment",
		},
		"status_info": fiber.Map{
			"current_status": "draft",
			"description": "Achievement is saved but not yet submitted for verification",
			"available_actions": []string{"edit", "delete", "submit_for_verification"},
		},
		"timestamp": time.Now(),
	})
}

func (s *AchievementService) GetStudentAchievementsRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	achievements, err := s.GetStudentAchievements(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    achievements,
	})
}

func (s *AchievementService) GetStudentAchievementReferencesRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	references, err := s.GetStudentAchievementReferences(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    references,
	})
}

func (s *AchievementService) GetAchievementByIDRequest(c *fiber.Ctx) error {
	idParam := c.Params("id")

	// Validate and convert ID
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid achievement ID",
		})
	}

	// Get active achievement (exclude soft deleted)
	achievement, err := s.GetActiveAchievementByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Achievement not found",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    achievement,
	})
}

func (s *AchievementService) UpdateAchievementRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	idParam := c.Params("id")
	
	var req CreateAchievementRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate and convert ID
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid achievement ID",
		})
	}

	// Process update
	achievement, err := s.UpdateAchievement(userID, id, &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    achievement,
	})
}

// FR-005: Hapus Prestasi - Service method untuk soft delete prestasi mahasiswa
func (s *AchievementService) DeleteAchievementRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	userRole := c.Locals("role").(string)
	username := c.Locals("username").(string)
	idParam := c.Params("id")

	// FR-005 Precondition: Hanya mahasiswa yang bisa hapus prestasi mereka sendiri
	if userRole != "student" && userRole != "Mahasiswa" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error": "Access denied",
			"message": "Only students can delete their own achievements",
			"code": "INSUFFICIENT_PERMISSIONS",
			"user_role": userRole,
			"required_role": "student",
		})
	}

	// Validate and convert ID
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": "Invalid achievement ID",
			"message": "The provided achievement ID is not valid",
			"code": "INVALID_ACHIEVEMENT_ID",
			"provided_id": idParam,
		})
	}

	// Get achievement details before deletion for response
	achievement, err := s.achievementRepo.GetByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error": "Achievement not found",
			"message": "The requested achievement does not exist or has been deleted",
			"code": "ACHIEVEMENT_NOT_FOUND",
		})
	}

	// Get reference for status check - use safe method
	reference, err := s.GetReferenceByAchievementIDSafe(id.Hex())
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error": "Achievement reference not found",
			"message": "Unable to find achievement reference. Try running /debug/fix-comprehensive first.",
			"code": "REFERENCE_NOT_FOUND",
			"hint": "POST /api/achievements/debug/fix-comprehensive to fix ID mismatches",
		})
	}

	// Get student info
	student, err := s.studentRepo.GetByUserID(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": "Student profile not found",
			"code": "STUDENT_PROFILE_NOT_FOUND",
		})
	}

	// FR-005 Flow: Soft delete prestasi
	err = s.SoftDeleteAchievement(userID, id)
	if err != nil {
		var errorCode string
		var message string
		
		switch err.Error() {
		case "achievement can only be deleted when status is 'draft'":
			errorCode = "INVALID_STATUS"
			message = "Only draft achievements can be deleted. Current status: " + reference.Status
		case "achievement not found":
			errorCode = "ACHIEVEMENT_NOT_FOUND"
			message = "The requested achievement does not exist"
		case "unauthorized":
			errorCode = "UNAUTHORIZED"
			message = "You can only delete your own achievements"
		default:
			errorCode = "DELETION_FAILED"
			message = "Failed to delete achievement"
		}
		
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": err.Error(),
			"message": message,
			"code": errorCode,
			"current_status": reference.Status,
			"allowed_status": []string{"draft"},
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Achievement deleted successfully (soft delete)",
		"code": "DELETION_SUCCESS",
		"data": fiber.Map{
			"achievement_id": id.Hex(),
			"reference_id": reference.ID.Hex(),
			"deleted_at": time.Now(),
			"previous_status": reference.Status,
			"new_status": "deleted",
			"student_info": fiber.Map{
				"student_id": student.StudentID,
				"full_name": username,
				"program_study": student.ProgramStudy,
			},
			"achievement_summary": fiber.Map{
				"title": achievement.Title,
				"category": achievement.Category,
				"created_at": achievement.CreatedAt,
			},
		},
		"deletion_info": fiber.Map{
			"type": "soft_delete",
			"description": "Achievement is marked as deleted but data is preserved",
			"recovery": "Data can be recovered by admin if needed",
			"permanent": false,
		},
		"next_steps": []string{
			"Achievement has been successfully deleted",
			"The data is soft deleted and can be recovered by admin if needed",
			"You can create a new achievement anytime",
		},
		"note": "Data is soft deleted and can be recovered by admin if needed",
		"timestamp": time.Now(),
	})
}

// FR-004: Submit untuk Verifikasi - Service method untuk submit prestasi untuk verifikasi
func (s *AchievementService) SubmitAchievementRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	userRole := c.Locals("role").(string)
	username := c.Locals("username").(string)
	achievementID := c.Params("achievement_id")

	// FR-004 Precondition: Hanya mahasiswa yang bisa submit
	if userRole != "student" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error": "Access denied",
			"message": "Only students can submit achievements for verification",
			"code": "INSUFFICIENT_PERMISSIONS",
			"user_role": userRole,
			"required_role": "student",
		})
	}

	if achievementID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": "Missing achievement ID",
			"message": "Achievement ID is required in the URL path",
			"code": "MISSING_ACHIEVEMENT_ID",
		})
	}

	// Get student and achievement info for detailed response
	student, err := s.studentRepo.GetByUserID(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": "Student profile not found",
			"message": "Unable to find student profile for this user",
			"code": "STUDENT_PROFILE_NOT_FOUND",
		})
	}

	// Get achievement details before submission
	achievementObjID, err := primitive.ObjectIDFromHex(achievementID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": "Invalid achievement ID",
			"message": "The provided achievement ID is not valid",
			"code": "INVALID_ACHIEVEMENT_ID",
		})
	}

	achievement, err := s.achievementRepo.GetByID(achievementObjID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error": "Achievement not found",
			"message": "The requested achievement does not exist or has been deleted",
			"code": "ACHIEVEMENT_NOT_FOUND",
		})
	}

	// FR-004 Flow: Submit prestasi untuk verifikasi
	updatedReference, err := s.SubmitAchievementForVerification(userID, achievementID)
	if err != nil {
		var errorCode string
		var message string
		
		switch err.Error() {
		case "achievement can only be submitted when status is 'draft'":
			errorCode = "INVALID_STATUS"
			message = "Only draft achievements can be submitted for verification"
		case "achievement not found":
			errorCode = "ACHIEVEMENT_NOT_FOUND"
			message = "The requested achievement does not exist"
		default:
			errorCode = "SUBMISSION_FAILED"
			message = "Failed to submit achievement for verification"
		}
		
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": err.Error(),
			"message": message,
			"code": errorCode,
		})
	}

	// Get advisor info if available
	var advisorInfo fiber.Map
	if student.AdvisorID != "" {
		lecturer, err := s.lecturerRepo.GetByUserID(student.AdvisorID)
		if err == nil {
			advisorInfo = fiber.Map{
				"advisor_id": lecturer.LecturerID,
				"department": lecturer.Department,
				"notified": true,
			}
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Achievement submitted for verification successfully",
		"code": "SUBMISSION_SUCCESS",
		"data": fiber.Map{
			"reference_id": updatedReference.ID.Hex(),
			"achievement_id": updatedReference.AchievementID,
			"status": updatedReference.Status,
			"submitted_at": updatedReference.SubmittedAt,
			"student_info": fiber.Map{
				"student_id": student.StudentID,
				"full_name": username,
				"program_study": student.ProgramStudy,
				"academic_year": student.AcademicYear,
			},
			"achievement_summary": fiber.Map{
				"title": achievement.Title,
				"category": achievement.Category,
				"description": achievement.Description,
			},
			"advisor_info": advisorInfo,
		},
		"status_info": fiber.Map{
			"previous_status": "draft",
			"current_status": "submitted",
			"description": "Achievement is now pending verification by your advisor",
			"available_actions": []string{"view", "wait_for_verification"},
		},
		"next_steps": []string{
			"Your achievement has been submitted to your advisor for verification",
			"You will receive a notification when the verification is complete",
			"You cannot edit the achievement while it's under review",
			"Check your notifications for updates",
		},
		"notification": fiber.Map{
			"advisor_notified": advisorInfo != nil,
			"message": "New achievement submission requires your verification",
			"type": "verification_request",
		},
		"timestamp": time.Now(),
	})
}

func (s *AchievementService) GetPendingVerificationsRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	references, err := s.GetPendingVerifications(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    references,
	})
}

// FR-007: Service method untuk get detail prestasi yang akan diverifikasi
func (s *AchievementService) GetVerificationDetailRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	userRole := c.Locals("role").(string)
	refIDParam := c.Params("reference_id")

	// FR-007 Precondition: Hanya dosen wali yang bisa akses
	if userRole != "lecturer" && userRole != "Dosen" && userRole != "Dosen Wali" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only lecturers can view verification details",
		})
	}

	// Validate and convert ID
	refID, err := primitive.ObjectIDFromHex(refIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid reference ID",
		})
	}

	// Get verification detail
	detail, err := s.GetVerificationDetail(userID, refID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    detail,
	})
}

// FR-007: Verify Prestasi & FR-008: Reject Prestasi - Service method untuk dosen wali memverifikasi/menolak prestasi mahasiswa
func (s *AchievementService) VerifyAchievementRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	userRole := c.Locals("role").(string)
	username := c.Locals("username").(string)
	refIDParam := c.Params("reference_id")

	// FR-007 Precondition: Hanya dosen wali yang bisa verify
	if userRole != "lecturer" && userRole != "Dosen" && userRole != "Dosen Wali" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error": "Access denied",
			"message": "Only lecturers can verify achievements",
			"code": "INSUFFICIENT_PERMISSIONS",
			"user_role": userRole,
			"required_role": "lecturer",
		})
	}
	
	var req VerifyAchievementRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": "Invalid request body",
			"message": "Please provide valid JSON data",
			"code": "INVALID_REQUEST_BODY",
			"details": err.Error(),
		})
	}

	// FR-007/FR-008 Step 1 & 2: Enhanced validation
	validationErrors := make(map[string]string)
	
	if req.Status != "verified" && req.Status != "rejected" {
		validationErrors["status"] = "Status must be 'verified' or 'rejected'"
	}

	// FR-008 Step 1: Dosen input rejection note (required when rejecting)
	if req.Status == "rejected" && req.RejectionNote == "" {
		validationErrors["rejection_note"] = "Rejection note is required when rejecting"
	}

	if len(validationErrors) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": "Validation failed",
			"message": "Please correct the following errors",
			"code": "VALIDATION_ERROR",
			"details": validationErrors,
			"valid_statuses": []string{"verified", "rejected"},
		})
	}

	// Validate and convert ID
	refID, err := primitive.ObjectIDFromHex(refIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": "Invalid reference ID",
			"message": "The provided reference ID is not valid",
			"code": "INVALID_REFERENCE_ID",
			"provided_id": refIDParam,
		})
	}

	// Get reference and achievement details before processing
	reference, err := s.achievementRepo.GetReferenceByID(refID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error": "Reference not found",
			"message": "The requested achievement reference does not exist",
			"code": "REFERENCE_NOT_FOUND",
		})
	}

	// Get achievement details
	achievementObjID, _ := primitive.ObjectIDFromHex(reference.AchievementID)
	achievement, err := s.achievementRepo.GetByID(achievementObjID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error": "Achievement not found",
			"code": "ACHIEVEMENT_NOT_FOUND",
		})
	}

	// Get student info
	student, err := s.studentRepo.GetByUserID(reference.StudentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error": "Student not found",
			"code": "STUDENT_NOT_FOUND",
		})
	}

	// Get lecturer info
	lecturer, err := s.lecturerRepo.GetByUserID(userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error": "Lecturer profile not found",
			"code": "LECTURER_NOT_FOUND",
		})
	}

	// FR-007/FR-008 Flow: Process verification/rejection
	updatedReference, err := s.VerifyAchievementWithDetails(userID, refID, &req)
	if err != nil {
		var errorCode string
		var message string
		
		switch err.Error() {
		case "only submitted achievements can be verified":
			errorCode = "INVALID_STATUS"
			message = "Only submitted achievements can be verified. Current status: " + reference.Status
		case "unauthorized":
			errorCode = "UNAUTHORIZED"
			message = "You can only verify achievements of your advisee students"
		default:
			errorCode = "VERIFICATION_FAILED"
			message = "Failed to process verification"
		}
		
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": err.Error(),
			"message": message,
			"code": errorCode,
			"current_status": reference.Status,
			"allowed_status": []string{"submitted"},
		})
	}

	// Calculate processing time
	var processingTime string
	if !reference.SubmittedAt.IsZero() && !updatedReference.VerifiedAt.IsZero() {
		duration := updatedReference.VerifiedAt.Sub(reference.SubmittedAt)
		if duration.Hours() < 1 {
			processingTime = fmt.Sprintf("%.0f minutes", duration.Minutes())
		} else if duration.Hours() < 24 {
			processingTime = fmt.Sprintf("%.1f hours", duration.Hours())
		} else {
			processingTime = fmt.Sprintf("%.1f days", duration.Hours()/24)
		}
	}

	// FR-007/FR-008 Step 5: Enhanced return response
	var message, actionType string
	var nextSteps []string
	
	if req.Status == "verified" {
		message = "Achievement verified successfully"
		actionType = "VERIFICATION_SUCCESS"
		nextSteps = []string{
			"Achievement has been approved and is now verified",
			"Student will receive a notification about the approval",
			"Achievement is now part of student's verified portfolio",
		}
	} else {
		message = "Achievement rejected successfully"
		actionType = "REJECTION_SUCCESS"
		nextSteps = []string{
			"Achievement has been rejected with feedback",
			"Student will receive a notification with rejection details",
			"Student can review feedback and resubmit after improvements",
		}
	}

	responseData := fiber.Map{
		"success": true,
		"message": message,
		"code": actionType,
		"data": fiber.Map{
			"reference_id": updatedReference.ID.Hex(),
			"achievement_id": updatedReference.AchievementID,
			"status": updatedReference.Status,
			"verified_by": updatedReference.VerifiedBy,
			"verified_at": updatedReference.VerifiedAt,
			"lecturer_info": fiber.Map{
				"lecturer_id": lecturer.LecturerID,
				"full_name": username,
				"department": lecturer.Department,
			},
			"student_info": fiber.Map{
				"student_id": student.StudentID,
				"program_study": student.ProgramStudy,
				"academic_year": student.AcademicYear,
				"user_id": student.UserID,
			},
			"achievement_summary": fiber.Map{
				"title": achievement.Title,
				"category": achievement.Category,
				"submitted_at": reference.SubmittedAt,
			},
		},
		"status_info": fiber.Map{
			"previous_status": "submitted",
			"current_status": updatedReference.Status,
			"processing_time": processingTime,
		},
		"next_steps": nextSteps,
		"timestamp": time.Now(),
	}

	// Add rejection-specific info
	if req.Status == "rejected" {
		responseData["data"].(fiber.Map)["rejection_note"] = updatedReference.RejectionNote
		responseData["notification_sent"] = fiber.Map{
			"to_student": true,
			"message": "Your achievement submission has been rejected",
			"includes_feedback": true,
		}
	}

	return c.JSON(responseData)
}

// UploadAttachmentRequest - Service method for file upload for achievement attachments
func (s *AchievementService) UploadAttachmentRequest(c *fiber.Ctx) error {
	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No file uploaded",
		})
	}

	// Validate file size (max 5MB)
	if file.Size > 5*1024*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "File size too large (max 5MB)",
		})
	}

	// Validate file type
	allowedTypes := map[string]bool{
		"application/pdf":  true,
		"image/jpeg":       true,
		"image/jpg":        true,
		"image/png":        true,
		"application/msword": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	}

	if !allowedTypes[file.Header.Get("Content-Type")] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid file type. Allowed: PDF, JPG, PNG, DOC, DOCX",
		})
	}

	// Create uploads directory if not exists
	uploadsDir := "./uploads/achievements"
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create upload directory",
		})
	}

	// Generate unique filename
	filename := fmt.Sprintf("%d_%s", time.Now().Unix(), file.Filename)
	filepath := fmt.Sprintf("%s/%s", uploadsDir, filename)

	// Save file
	if err := c.SaveFile(file, filepath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save file",
		})
	}

	// Create attachment object
	attachment := model.Attachment{
		FileName: file.Filename,
		FileURL:  fmt.Sprintf("/uploads/achievements/%s", filename),
		FileType: file.Header.Get("Content-Type"),
		FileSize: file.Size,
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "File uploaded successfully",
		"data":    attachment,
	})
}

// FR-003: SubmitAchievement - Main method untuk submit prestasi mahasiswa
func (s *AchievementService) SubmitAchievement(studentID string, req *CreateAchievementRequest) (*model.Achievement, *model.AchievementReference, error) {
	// Get student info from PostgreSQL
	student, err := s.studentRepo.GetByUserID(studentID)
	if err != nil {
		return nil, nil, errors.New("student not found")
	}

	// Create achievement object
	achievement := &model.Achievement{
		StudentID:    studentID,
		StudentInfo:  student.StudentID,
		Category:     req.Category,
		Title:        req.Title,
		Description:  req.Description,
		CustomFields: req.CustomFields,
		Attachments:  req.Attachments,
		Tags:         req.Tags,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Map details based on category
	s.mapDetailsToAchievement(achievement, req.Details, req.Category)

	// Save achievement to MongoDB
	err = s.achievementRepo.Create(achievement)
	if err != nil {
		return nil, nil, errors.New("failed to save achievement: " + err.Error())
	}

	// Create achievement reference for tracking (status: draft)
	reference := &model.AchievementReference{
		StudentID:     studentID,
		AchievementID: achievement.ID.Hex(), // Fix: Use actual MongoDB ID, not ObjectID field
		Status:        "draft", // FR-003 Step 4: Status awal 'draft'
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Save reference to MongoDB
	err = s.achievementRepo.CreateReference(reference)
	if err != nil {
		// Rollback: delete achievement if reference creation fails
		s.achievementRepo.Delete(achievement.ID)
		return nil, nil, errors.New("failed to create achievement reference: " + err.Error())
	}

	return achievement, reference, nil
}

// Legacy method for backward compatibility
func (s *AchievementService) CreateAchievement(studentID string, req *CreateAchievementRequest) (*model.Achievement, error) {
	achievement, _, err := s.SubmitAchievement(studentID, req)
	return achievement, err
}

func (s *AchievementService) SubmitAchievementForVerification(studentID string, achievementID string) (*model.AchievementReference, error) {
	// FR-004 Step 1: Find the reference
	references, err := s.achievementRepo.GetReferencesByStudentID(studentID)
	if err != nil {
		return nil, err
	}

	var targetRef *model.AchievementReference
	for _, ref := range references {
		if ref.AchievementID == achievementID {
			targetRef = &ref
			break
		}
	}

	if targetRef == nil {
		return nil, errors.New("achievement not found")
	}

	// FR-004 Precondition: Prestasi berstatus 'draft'
	if targetRef.Status != "draft" {
		return nil, errors.New("only draft achievements can be submitted for verification")
	}

	// FR-004 Step 2: Update status menjadi 'submitted'
	targetRef.Status = "submitted"
	targetRef.SubmittedAt = time.Now()
	targetRef.UpdatedAt = time.Now()

	err = s.achievementRepo.UpdateReference(targetRef)
	if err != nil {
		return nil, err
	}

	// FR-004 Step 3: Create notification untuk dosen wali
	err = s.createNotificationForAdvisor(studentID, achievementID)
	if err != nil {
		// Log error but don't fail the submission
		fmt.Printf("Failed to create notification: %v\n", err)
	}

	// FR-004 Step 4: Return updated status
	return targetRef, nil
}

// Helper method to create notification for advisor
func (s *AchievementService) createNotificationForAdvisor(studentID, achievementID string) error {
	// Get student info
	student, err := s.studentRepo.GetByUserID(studentID)
	if err != nil {
		return err
	}

	// Get student user info for name
	// Note: We need to add a method to get user info, for now we'll use student ID
	studentName := student.StudentID // Fallback to student ID

	// Get achievement info
	achievement, err := s.achievementRepo.GetByObjectID(achievementID)
	if err != nil {
		return err
	}

	// Get advisor info
	if student.AdvisorID == "" {
		return errors.New("student has no advisor assigned")
	}

	lecturer, err := s.lecturerRepo.GetByID(student.AdvisorID)
	if err != nil {
		return err
	}

	// Create notification (simplified - in real app you'd use notification service)
	fmt.Printf("NOTIFICATION: Achievement '%s' submitted by student %s for verification by lecturer %s\n", 
		achievement.Title, studentName, lecturer.LecturerID)

	return nil
}

// FR-007: VerifyAchievementWithDetails - Main method untuk verify prestasi dengan return details
func (s *AchievementService) VerifyAchievementWithDetails(lecturerID string, referenceID primitive.ObjectID, req *VerifyAchievementRequest) (*model.AchievementReference, error) {
	// FR-007 Step 1: Get the reference untuk review prestasi detail
	reference, err := s.achievementRepo.GetReferenceByID(referenceID)
	if err != nil {
		return nil, errors.New("achievement reference not found")
	}

	// FR-007 Precondition: Status harus 'submitted'
	if reference.Status != "submitted" {
		return nil, fmt.Errorf("only submitted achievements can be verified. Current status: '%s'", reference.Status)
	}

	// Check if lecturer is the advisor of the student
	student, err := s.studentRepo.GetByUserID(reference.StudentID)
	if err != nil {
		return nil, errors.New("student not found")
	}

	lecturer, err := s.lecturerRepo.GetByUserID(lecturerID)
	if err != nil {
		return nil, errors.New("lecturer not found")
	}

	if student.AdvisorID != lecturer.ID {
		return nil, errors.New("you can only verify achievements of your advisees")
	}

	// FR-007/FR-008 Step 2 & 3: Update status menjadi 'verified' atau 'rejected'
	reference.Status = req.Status
	reference.VerifiedBy = lecturerID
	reference.VerifiedAt = time.Now()
	reference.UpdatedAt = time.Now()
	
	// FR-008 Step 3: Save rejection_note (if rejected)
	if req.Status == "rejected" {
		reference.RejectionNote = req.RejectionNote
	}

	err = s.achievementRepo.UpdateReference(reference)
	if err != nil {
		return nil, errors.New("failed to update achievement reference: " + err.Error())
	}

	// FR-008 Step 4: Create notification untuk mahasiswa (if rejected)
	if req.Status == "rejected" && s.notificationService != nil {
		err = s.createRejectionNotification(reference.StudentID, reference.AchievementID, req.RejectionNote)
		if err != nil {
			// Log error but don't fail the verification process
			fmt.Printf("Warning: Failed to create rejection notification: %v\n", err)
		}
	}

	// FR-007 Step 5 / FR-008 Step 5: Return updated status
	return reference, nil
}

// FR-008 Step 4: Create notification untuk mahasiswa when achievement is rejected
func (s *AchievementService) createRejectionNotification(studentID, achievementID, rejectionNote string) error {
	// Get achievement detail for notification
	achievement, err := s.achievementRepo.GetByObjectID(achievementID)
	if err != nil {
		return err
	}

	// Create notification title and message
	title := "Achievement Rejected"
	message := fmt.Sprintf("Your achievement '%s' has been rejected by your advisor. Please review the feedback and resubmit if needed.", achievement.Title)
	
	// Notification data
	data := map[string]interface{}{
		"achievement_id":    achievementID,
		"achievement_title": achievement.Title,
		"rejection_note":    rejectionNote,
		"action_required":   "review_and_resubmit",
		"type":             "achievement_rejected",
	}

	// Create notification
	return s.notificationService.CreateNotification(studentID, "achievement_rejected", title, message, data)
}

// Legacy method for backward compatibility
func (s *AchievementService) VerifyAchievement(lecturerID string, referenceID primitive.ObjectID, req *VerifyAchievementRequest) error {
	_, err := s.VerifyAchievementWithDetails(lecturerID, referenceID, req)
	return err
}

func (s *AchievementService) GetStudentAchievements(studentID string) ([]model.Achievement, error) {
	return s.achievementRepo.GetByStudentID(studentID)
}

func (s *AchievementService) GetStudentAchievementReferences(studentID string) ([]model.AchievementReference, error) {
	return s.achievementRepo.GetReferencesByStudentID(studentID)
}

func (s *AchievementService) GetPendingVerifications(lecturerID string) ([]model.AchievementReference, error) {
	// Get lecturer info
	lecturer, err := s.lecturerRepo.GetByUserID(lecturerID)
	if err != nil {
		return nil, errors.New("lecturer not found")
	}

	// Get students under this lecturer
	students, err := s.studentRepo.GetByAdvisorID(lecturer.ID)
	if err != nil {
		return nil, err
	}

	var allReferences []model.AchievementReference
	for _, student := range students {
		references, err := s.achievementRepo.GetReferencesByStudentID(student.UserID)
		if err != nil {
			continue
		}

		// Filter only submitted achievements
		for _, ref := range references {
			if ref.Status == "submitted" {
				allReferences = append(allReferences, ref)
			}
		}
	}

	return allReferences, nil
}

// FR-007: GetVerificationDetail - Get detail prestasi untuk review oleh dosen
type VerificationDetail struct {
	Reference    *model.AchievementReference `json:"reference"`
	Achievement  *model.Achievement          `json:"achievement"`
	StudentInfo  StudentBasicInfo            `json:"student_info"`
}

func (s *AchievementService) GetVerificationDetail(lecturerID string, referenceID primitive.ObjectID) (*VerificationDetail, error) {
	// Get the reference
	reference, err := s.achievementRepo.GetReferenceByID(referenceID)
	if err != nil {
		return nil, errors.New("achievement reference not found")
	}

	// Check if lecturer is the advisor of the student
	student, err := s.studentRepo.GetByUserID(reference.StudentID)
	if err != nil {
		return nil, errors.New("student not found")
	}

	lecturer, err := s.lecturerRepo.GetByUserID(lecturerID)
	if err != nil {
		return nil, errors.New("lecturer not found")
	}

	if student.AdvisorID != lecturer.ID {
		return nil, errors.New("you can only view achievements of your advisees")
	}

	// Get achievement detail from MongoDB
	achievement, err := s.achievementRepo.GetByObjectID(reference.AchievementID)
	if err != nil {
		return nil, errors.New("achievement not found")
	}

	// Skip soft deleted achievements
	if achievement.DeletedAt != nil {
		return nil, errors.New("achievement has been deleted")
	}

	// Prepare student info
	studentInfo := StudentBasicInfo{
		StudentID:    student.StudentID,
		ProgramStudy: student.ProgramStudy,
		UserID:       student.UserID,
	}

	return &VerificationDetail{
		Reference:   reference,
		Achievement: achievement,
		StudentInfo: studentInfo,
	}, nil
}

func (s *AchievementService) GetAchievementByID(id primitive.ObjectID) (*model.Achievement, error) {
	return s.achievementRepo.GetByID(id)
}

func (s *AchievementService) GetActiveAchievementByID(id primitive.ObjectID) (*model.Achievement, error) {
	return s.achievementRepo.GetByIDActive(id)
}

func (s *AchievementService) UpdateAchievement(studentID string, achievementID primitive.ObjectID, req *CreateAchievementRequest) (*model.Achievement, error) {
	// Get existing achievement
	achievement, err := s.achievementRepo.GetByID(achievementID)
	if err != nil {
		return nil, errors.New("achievement not found")
	}

	// Check ownership
	if achievement.StudentID != studentID {
		return nil, errors.New("unauthorized")
	}

	// Check if achievement is deleted
	if achievement.DeletedAt != nil {
		return nil, errors.New("cannot update deleted achievement")
	}

	// Update fields
	achievement.Category = req.Category
	achievement.Title = req.Title
	achievement.Description = req.Description
	achievement.CustomFields = req.CustomFields
	achievement.Attachments = req.Attachments
	achievement.Tags = req.Tags

	// Map details
	s.mapDetailsToAchievement(achievement, req.Details, req.Category)

	err = s.achievementRepo.Update(achievement)
	if err != nil {
		return nil, err
	}

	return achievement, nil
}

// FR-005: SoftDeleteAchievement - Soft delete prestasi dengan validasi status draft
func (s *AchievementService) SoftDeleteAchievement(studentID string, achievementID primitive.ObjectID) error {
	// FR-005 Step 1: Get existing achievement
	achievement, err := s.achievementRepo.GetByID(achievementID)
	if err != nil {
		return errors.New("achievement not found")
	}

	// FR-005 Step 2: Check ownership
	if achievement.StudentID != studentID {
		return errors.New("unauthorized: you can only delete your own achievements")
	}

	// FR-005 Step 3: Check if achievement is already deleted
	if achievement.DeletedAt != nil {
		return errors.New("achievement is already deleted")
	}

	// FR-005 Step 4: Get achievement reference to check status - use safe method
	targetRef, err := s.GetReferenceByAchievementIDSafe(achievementID.Hex())
	if err != nil {
		return errors.New("achievement reference not found")
	}

	// FR-005 Precondition: Hanya prestasi dengan status 'draft' yang bisa dihapus
	if targetRef.Status != "draft" {
		return errors.New("only draft achievements can be deleted")
	}

	// FR-005 Step 5: Perform soft delete
	now := time.Now()
	achievement.DeletedAt = &now
	achievement.UpdatedAt = now

	err = s.achievementRepo.Update(achievement)
	if err != nil {
		return errors.New("failed to delete achievement: " + err.Error())
	}

	// FR-005 Step 6: Update reference status to 'deleted'
	targetRef.Status = "deleted"
	targetRef.UpdatedAt = now

	err = s.achievementRepo.UpdateReference(targetRef)
	if err != nil {
		// Log error but don't fail the deletion since achievement is already soft deleted
		fmt.Printf("Warning: Failed to update reference status to deleted: %v\n", err)
	}

	return nil
}

// Legacy method for backward compatibility (now deprecated)
func (s *AchievementService) DeleteAchievement(studentID string, achievementID primitive.ObjectID) error {
	return s.SoftDeleteAchievement(studentID, achievementID)
}

// FR-006: View Prestasi Mahasiswa Bimbingan - Service method untuk dosen wali
func (s *AchievementService) GetAdviseeAchievementsRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	userRole := c.Locals("role").(string)
	username := c.Locals("username").(string)

	// FR-006 Precondition: Hanya dosen yang bisa akses
	if userRole != "lecturer" && userRole != "Dosen" && userRole != "Dosen Wali" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error": "Access denied",
			"message": "Only lecturers can view advisee achievements",
			"code": "INSUFFICIENT_PERMISSIONS",
			"user_role": userRole,
			"required_role": "lecturer",
		})
	}

	// Parse pagination parameters with validation
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	status := c.Query("status", "")
	category := c.Query("category", "")
	
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	// Get lecturer info for response
	lecturer, err := s.lecturerRepo.GetByUserID(userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error": "Lecturer profile not found",
			"message": "Unable to find lecturer profile for this user",
			"code": "LECTURER_PROFILE_NOT_FOUND",
		})
	}

	// FR-006 Flow: Get advisee achievements with pagination
	result, err := s.GetAdviseeAchievements(userID, limit, offset)
	if err != nil {
		var errorCode string
		var message string
		
		switch err.Error() {
		case "no advisee students found":
			errorCode = "NO_ADVISEES"
			message = "You don't have any advisee students assigned"
		default:
			errorCode = "FETCH_FAILED"
			message = "Failed to retrieve advisee achievements"
		}
		
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": err.Error(),
			"message": message,
			"code": errorCode,
		})
	}

	// Get advisee count and statistics
	adviseeStudents, _ := s.studentRepo.GetByAdvisorID(userID)
	totalAdvisees := len(adviseeStudents)
	
	// Calculate statistics
	statusCounts := make(map[string]int)
	categoryCounts := make(map[string]int)
	
	for _, achievement := range result.Achievements {
		statusCounts[achievement.Reference.Status]++
		categoryCounts[achievement.Achievement.Category]++
	}

	// Calculate pagination
	totalPages := (result.Total + int64(limit) - 1) / int64(limit)
	hasNextPage := page < int(totalPages)
	hasPreviousPage := page > 1

	// Build next/previous URLs
	var nextPageURL, previousPageURL *string
	if hasNextPage {
		nextURL := fmt.Sprintf("/api/achievements/advisee?page=%d&limit=%d", page+1, limit)
		if status != "" {
			nextURL += "&status=" + status
		}
		if category != "" {
			nextURL += "&category=" + category
		}
		nextPageURL = &nextURL
	}
	if hasPreviousPage {
		prevURL := fmt.Sprintf("/api/achievements/advisee?page=%d&limit=%d", page-1, limit)
		if status != "" {
			prevURL += "&status=" + status
		}
		if category != "" {
			prevURL += "&category=" + category
		}
		previousPageURL = &prevURL
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Advisee achievements retrieved successfully",
		"code": "ADVISEE_ACHIEVEMENTS_FOUND",
		"summary": fiber.Map{
			"total_achievements": result.Total,
			"page_results": len(result.Achievements),
			"total_advisees": totalAdvisees,
			"filters_applied": fiber.Map{
				"status": status,
				"category": category,
			},
		},
		"data": result.Achievements,
		"pagination": fiber.Map{
			"current_page": page,
			"per_page": limit,
			"total_items": result.Total,
			"total_pages": totalPages,
			"has_next_page": hasNextPage,
			"has_previous_page": hasPreviousPage,
			"next_page_url": nextPageURL,
			"previous_page_url": previousPageURL,
		},
		"lecturer_info": fiber.Map{
			"lecturer_id": lecturer.LecturerID,
			"full_name": username,
			"department": lecturer.Department,
			"total_advisees": totalAdvisees,
			"total_achievements": result.Total,
		},
		"statistics": fiber.Map{
			"by_status": statusCounts,
			"by_category": categoryCounts,
		},
		"filters": fiber.Map{
			"applied": fiber.Map{
				"status": status,
				"category": category,
			},
			"available": fiber.Map{
				"statuses": []string{"draft", "submitted", "verified", "rejected"},
				"categories": []string{"competition", "research", "community_service", "academic", "organization"},
			},
		},
		"actions_available": []string{
			"verify_achievement",
			"reject_achievement",
			"view_achievement_detail",
		},
		"timestamp": time.Now(),
	})
}

// FR-006: GetAdviseeAchievements - Main method untuk get prestasi mahasiswa bimbingan
type AdviseeAchievementsResult struct {
	Achievements []AdviseeAchievementDetail `json:"achievements"`
	Total        int64                      `json:"total"`
}

type AdviseeAchievementDetail struct {
	Achievement  *model.Achievement          `json:"achievement"`
	Reference    *model.AchievementReference `json:"reference"`
	StudentInfo  StudentBasicInfo            `json:"student_info"`
}

type StudentBasicInfo struct {
	StudentID    string `json:"student_id"`
	ProgramStudy string `json:"program_study"`
	UserID       string `json:"user_id"`
}

func (s *AchievementService) GetAdviseeAchievements(lecturerUserID string, limit, offset int) (*AdviseeAchievementsResult, error) {
	// FR-006 Step 1: Get lecturer info
	lecturer, err := s.lecturerRepo.GetByUserID(lecturerUserID)
	if err != nil {
		return nil, errors.New("lecturer not found")
	}

	// FR-006 Step 2: Get list student IDs dari tabel students where advisor_id
	students, err := s.studentRepo.GetByAdvisorID(lecturer.ID)
	if err != nil {
		return nil, errors.New("failed to get advisee students: " + err.Error())
	}

	if len(students) == 0 {
		return &AdviseeAchievementsResult{
			Achievements: []AdviseeAchievementDetail{},
			Total:        0,
		}, nil
	}

	// Extract student user IDs for achievement query
	var studentUserIDs []string
	studentInfoMap := make(map[string]StudentBasicInfo)
	
	for _, student := range students {
		studentUserIDs = append(studentUserIDs, student.UserID)
		studentInfoMap[student.UserID] = StudentBasicInfo{
			StudentID:    student.StudentID,
			ProgramStudy: student.ProgramStudy,
			UserID:       student.UserID,
		}
	}

	// FR-006 Step 3: Get achievements references dengan filter student_ids
	references, err := s.achievementRepo.GetReferencesByStudentIDs(studentUserIDs, limit, offset)
	if err != nil {
		return nil, errors.New("failed to get achievement references: " + err.Error())
	}

	// Get total count for pagination
	totalCount, err := s.achievementRepo.CountByStudentIDs(studentUserIDs)
	if err != nil {
		return nil, errors.New("failed to count achievements: " + err.Error())
	}

	// FR-006 Step 4: Fetch detail dari MongoDB
	var result []AdviseeAchievementDetail
	
	for _, ref := range references {
		// Get achievement detail from MongoDB using ObjectID
		achievementObjID, err := primitive.ObjectIDFromHex(ref.AchievementID)
		if err != nil {
			fmt.Printf("Warning: Invalid achievement ID %s: %v\n", ref.AchievementID, err)
			continue
		}
		
		achievement, err := s.achievementRepo.GetByID(achievementObjID)
		if err != nil {
			// Log error but continue with other achievements
			fmt.Printf("Warning: Failed to get achievement %s: %v\n", ref.AchievementID, err)
			continue
		}

		// Skip soft deleted achievements
		if achievement.DeletedAt != nil {
			continue
		}

		// Get student info
		studentInfo, exists := studentInfoMap[ref.StudentID]
		if !exists {
			// Fallback: try to get student info directly
			student, err := s.studentRepo.GetByUserID(ref.StudentID)
			if err != nil {
				fmt.Printf("Warning: Failed to get student info for %s: %v\n", ref.StudentID, err)
				studentInfo = StudentBasicInfo{
					StudentID:    "Unknown",
					ProgramStudy: "Unknown",
					UserID:       ref.StudentID,
				}
			} else {
				studentInfo = StudentBasicInfo{
					StudentID:    student.StudentID,
					ProgramStudy: student.ProgramStudy,
					UserID:       student.UserID,
				}
			}
		}

		detail := AdviseeAchievementDetail{
			Achievement: achievement,
			Reference:   &ref,
			StudentInfo: studentInfo,
		}

		result = append(result, detail)
	}

	// FR-006 Step 5: Return list dengan pagination
	return &AdviseeAchievementsResult{
		Achievements: result,
		Total:        totalCount,
	}, nil
}

func (s *AchievementService) mapDetailsToAchievement(achievement *model.Achievement, details map[string]interface{}, category string) {
	// Map details based on category
	switch category {
	case "competition":
		if val, ok := details["competition_name"].(string); ok {
			achievement.Details.CompetitionName = val
		}
		if val, ok := details["competition_level"].(string); ok {
			achievement.Details.CompetitionLevel = val
		}
		if val, ok := details["rank"].(float64); ok {
			achievement.Details.Rank = int(val)
		}
		if val, ok := details["medal"].(string); ok {
			achievement.Details.Medal = val
		}

	case "publication":
		if val, ok := details["publication_type"].(string); ok {
			achievement.Details.PublicationType = val
		}
		if val, ok := details["publication_title"].(string); ok {
			achievement.Details.PublicationTitle = val
		}
		if val, ok := details["publication_journal"].(string); ok {
			achievement.Details.PublicationJournal = val
		}
		if val, ok := details["publisher"].(string); ok {
			achievement.Details.Publisher = val
		}
		if val, ok := details["issn"].(string); ok {
			achievement.Details.ISSN = val
		}

	case "organization":
		if val, ok := details["organization_name"].(string); ok {
			achievement.Details.OrganizationName = val
		}
		if val, ok := details["position"].(string); ok {
			achievement.Details.Position = val
		}

	case "certification":
		if val, ok := details["certification_name"].(string); ok {
			achievement.Details.CertificationName = val
		}
		if val, ok := details["issued_by"].(string); ok {
			achievement.Details.IssuedBy = val
		}
		if val, ok := details["certification_number"].(string); ok {
			achievement.Details.CertificationNumber = val
		}
	}

	// Common fields
	if val, ok := details["location"].(string); ok {
		achievement.Details.Location = val
	}
	if val, ok := details["organizer"].(string); ok {
		achievement.Details.Organizer = val
	}
	if val, ok := details["score"].(float64); ok {
		achievement.Details.Score = int(val)
	}
}
// FR-011: Achievement Statistics - Generate statistik prestasi
// GetAchievementStatisticsRequest handles achievement statistics
// @Summary Get Achievement Statistics
// @Description Get achievement statistics based on user role (FR-011)
// @Description - Student: own statistics
// @Description - Lecturer: advisee statistics  
// @Description - Admin: all statistics
// @Tags Statistics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param period query string false "Statistics period" Enums(all, year, month) default(all)
// @Param year query int false "Year for statistics" default(2024)
// @Param month query int false "Month for statistics (1-12)"
// @Success 200 {object} map[string]interface{} "Statistics retrieved successfully"
// @Failure 403 {object} map[string]interface{} "Forbidden"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /achievements/statistics [get]
func (s *AchievementService) GetAchievementStatisticsRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	userRole := c.Locals("role").(string)
	username := c.Locals("username").(string)

	// Validate user data
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": "Invalid user ID",
			"message": "User ID is required for statistics access",
			"code": "INVALID_USER_ID",
		})
	}

	// Get query parameters
	period := c.Query("period", "all") // all, year, month
	year := c.QueryInt("year", time.Now().Year())
	month := c.QueryInt("month", 0)

	var stats fiber.Map
	var err error

	switch userRole {
	case "student", "Mahasiswa":
		// FR-011: Mahasiswa melihat statistik prestasi sendiri
		stats, err = s.GetStudentStatistics(userID, period, year, month)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error": "Failed to get student statistics",
				"message": err.Error(),
				"code": "STUDENT_STATS_FAILED",
			})
		}

	case "lecturer", "Dosen", "Dosen Wali":
		// FR-011: Dosen Wali melihat statistik prestasi mahasiswa bimbingan
		stats, err = s.GetLecturerStatistics(userID, period, year, month)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error": "Failed to get lecturer statistics",
				"message": err.Error(),
				"code": "LECTURER_STATS_FAILED",
			})
		}

	case "admin":
		// FR-011: Admin melihat statistik semua prestasi
		stats, err = s.GetAdminStatistics(period, year, month)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error": "Failed to get admin statistics",
				"message": err.Error(),
				"code": "ADMIN_STATS_FAILED",
			})
		}

	default:
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error": "Access denied",
			"message": "Invalid user role for statistics access",
			"code": "INVALID_ROLE",
			"user_role": userRole,
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Achievement statistics retrieved successfully",
		"code": "STATISTICS_SUCCESS",
		"data": stats,
		"user_info": fiber.Map{
			"user_id": userID,
			"username": username,
			"role": userRole,
		},
		"query_parameters": fiber.Map{
			"period": period,
			"year": year,
			"month": month,
		},
		"timestamp": time.Now(),
	})
}

// FR-011: Get Student Statistics - Statistik prestasi mahasiswa sendiri
func (s *AchievementService) GetStudentStatistics(userID string, period string, year, month int) (fiber.Map, error) {
	// Get student info
	student, err := s.studentRepo.GetByUserID(userID)
	if err != nil {
		return nil, errors.New("student profile not found")
	}

	// Get student achievements
	achievements, err := s.achievementRepo.GetByStudentID(userID)
	if err != nil {
		return nil, err
	}

	// Get achievement references
	references, err := s.achievementRepo.GetReferencesByStudentID(userID)
	if err != nil {
		return nil, err
	}

	// Calculate statistics
	totalAchievements := len(achievements)
	statusCounts := make(map[string]int)
	categoryCounts := make(map[string]int)
	levelCounts := make(map[string]int)
	monthlyStats := make(map[string]int)

	// Process achievements with error handling
	for _, achievement := range achievements {
		// Safe category processing
		if achievement.Category != "" {
			categoryCounts[achievement.Category]++
		}
		
		// Safe competition level processing
		if achievement.Details.CompetitionLevel != "" {
			levelCounts[achievement.Details.CompetitionLevel]++
		}

		// Safe monthly statistics
		if !achievement.CreatedAt.IsZero() {
			monthKey := achievement.CreatedAt.Format("2006-01")
			monthlyStats[monthKey]++
		}
	}

	// Process references for status
	for _, ref := range references {
		statusCounts[ref.Status]++
	}

	// Calculate achievements by period
	var periodStats fiber.Map
	if period == "year" {
		periodStats = s.calculateYearlyStats(achievements, year)
	} else if period == "month" {
		periodStats = s.calculateMonthlyStats(achievements, year, month)
	}

	return fiber.Map{
		"overview": fiber.Map{
			"total_achievements": totalAchievements,
			"verified_achievements": statusCounts["verified"],
			"pending_achievements": statusCounts["submitted"],
			"draft_achievements": statusCounts["draft"],
		},
		"by_category": categoryCounts,
		"by_status": statusCounts,
		"by_competition_level": levelCounts,
		"monthly_trend": monthlyStats,
		"period_stats": periodStats,
		"student_info": fiber.Map{
			"student_id": student.StudentID,
			"program_study": student.ProgramStudy,
			"academic_year": student.AcademicYear,
		},
		"achievements_summary": fiber.Map{
			"most_active_category": s.getMostActiveCategory(categoryCounts),
			"achievement_rate": s.calculateAchievementRate(achievements, references),
			"recent_achievements": s.getRecentAchievements(achievements, 5),
		},
	}, nil
}

// FR-011: Get Lecturer Statistics - Statistik prestasi mahasiswa bimbingan
func (s *AchievementService) GetLecturerStatistics(lecturerID string, period string, year, month int) (fiber.Map, error) {
	// Get lecturer info
	lecturer, err := s.lecturerRepo.GetByUserID(lecturerID)
	if err != nil {
		return nil, errors.New("lecturer profile not found")
	}

	// Get advisee students
	adviseeStudents, err := s.studentRepo.GetByAdvisorID(lecturerID)
	if err != nil {
		return nil, err
	}

	if len(adviseeStudents) == 0 {
		return fiber.Map{
			"overview": fiber.Map{
				"total_advisees": 0,
				"total_achievements": 0,
				"message": "No advisee students found",
			},
			"lecturer_info": fiber.Map{
				"lecturer_id": lecturer.LecturerID,
				"department": lecturer.Department,
			},
		}, nil
	}

	// Get student IDs
	var studentIDs []string
	for _, student := range adviseeStudents {
		studentIDs = append(studentIDs, student.UserID)
	}

	// Get all achievements from advisee students
	achievements, err := s.achievementRepo.GetByStudentIDs(studentIDs, 0, 0)
	if err != nil {
		return nil, err
	}

	// Calculate statistics
	totalAchievements := len(achievements)
	statusCounts := make(map[string]int)
	categoryCounts := make(map[string]int)
	levelCounts := make(map[string]int)
	studentStats := make(map[string]int)

	// Process achievements with error handling
	for _, achievement := range achievements {
		// Safe category processing
		if achievement.Category != "" {
			categoryCounts[achievement.Category]++
		}
		
		// Safe student stats
		if achievement.StudentID != "" {
			studentStats[achievement.StudentID]++
		}
		
		// Safe competition level processing
		if achievement.Details.CompetitionLevel != "" {
			levelCounts[achievement.Details.CompetitionLevel]++
		}
	}

	// Get references for status counts
	for _, studentID := range studentIDs {
		refs, err := s.achievementRepo.GetReferencesByStudentID(studentID)
		if err != nil {
			continue
		}
		for _, ref := range refs {
			statusCounts[ref.Status]++
		}
	}

	// Get top performing students
	topStudents := s.getTopStudents(studentStats, adviseeStudents, 5)

	return fiber.Map{
		"overview": fiber.Map{
			"total_advisees": len(adviseeStudents),
			"total_achievements": totalAchievements,
			"verified_achievements": statusCounts["verified"],
			"pending_verifications": statusCounts["submitted"],
		},
		"by_category": categoryCounts,
		"by_status": statusCounts,
		"by_competition_level": levelCounts,
		"top_students": topStudents,
		"lecturer_info": fiber.Map{
			"lecturer_id": lecturer.LecturerID,
			"department": lecturer.Department,
			"total_advisees": len(adviseeStudents),
		},
		"advisee_performance": fiber.Map{
			"average_achievements_per_student": float64(totalAchievements) / float64(len(adviseeStudents)),
			"most_active_category": s.getMostActiveCategory(categoryCounts),
			"verification_rate": s.calculateVerificationRate(statusCounts),
		},
	}, nil
}

// FR-011: Get Admin Statistics - Statistik semua prestasi
func (s *AchievementService) GetAdminStatistics(period string, year, month int) (fiber.Map, error) {
	// Get all achievements
	achievements, err := s.achievementRepo.GetAll(0, 0)
	if err != nil {
		return nil, err
	}

	// Get all students
	students, err := s.studentRepo.GetAll()
	if err != nil {
		return nil, err
	}

	// Calculate comprehensive statistics
	totalAchievements := len(achievements)
	categoryCounts := make(map[string]int)
	levelCounts := make(map[string]int)
	monthlyStats := make(map[string]int)
	studentStats := make(map[string]int)

	// Process achievements
	for _, achievement := range achievements {
		categoryCounts[achievement.Category]++
		studentStats[achievement.StudentID]++
		
		// Extract competition level
		if achievement.Details.CompetitionLevel != "" {
			levelCounts[achievement.Details.CompetitionLevel]++
		}

		// Monthly statistics
		monthKey := achievement.CreatedAt.Format("2006-01")
		monthlyStats[monthKey]++
	}

	// Get status statistics from repository
	statusStats, err := s.achievementRepo.GetAchievementStatistics()
	if err != nil {
		return nil, err
	}

	// Get top performing students
	topStudents := s.getTopStudentsAdmin(studentStats, students, 10)

	// Calculate period-specific statistics
	var periodStats fiber.Map
	if period == "year" {
		periodStats = s.calculateYearlyStats(achievements, year)
	} else if period == "month" {
		periodStats = s.calculateMonthlyStats(achievements, year, month)
	}

	// Extract status stats safely
	var byStatusMap map[string]int
	if statusStats != nil {
		if statusMap, ok := statusStats["by_status"].(map[string]int); ok {
			byStatusMap = statusMap
		}
	}
	if byStatusMap == nil {
		byStatusMap = make(map[string]int)
	}

	return fiber.Map{
		"overview": fiber.Map{
			"total_achievements": totalAchievements,
			"total_students": len(students),
			"verified_achievements": byStatusMap["verified"],
			"pending_achievements": byStatusMap["submitted"],
		},
		"by_category": categoryCounts,
		"by_status": byStatusMap,
		"by_competition_level": levelCounts,
		"monthly_trend": monthlyStats,
		"period_stats": periodStats,
		"top_students": topStudents,
		"system_performance": fiber.Map{
			"average_achievements_per_student": float64(totalAchievements) / float64(len(students)),
			"most_popular_category": s.getMostActiveCategory(categoryCounts),
			"most_common_level": s.getMostActiveLevel(levelCounts),
			"achievement_growth": s.calculateGrowthRate(monthlyStats),
		},
	}, nil
}

// Helper functions for statistics calculation
func (s *AchievementService) calculateYearlyStats(achievements []model.Achievement, year int) fiber.Map {
	monthlyCount := make(map[int]int)
	
	for _, achievement := range achievements {
		if achievement.CreatedAt.Year() == year {
			monthlyCount[int(achievement.CreatedAt.Month())]++
		}
	}
	
	return fiber.Map{
		"year": year,
		"monthly_breakdown": monthlyCount,
		"total_for_year": s.sumMapValues(monthlyCount),
	}
}

func (s *AchievementService) calculateMonthlyStats(achievements []model.Achievement, year, month int) fiber.Map {
	dailyCount := make(map[int]int)
	
	for _, achievement := range achievements {
		if achievement.CreatedAt.Year() == year && int(achievement.CreatedAt.Month()) == month {
			dailyCount[achievement.CreatedAt.Day()]++
		}
	}
	
	return fiber.Map{
		"year": year,
		"month": month,
		"daily_breakdown": dailyCount,
		"total_for_month": s.sumMapValues(dailyCount),
	}
}

func (s *AchievementService) getMostActiveCategory(categoryCounts map[string]int) string {
	maxCount := 0
	mostActive := ""
	
	for category, count := range categoryCounts {
		if count > maxCount {
			maxCount = count
			mostActive = category
		}
	}
	
	return mostActive
}

func (s *AchievementService) getMostActiveLevel(levelCounts map[string]int) string {
	maxCount := 0
	mostActive := ""
	
	for level, count := range levelCounts {
		if count > maxCount {
			maxCount = count
			mostActive = level
		}
	}
	
	return mostActive
}

func (s *AchievementService) calculateAchievementRate(achievements []model.Achievement, references []model.AchievementReference) float64 {
	if len(references) == 0 {
		return 0
	}
	
	verifiedCount := 0
	for _, ref := range references {
		if ref.Status == "verified" {
			verifiedCount++
		}
	}
	
	return float64(verifiedCount) / float64(len(references)) * 100
}

func (s *AchievementService) calculateVerificationRate(statusCounts map[string]int) float64 {
	total := 0
	verified := statusCounts["verified"]
	
	for _, count := range statusCounts {
		total += count
	}
	
	if total == 0 {
		return 0
	}
	
	return float64(verified) / float64(total) * 100
}

func (s *AchievementService) getRecentAchievements(achievements []model.Achievement, limit int) []fiber.Map {
	// Sort by created_at desc and take limit
	var recent []fiber.Map
	
	// Simple implementation - in production, you'd sort properly
	count := 0
	for i := len(achievements) - 1; i >= 0 && count < limit; i-- {
		achievement := achievements[i]
		recent = append(recent, fiber.Map{
			"id": achievement.ID.Hex(),
			"title": achievement.Title,
			"category": achievement.Category,
			"created_at": achievement.CreatedAt,
		})
		count++
	}
	
	return recent
}

func (s *AchievementService) getTopStudents(studentStats map[string]int, students []model.Student, limit int) []fiber.Map {
	// Create student map for quick lookup
	studentMap := make(map[string]*model.Student)
	for i, student := range students {
		studentMap[student.UserID] = &students[i]
	}
	
	// Convert to slice for sorting
	type studentScore struct {
		StudentID string
		Count     int
		Student   *model.Student
	}
	
	var scores []studentScore
	for studentID, count := range studentStats {
		if student, exists := studentMap[studentID]; exists {
			scores = append(scores, studentScore{
				StudentID: studentID,
				Count:     count,
				Student:   student,
			})
		}
	}
	
	// Simple sorting (in production, use proper sorting)
	var topStudents []fiber.Map
	for i := 0; i < len(scores) && i < limit; i++ {
		score := scores[i]
		topStudents = append(topStudents, fiber.Map{
			"student_id": score.Student.StudentID,
			"program_study": score.Student.ProgramStudy,
			"achievement_count": score.Count,
			"rank": i + 1,
		})
	}
	
	return topStudents
}

func (s *AchievementService) getTopStudentsAdmin(studentStats map[string]int, students []model.Student, limit int) []fiber.Map {
	return s.getTopStudents(studentStats, students, limit)
}

func (s *AchievementService) calculateGrowthRate(monthlyStats map[string]int) float64 {
	// Simple growth calculation - compare last month with previous
	// In production, implement proper growth rate calculation
	return 0.0 // Placeholder
}

func (s *AchievementService) sumMapValues(m map[int]int) int {
	sum := 0
	for _, v := range m {
		sum += v
	}
	return sum
}

// Debug method to check data consistency
func (s *AchievementService) DebugDataRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	
	// Get all achievements for this user
	achievements, err := s.achievementRepo.GetByStudentID(userID)
	if err != nil {
		return c.JSON(fiber.Map{
			"error": "Failed to get achievements",
			"details": err.Error(),
		})
	}
	
	// Get all references for this user
	references, err := s.achievementRepo.GetReferencesByStudentID(userID)
	if err != nil {
		return c.JSON(fiber.Map{
			"error": "Failed to get references",
			"details": err.Error(),
		})
	}
	
	// Check for mismatches
	var mismatches []fiber.Map
	for _, ref := range references {
		found := false
		for _, achievement := range achievements {
			if achievement.ID.Hex() == ref.AchievementID {
				found = true
				break
			}
		}
		if !found {
			mismatches = append(mismatches, fiber.Map{
				"reference_id": ref.ID.Hex(),
				"achievement_id": ref.AchievementID,
				"status": ref.Status,
				"problem": "Achievement not found",
			})
		}
	}
	
	return c.JSON(fiber.Map{
		"user_id": userID,
		"achievements": achievements,
		"references": references,
		"mismatches": mismatches,
		"summary": fiber.Map{
			"achievement_count": len(achievements),
			"reference_count": len(references),
			"mismatch_count": len(mismatches),
		},
	})
}
// Fix data mismatches
func (s *AchievementService) FixDataMismatchesRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	
	// Get all achievements for this user
	achievements, err := s.achievementRepo.GetByStudentID(userID)
	if err != nil {
		return c.JSON(fiber.Map{
			"error": "Failed to get achievements",
			"details": err.Error(),
		})
	}
	
	// Get all references for this user
	references, err := s.achievementRepo.GetReferencesByStudentID(userID)
	if err != nil {
		return c.JSON(fiber.Map{
			"error": "Failed to get references",
			"details": err.Error(),
		})
	}
	
	// Create map of achievements for quick lookup
	achievementMap := make(map[string]*model.Achievement)
	for i, achievement := range achievements {
		achievementMap[achievement.ID.Hex()] = &achievements[i]
	}
	
	var fixes []fiber.Map
	var errors []fiber.Map
	
	// Fix each reference
	for _, ref := range references {
		// Skip if reference already points to existing achievement
		if _, exists := achievementMap[ref.AchievementID]; exists {
			continue
		}
		
		// Try to find the correct achievement by matching similar IDs or other criteria
		var correctAchievement *model.Achievement
		
		// Strategy 1: Find achievement with similar ID (off by 1 character)
		for _, achievement := range achievements {
			achievementID := achievement.ID.Hex()
			if len(achievementID) == len(ref.AchievementID) {
				// Count different characters
				diff := 0
				for i := 0; i < len(achievementID); i++ {
					if achievementID[i] != ref.AchievementID[i] {
						diff++
					}
				}
				// If only 1 character different, likely a match
				if diff == 1 {
					correctAchievement = &achievement
					break
				}
			}
		}
		
		// Strategy 2: If no similar ID found, match by creation time (within 1 second)
		if correctAchievement == nil {
			for _, achievement := range achievements {
				timeDiff := achievement.CreatedAt.Sub(ref.CreatedAt)
				if timeDiff < time.Second && timeDiff > -time.Second {
					correctAchievement = &achievement
					break
				}
			}
		}
		
		if correctAchievement != nil {
			// Update the reference
			ref.AchievementID = correctAchievement.ID.Hex()
			err := s.achievementRepo.UpdateReference(&ref)
			if err != nil {
				errors = append(errors, fiber.Map{
					"reference_id": ref.ID.Hex(),
					"error": err.Error(),
				})
			} else {
				fixes = append(fixes, fiber.Map{
					"reference_id": ref.ID.Hex(),
					"old_achievement_id": ref.AchievementID,
					"new_achievement_id": correctAchievement.ID.Hex(),
					"status": ref.Status,
				})
			}
		} else {
			errors = append(errors, fiber.Map{
				"reference_id": ref.ID.Hex(),
				"achievement_id": ref.AchievementID,
				"error": "No matching achievement found",
			})
		}
	}
	
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Data mismatch fix completed",
		"fixes_applied": fixes,
		"errors": errors,
		"summary": fiber.Map{
			"total_references": len(references),
			"fixes_applied": len(fixes),
			"errors": len(errors),
		},
	})
}
// Debug advisor relationship
func (s *AchievementService) DebugAdvisorRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	userRole := c.Locals("role").(string)
	
	if userRole != "lecturer" && userRole != "Dosen" && userRole != "Dosen Wali" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only lecturers can access this debug endpoint",
		})
	}
	
	// Get lecturer info
	lecturer, err := s.lecturerRepo.GetByUserID(userID)
	if err != nil {
		return c.JSON(fiber.Map{
			"error": "Lecturer not found",
			"details": err.Error(),
		})
	}
	
	// Get all students to see advisor relationships
	allStudents, err := s.studentRepo.GetAll()
	if err != nil {
		return c.JSON(fiber.Map{
			"error": "Failed to get students",
			"details": err.Error(),
		})
	}
	
	// Find students with this lecturer as advisor
	var adviseeStudents []model.Student
	var otherStudents []model.Student
	
	for _, student := range allStudents {
		if student.AdvisorID == lecturer.ID || student.AdvisorID == userID {
			adviseeStudents = append(adviseeStudents, student)
		} else {
			otherStudents = append(otherStudents, student)
		}
	}
	
	return c.JSON(fiber.Map{
		"lecturer_info": fiber.Map{
			"user_id": userID,
			"lecturer_id": lecturer.LecturerID,
			"lecturer_internal_id": lecturer.ID,
			"department": lecturer.Department,
		},
		"advisee_students": adviseeStudents,
		"other_students": otherStudents,
		"summary": fiber.Map{
			"total_students": len(allStudents),
			"advisee_count": len(adviseeStudents),
			"other_count": len(otherStudents),
		},
	})
}
// Fix advisor relationship - assign students to lecturers
func (s *AchievementService) FixAdvisorRelationshipRequest(c *fiber.Ctx) error {
	userRole := c.Locals("role").(string)
	
	// Only admin can fix advisor relationships
	if userRole != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only admin can fix advisor relationships",
		})
	}
	
	// Get all students without advisors
	allStudents, err := s.studentRepo.GetAll()
	if err != nil {
		return c.JSON(fiber.Map{
			"error": "Failed to get students",
			"details": err.Error(),
		})
	}
	
	// Get all lecturers
	allLecturers, err := s.lecturerRepo.GetAll()
	if err != nil {
		return c.JSON(fiber.Map{
			"error": "Failed to get lecturers",
			"details": err.Error(),
		})
	}
	
	if len(allLecturers) == 0 {
		return c.JSON(fiber.Map{
			"error": "No lecturers found",
		})
	}
	
	var fixes []fiber.Map
	var errors []fiber.Map
	
	// Assign students to the first available lecturer (for testing)
	defaultLecturer := allLecturers[0]
	
	for _, student := range allStudents {
		// If student doesn't have advisor or has invalid advisor
		if student.AdvisorID == "" {
			// Update student with advisor
			student.AdvisorID = defaultLecturer.UserID // Use lecturer's UserID
			
			err := s.studentRepo.Update(&student)
			if err != nil {
				errors = append(errors, fiber.Map{
					"student_id": student.StudentID,
					"error": err.Error(),
				})
			} else {
				fixes = append(fixes, fiber.Map{
					"student_id": student.StudentID,
					"student_user_id": student.UserID,
					"assigned_advisor_id": defaultLecturer.UserID,
					"advisor_name": defaultLecturer.LecturerID,
				})
			}
		}
	}
	
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Advisor relationship fix completed",
		"default_lecturer": fiber.Map{
			"user_id": defaultLecturer.UserID,
			"lecturer_id": defaultLecturer.LecturerID,
			"department": defaultLecturer.Department,
		},
		"fixes_applied": fixes,
		"errors": errors,
		"summary": fiber.Map{
			"total_students": len(allStudents),
			"fixes_applied": len(fixes),
			"errors": len(errors),
		},
	})
}
// Quick fix - assign current student to current lecturer
func (s *AchievementService) AssignStudentToLecturerRequest(c *fiber.Ctx) error {
	userRole := c.Locals("role").(string)
	
	// Only admin can assign students to lecturers
	if userRole != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only admin can assign students to lecturers",
		})
	}
	
	// Get request body
	var req struct {
		StudentUserID   string `json:"student_user_id"`
		LecturerUserID  string `json:"lecturer_user_id"`
	}
	
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	
	// Get student
	student, err := s.studentRepo.GetByUserID(req.StudentUserID)
	if err != nil {
		return c.JSON(fiber.Map{
			"error": "Student not found",
			"student_user_id": req.StudentUserID,
		})
	}
	
	// Get lecturer
	lecturer, err := s.lecturerRepo.GetByUserID(req.LecturerUserID)
	if err != nil {
		return c.JSON(fiber.Map{
			"error": "Lecturer not found",
			"lecturer_user_id": req.LecturerUserID,
		})
	}
	
	// Update student's advisor - use lecturer's internal ID, not UserID
	student.AdvisorID = lecturer.ID // Use lecturer's internal ID for foreign key
	err = s.studentRepo.Update(student)
	if err != nil {
		return c.JSON(fiber.Map{
			"error": "Failed to update student advisor",
			"details": err.Error(),
		})
	}
	
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Student assigned to lecturer successfully",
		"assignment": fiber.Map{
			"student": fiber.Map{
				"user_id": student.UserID,
				"student_id": student.StudentID,
				"program_study": student.ProgramStudy,
			},
			"lecturer": fiber.Map{
				"user_id": lecturer.UserID,
				"lecturer_id": lecturer.LecturerID,
				"department": lecturer.Department,
			},
		},
	})
}
// Debug all data - show students, lecturers, and relationships
func (s *AchievementService) DebugAllDataRequest(c *fiber.Ctx) error {
	// Get all students
	allStudents, err := s.studentRepo.GetAll()
	if err != nil {
		return c.JSON(fiber.Map{
			"error": "Failed to get students",
			"details": err.Error(),
		})
	}
	
	// Get all lecturers
	allLecturers, err := s.lecturerRepo.GetAll()
	if err != nil {
		return c.JSON(fiber.Map{
			"error": "Failed to get lecturers",
			"details": err.Error(),
		})
	}
	
	// Format student data with advisor info
	var studentData []fiber.Map
	for _, student := range allStudents {
		studentInfo := fiber.Map{
			"user_id": student.UserID,
			"student_id": student.StudentID,
			"program_study": student.ProgramStudy,
			"advisor_id": student.AdvisorID,
			"advisor_status": "no_advisor",
		}
		
		// Check if advisor exists
		if student.AdvisorID != "" {
			for _, lecturer := range allLecturers {
				if lecturer.ID == student.AdvisorID {
					studentInfo["advisor_status"] = "advisor_found"
					studentInfo["advisor_lecturer_id"] = lecturer.LecturerID
					studentInfo["advisor_user_id"] = lecturer.UserID
					break
				}
			}
			if studentInfo["advisor_status"] == "no_advisor" {
				studentInfo["advisor_status"] = "advisor_not_found"
			}
		}
		
		studentData = append(studentData, studentInfo)
	}
	
	// Format lecturer data
	var lecturerData []fiber.Map
	for _, lecturer := range allLecturers {
		lecturerInfo := fiber.Map{
			"internal_id": lecturer.ID,
			"user_id": lecturer.UserID,
			"lecturer_id": lecturer.LecturerID,
			"department": lecturer.Department,
		}
		lecturerData = append(lecturerData, lecturerInfo)
	}
	
	return c.JSON(fiber.Map{
		"students": studentData,
		"lecturers": lecturerData,
		"summary": fiber.Map{
			"total_students": len(allStudents),
			"total_lecturers": len(allLecturers),
		},
	})
}
// Temporary endpoint to show all submitted achievements for lecturer
func (s *AchievementService) GetAllSubmittedAchievementsRequest(c *fiber.Ctx) error {
	userRole := c.Locals("role").(string)
	
	// Only lecturers can access
	if userRole != "lecturer" && userRole != "Dosen" && userRole != "Dosen Wali" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only lecturers can access this endpoint",
		})
	}
	
	// Get all references with status "submitted"
	references, err := s.achievementRepo.GetReferencesByStatus("submitted")
	if err != nil {
		return c.JSON(fiber.Map{
			"error": "Failed to get submitted achievements",
			"details": err.Error(),
		})
	}
	
	var result []fiber.Map
	
	for _, ref := range references {
		// Get achievement detail
		achievementObjID, err := primitive.ObjectIDFromHex(ref.AchievementID)
		if err != nil {
			continue
		}
		
		achievement, err := s.achievementRepo.GetByID(achievementObjID)
		if err != nil {
			continue
		}
		
		// Get student info
		student, err := s.studentRepo.GetByUserID(ref.StudentID)
		if err != nil {
			continue
		}
		
		result = append(result, fiber.Map{
			"reference_id": ref.ID.Hex(),
			"achievement_id": ref.AchievementID,
			"status": ref.Status,
			"submitted_at": ref.SubmittedAt,
			"achievement": fiber.Map{
				"title": achievement.Title,
				"category": achievement.Category,
				"description": achievement.Description,
				"created_at": achievement.CreatedAt,
			},
			"student": fiber.Map{
				"student_id": student.StudentID,
				"program_study": student.ProgramStudy,
				"user_id": student.UserID,
			},
		})
	}
	
	return c.JSON(fiber.Map{
		"success": true,
		"message": "All submitted achievements retrieved",
		"data": result,
		"total": len(result),
	})
}
// Debug GetByAdvisorID method
func (s *AchievementService) DebugGetByAdvisorIDRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	userRole := c.Locals("role").(string)
	
	if userRole != "lecturer" && userRole != "Dosen" && userRole != "Dosen Wali" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only lecturers can access this endpoint",
		})
	}
	
	// Get lecturer info
	lecturer, err := s.lecturerRepo.GetByUserID(userID)
	if err != nil {
		return c.JSON(fiber.Map{
			"error": "Lecturer not found",
			"details": err.Error(),
		})
	}
	
	// Try GetByAdvisorID
	students, err := s.studentRepo.GetByAdvisorID(lecturer.ID)
	if err != nil {
		return c.JSON(fiber.Map{
			"error": "GetByAdvisorID failed",
			"lecturer_id": lecturer.ID,
			"details": err.Error(),
		})
	}
	
	// Get student user IDs
	var studentUserIDs []string
	for _, student := range students {
		studentUserIDs = append(studentUserIDs, student.UserID)
	}
	
	// Try to get references for these students
	references, err := s.achievementRepo.GetReferencesByStudentIDs(studentUserIDs, 10, 0)
	if err != nil {
		return c.JSON(fiber.Map{
			"error": "GetReferencesByStudentIDs failed",
			"student_user_ids": studentUserIDs,
			"details": err.Error(),
		})
	}
	
	return c.JSON(fiber.Map{
		"lecturer_info": fiber.Map{
			"user_id": lecturer.UserID,
			"internal_id": lecturer.ID,
			"lecturer_id": lecturer.LecturerID,
		},
		"students_found": students,
		"student_user_ids": studentUserIDs,
		"references_found": references,
		"debug_info": fiber.Map{
			"students_count": len(students),
			"references_count": len(references),
		},
	})
}

// Fix Achievement ID Mismatch - Method khusus untuk memperbaiki ketidakcocokan ID
func (s *AchievementService) FixAchievementIDMismatchRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	
	// Get all achievements for this user
	achievements, err := s.achievementRepo.GetByStudentID(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get achievements",
			"details": err.Error(),
		})
	}
	
	// Get all references for this user
	references, err := s.achievementRepo.GetReferencesByStudentID(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get references",
			"details": err.Error(),
		})
	}
	
	var fixes []fiber.Map
	var errors []fiber.Map
	
	// Create map of existing achievement IDs
	achievementIDs := make(map[string]bool)
	for _, achievement := range achievements {
		achievementIDs[achievement.ID.Hex()] = true
	}
	
	// Fix each reference that has mismatched achievement_id
	for _, ref := range references {
		// Skip if reference already points to existing achievement
		if achievementIDs[ref.AchievementID] {
			continue
		}
		
		// Find the correct achievement for this reference
		// Strategy: Match by creation time (within 5 seconds) and student_id
		var correctAchievement *model.Achievement
		for _, achievement := range achievements {
			// Check if student_id matches and creation time is close
			if achievement.StudentID == ref.StudentID {
				timeDiff := achievement.CreatedAt.Sub(ref.CreatedAt)
				if timeDiff < 5*time.Second && timeDiff > -5*time.Second {
					correctAchievement = &achievement
					break
				}
			}
		}
		
		if correctAchievement != nil {
			// Update the reference with correct achievement_id
			oldAchievementID := ref.AchievementID
			ref.AchievementID = correctAchievement.ID.Hex()
			
			err := s.achievementRepo.UpdateReference(&ref)
			if err != nil {
				errors = append(errors, fiber.Map{
					"reference_id": ref.ID.Hex(),
					"old_achievement_id": oldAchievementID,
					"new_achievement_id": correctAchievement.ID.Hex(),
					"error": err.Error(),
				})
			} else {
				fixes = append(fixes, fiber.Map{
					"reference_id": ref.ID.Hex(),
					"old_achievement_id": oldAchievementID,
					"new_achievement_id": correctAchievement.ID.Hex(),
					"achievement_title": correctAchievement.Title,
					"status": "fixed",
				})
			}
		} else {
			errors = append(errors, fiber.Map{
				"reference_id": ref.ID.Hex(),
				"achievement_id": ref.AchievementID,
				"error": "No matching achievement found",
			})
		}
	}
	
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Achievement ID mismatch fix completed",
		"user_id": userID,
		"summary": fiber.Map{
			"total_achievements": len(achievements),
			"total_references": len(references),
			"fixes_applied": len(fixes),
			"errors_encountered": len(errors),
		},
		"fixes": fixes,
		"errors": errors,
		"next_steps": []string{
			"Try submitting your achievement again",
			"The achievement ID should now match correctly",
		},
	})
}
// Comprehensive ID Fix - Memperbaiki semua masalah ID mismatch
func (s *AchievementService) ComprehensiveIDFixRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	
	// Get all achievements for this user
	achievements, err := s.achievementRepo.GetByStudentID(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get achievements",
			"details": err.Error(),
		})
	}
	
	// Get all references for this user
	references, err := s.achievementRepo.GetReferencesByStudentID(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get references",
			"details": err.Error(),
		})
	}
	
	var fixes []fiber.Map
	var errors []fiber.Map
	var orphanedReferences []fiber.Map
	var orphanedAchievements []fiber.Map
	
	// Create maps for quick lookup
	achievementMap := make(map[string]*model.Achievement)
	for i, achievement := range achievements {
		achievementMap[achievement.ID.Hex()] = &achievements[i]
	}
	
	referenceMap := make(map[string]*model.AchievementReference)
	for i, ref := range references {
		referenceMap[ref.AchievementID] = &references[i]
	}
	
	// Step 1: Fix references that point to non-existent achievements
	for i, ref := range references {
		if _, exists := achievementMap[ref.AchievementID]; !exists {
			// Try to find matching achievement by creation time and student_id
			var matchedAchievement *model.Achievement
			
			for _, achievement := range achievements {
				if achievement.StudentID == ref.StudentID {
					// Check if creation times are close (within 10 seconds)
					timeDiff := achievement.CreatedAt.Sub(ref.CreatedAt)
					if timeDiff < 10*time.Second && timeDiff > -10*time.Second {
						// Check if this achievement doesn't already have a reference
						if _, hasRef := referenceMap[achievement.ID.Hex()]; !hasRef {
							matchedAchievement = &achievement
							break
						}
					}
				}
			}
			
			if matchedAchievement != nil {
				// Update reference with correct achievement_id
				oldAchievementID := ref.AchievementID
				references[i].AchievementID = matchedAchievement.ID.Hex()
				
				err := s.achievementRepo.UpdateReference(&references[i])
				if err != nil {
					errors = append(errors, fiber.Map{
						"type": "reference_update_failed",
						"reference_id": ref.ID.Hex(),
						"old_achievement_id": oldAchievementID,
						"new_achievement_id": matchedAchievement.ID.Hex(),
						"error": err.Error(),
					})
				} else {
					fixes = append(fixes, fiber.Map{
						"type": "reference_fixed",
						"reference_id": ref.ID.Hex(),
						"old_achievement_id": oldAchievementID,
						"new_achievement_id": matchedAchievement.ID.Hex(),
						"achievement_title": matchedAchievement.Title,
						"status": ref.Status,
					})
					// Update the map
					referenceMap[matchedAchievement.ID.Hex()] = &references[i]
					delete(referenceMap, oldAchievementID)
				}
			} else {
				orphanedReferences = append(orphanedReferences, fiber.Map{
					"reference_id": ref.ID.Hex(),
					"achievement_id": ref.AchievementID,
					"status": ref.Status,
					"created_at": ref.CreatedAt,
				})
			}
		}
	}
	
	// Step 2: Find achievements without references and create them
	for _, achievement := range achievements {
		if _, hasRef := referenceMap[achievement.ID.Hex()]; !hasRef {
			// Create missing reference
			newRef := &model.AchievementReference{
				StudentID:     achievement.StudentID,
				AchievementID: achievement.ID.Hex(),
				Status:        "draft", // Default status
				CreatedAt:     achievement.CreatedAt,
				UpdatedAt:     time.Now(),
			}
			
			err := s.achievementRepo.CreateReference(newRef)
			if err != nil {
				errors = append(errors, fiber.Map{
					"type": "reference_creation_failed",
					"achievement_id": achievement.ID.Hex(),
					"achievement_title": achievement.Title,
					"error": err.Error(),
				})
			} else {
				fixes = append(fixes, fiber.Map{
					"type": "reference_created",
					"achievement_id": achievement.ID.Hex(),
					"achievement_title": achievement.Title,
					"reference_id": newRef.ID.Hex(),
					"status": newRef.Status,
				})
			}
		}
	}
	
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Comprehensive ID fix completed",
		"user_id": userID,
		"summary": fiber.Map{
			"total_achievements": len(achievements),
			"total_references": len(references),
			"fixes_applied": len(fixes),
			"errors_encountered": len(errors),
			"orphaned_references": len(orphanedReferences),
			"orphaned_achievements": len(orphanedAchievements),
		},
		"fixes": fixes,
		"errors": errors,
		"orphaned_references": orphanedReferences,
		"orphaned_achievements": orphanedAchievements,
		"next_steps": []string{
			"All ID mismatches have been fixed",
			"You can now submit, delete, or update your achievements",
			"Orphaned references may need manual cleanup",
		},
	})
}

// Safe method to get reference by achievement ID with better error handling
func (s *AchievementService) GetReferenceByAchievementIDSafe(achievementID string) (*model.AchievementReference, error) {
	// First try direct lookup
	reference, err := s.achievementRepo.GetReferenceByAchievementID(achievementID)
	if err == nil {
		return reference, nil
	}
	
	// If not found, try to find by similar ID (in case of minor mismatch)
	// Get achievement to find student_id
	achievementObjID, err := primitive.ObjectIDFromHex(achievementID)
	if err != nil {
		return nil, errors.New("invalid achievement ID format")
	}
	
	achievement, err := s.achievementRepo.GetByID(achievementObjID)
	if err != nil {
		return nil, errors.New("achievement not found")
	}
	
	// Get all references for this student
	references, err := s.achievementRepo.GetReferencesByStudentID(achievement.StudentID)
	if err != nil {
		return nil, err
	}
	
	// Try to find reference by creation time match
	for _, ref := range references {
		timeDiff := achievement.CreatedAt.Sub(ref.CreatedAt)
		if timeDiff < 5*time.Second && timeDiff > -5*time.Second {
			return &ref, nil
		}
	}
	
	return nil, errors.New("reference not found")
}
// Fix New Achievement ID - Memperbaiki achievement yang baru dibuat dengan ID yang salah
func (s *AchievementService) FixNewAchievementIDRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	achievementIDParam := c.Params("achievement_id")
	
	if achievementIDParam == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Achievement ID is required",
			"message": "Please provide achievement ID in URL path",
		})
	}
	
	// Get all achievements for this user
	achievements, err := s.achievementRepo.GetByStudentID(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get achievements",
			"details": err.Error(),
		})
	}
	
	// Find the achievement by ID.Hex() or by creation time match
	var targetAchievement *model.Achievement
	
	// First try to find by exact ID match
	for _, achievement := range achievements {
		if achievement.ID.Hex() == achievementIDParam {
			targetAchievement = &achievement
			break
		}
	}
	
	// If not found by ID, try to find by reference lookup
	if targetAchievement == nil {
		reference, err := s.achievementRepo.GetReferenceByAchievementID(achievementIDParam)
		if err == nil {
			// Find achievement by student and creation time match
			for _, achievement := range achievements {
				if achievement.StudentID == reference.StudentID {
					timeDiff := achievement.CreatedAt.Sub(reference.CreatedAt)
					if timeDiff < 5*time.Second && timeDiff > -5*time.Second {
						targetAchievement = &achievement
						break
					}
				}
			}
		}
	}
	
	if targetAchievement == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Achievement not found",
			"message": "No achievement found with the provided ID",
		})
	}
	
	// Get the reference that points to wrong achievement_id
	reference, err := s.achievementRepo.GetReferenceByAchievementID(achievementIDParam)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Reference not found",
			"message": "No reference found for this achievement",
		})
	}
	
	// Update reference to point to correct achievement ID
	reference.AchievementID = targetAchievement.ID.Hex()
	err = s.achievementRepo.UpdateReference(reference)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update reference",
			"details": err.Error(),
		})
	}
	
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Achievement ID fixed successfully",
		"data": fiber.Map{
			"achievement_id": targetAchievement.ID.Hex(),
			"reference_id": reference.ID.Hex(),
			"old_achievement_id": achievementIDParam,
			"new_achievement_id": targetAchievement.ID.Hex(),
			"achievement_title": targetAchievement.Title,
			"status": reference.Status,
		},
		"next_steps": []string{
			"Now you can submit this achievement using the correct ID: " + targetAchievement.ID.Hex(),
			"Use: POST /api/achievements/" + targetAchievement.ID.Hex() + "/submit",
		},
	})
}
// Clean Legacy ObjectID - Membersihkan field ObjectID yang tidak diperlukan dari database
func (s *AchievementService) CleanLegacyObjectIDRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	
	// Get all achievements for this user
	achievements, err := s.achievementRepo.GetByStudentID(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get achievements",
			"details": err.Error(),
		})
	}
	
	var cleaned []fiber.Map
	var errors []fiber.Map
	
	for _, achievement := range achievements {
		// Update achievement to remove ObjectID field (if it exists in database)
		// This will be handled by the model change - just update to refresh
		achievement.UpdatedAt = time.Now()
		
		err := s.achievementRepo.Update(&achievement)
		if err != nil {
			errors = append(errors, fiber.Map{
				"achievement_id": achievement.ID.Hex(),
				"title": achievement.Title,
				"error": err.Error(),
			})
		} else {
			cleaned = append(cleaned, fiber.Map{
				"achievement_id": achievement.ID.Hex(),
				"title": achievement.Title,
				"status": "cleaned",
			})
		}
	}
	
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Legacy ObjectID field cleanup completed",
		"user_id": userID,
		"summary": fiber.Map{
			"total_achievements": len(achievements),
			"cleaned_count": len(cleaned),
			"error_count": len(errors),
		},
		"cleaned": cleaned,
		"errors": errors,
		"note": "ObjectID field has been removed from the model. All new achievements will use proper ID field.",
	})
}
// DebugAdminAchievementsRequest - Debug admin view all achievements
func (s *AchievementService) DebugAdminAchievementsRequest(c *fiber.Ctx) error {
	userRole := c.Locals("role").(string)
	
	// Only admin can debug
	if userRole != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only admin can debug achievements",
		})
	}

	// Get query parameters
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	status := c.Query("status", "")
	studentID := c.Query("student_id", "")

	// Step 1: Get achievement references
	references, total, err := s.achievementRepo.GetAllReferencesWithFilters(page, limit, status, studentID)
	if err != nil {
		return c.JSON(fiber.Map{
			"step": "get_references",
			"error": err.Error(),
		})
	}

	if len(references) == 0 {
		return c.JSON(fiber.Map{
			"step": "get_references",
			"message": "No references found",
			"total": total,
		})
	}

	// Step 2: Extract student IDs (filter out empty strings)
	studentIDs := make(map[string]bool)
	for _, ref := range references {
		if ref.StudentID != "" && len(ref.StudentID) > 0 {
			studentIDs[ref.StudentID] = true
		}
	}

	var studentIDList []string
	for id := range studentIDs {
		if id != "" && len(id) > 0 {
			studentIDList = append(studentIDList, id)
		}
	}

	// Step 3: Try to get students
	students, err := s.studentRepo.GetStudentsByUserIDs(studentIDList)
	if err != nil {
		return c.JSON(fiber.Map{
			"step": "get_students",
			"error": err.Error(),
			"student_ids_requested": studentIDList,
			"total_references": len(references),
		})
	}

	return c.JSON(fiber.Map{
		"step": "success",
		"data": fiber.Map{
			"total_references": len(references),
			"unique_student_ids": len(studentIDList),
			"students_found": len(students),
			"student_ids_requested": studentIDList,
			"students_found_data": students,
			"sample_references": references[:min(3, len(references))],
		},
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}