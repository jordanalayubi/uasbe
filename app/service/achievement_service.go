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

	// Get all student IDs for batch lookup
	studentIDs := make(map[string]bool)
	for _, ref := range references {
		studentIDs[ref.StudentID] = true
	}

	// Get student info batch
	var studentIDList []string
	for id := range studentIDs {
		studentIDList = append(studentIDList, id)
	}

	students, err := s.studentRepo.GetStudentsByUserIDs(studentIDList)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get student information",
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
				"program_study": student.ProgramStudy,
				"user_id":       student.UserID,
			}
		}

		result = append(result, fiber.Map{
			"achievement":  achievement,
			"reference":    ref,
			"student_info": studentInfo,
		})
	}

	// Calculate pagination
	totalPages := (total + limit - 1) / limit

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result,
		"pagination": fiber.Map{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
		"filters": fiber.Map{
			"status":     status,
			"category":   category,
			"student_id": studentID,
			"sort_by":    sortBy,
			"sort_order": sortOrder,
		},
	})
}

// FR-003: Submit Prestasi - Service method untuk mahasiswa submit prestasi
func (s *AchievementService) CreateAchievementRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	userRole := c.Locals("role").(string)
	
	// FR-003 Precondition: User terautentikasi sebagai mahasiswa
	if userRole != "student" && userRole != "Mahasiswa" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only students can submit achievements",
		})
	}
	
	var req CreateAchievementRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// FR-003 Step 1: Mahasiswa mengisi data prestasi - Validate request
	if req.Category == "" || req.Title == "" || req.Description == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Category, title, and description are required",
		})
	}

	// FR-003 Step 2: Mahasiswa upload dokumen pendukung (handled in request)
	// FR-003 Step 3: Sistem simpan ke MongoDB (achievement) dan PostgreSQL (reference)
	// FR-003 Step 4: Status awal: 'draft'
	achievement, reference, err := s.SubmitAchievement(userID, &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// FR-003 Step 5: Return achievement data
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Achievement submitted successfully",
		"data": fiber.Map{
			"achievement": achievement,
			"reference":   reference,
		},
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
	idParam := c.Params("id")

	// FR-005 Precondition: Hanya mahasiswa yang bisa hapus prestasi mereka sendiri
	if userRole != "student" && userRole != "Mahasiswa" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only students can delete their own achievements",
		})
	}

	// Validate and convert ID
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid achievement ID",
		})
	}

	// FR-005 Flow: Soft delete prestasi
	err = s.SoftDeleteAchievement(userID, id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Achievement deleted successfully",
	})
}

// FR-004: Submit untuk Verifikasi - Service method untuk submit prestasi untuk verifikasi
func (s *AchievementService) SubmitAchievementRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	userRole := c.Locals("role").(string)
	achievementID := c.Params("achievement_id")

	// FR-004 Precondition: Hanya mahasiswa yang bisa submit
	if userRole != "student" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only students can submit achievements for verification",
		})
	}

	// FR-004 Flow: Submit prestasi untuk verifikasi
	updatedReference, err := s.SubmitAchievementForVerification(userID, achievementID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Achievement submitted for verification",
		"data":    updatedReference,
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
	refIDParam := c.Params("reference_id")

	// FR-007 Precondition: Hanya dosen wali yang bisa verify
	if userRole != "lecturer" && userRole != "Dosen" && userRole != "Dosen Wali" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only lecturers can verify achievements",
		})
	}
	
	var req VerifyAchievementRequest
	if err := c.BodyParser(&req); err != nil {
		// Debug: log the error and body content
		fmt.Printf("DEBUG: Body parsing error: %v\n", err)
		fmt.Printf("DEBUG: Request body: %s\n", string(c.Body()))
		fmt.Printf("DEBUG: Content-Type: %s\n", c.Get("Content-Type"))
		
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body: " + err.Error(),
		})
	}

	// Debug: log parsed request
	fmt.Printf("DEBUG: Parsed request - Status: %s, RejectionNote: %s\n", req.Status, req.RejectionNote)

	// FR-007/FR-008 Step 1 & 2: Validate request - dosen approve atau reject prestasi
	if req.Status != "verified" && req.Status != "rejected" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Status must be 'verified' or 'rejected'",
		})
	}

	// FR-008 Step 1: Dosen input rejection note (required when rejecting)
	if req.Status == "rejected" && req.RejectionNote == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Rejection note is required when rejecting",
		})
	}

	// Validate and convert ID
	refID, err := primitive.ObjectIDFromHex(refIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid reference ID",
		})
	}

	// FR-007/FR-008 Flow: Process verification/rejection
	updatedReference, err := s.VerifyAchievementWithDetails(userID, refID, &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// FR-007/FR-008 Step 5: Return updated status
	var message string
	if req.Status == "verified" {
		message = "Achievement verified successfully"
	} else {
		message = "Achievement rejected successfully"
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": message,
		"data": fiber.Map{
			"reference_id":   updatedReference.ID.Hex(),
			"status":         updatedReference.Status,
			"verified_by":    updatedReference.VerifiedBy,
			"verified_at":    updatedReference.VerifiedAt,
			"rejection_note": updatedReference.RejectionNote,
		},
	})
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
		ObjectID:     primitive.NewObjectID().Hex(),
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
		AchievementID: achievement.ObjectID,
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
	fmt.Printf("DEBUG: Reference status: '%s'\n", reference.Status)
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

	// FR-005 Step 4: Get achievement reference to check status
	references, err := s.achievementRepo.GetReferencesByStudentID(studentID)
	if err != nil {
		return errors.New("failed to get achievement references")
	}

	var targetRef *model.AchievementReference
	for _, ref := range references {
		if ref.AchievementID == achievement.ObjectID {
			targetRef = &ref
			break
		}
	}

	if targetRef == nil {
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

	// FR-006 Precondition: Hanya dosen yang bisa akses
	if userRole != "lecturer" && userRole != "Dosen" && userRole != "Dosen Wali" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only lecturers can view advisee achievements",
		})
	}

	// Parse pagination parameters
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	// FR-006 Flow: Get advisee achievements with pagination
	result, err := s.GetAdviseeAchievements(userID, limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    result.Achievements,
		"pagination": fiber.Map{
			"page":        page,
			"limit":       limit,
			"total":       result.Total,
			"total_pages": (result.Total + int64(limit) - 1) / int64(limit),
		},
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
		// Get achievement detail from MongoDB
		achievement, err := s.achievementRepo.GetByObjectID(ref.AchievementID)
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