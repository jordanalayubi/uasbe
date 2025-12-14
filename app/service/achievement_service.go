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
	achievementRepo *repository.AchievementRepository
	studentRepo     *repository.StudentRepository
	lecturerRepo    *repository.LecturerRepository
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

func NewAchievementService(achievementRepo *repository.AchievementRepository, studentRepo *repository.StudentRepository, lecturerRepo *repository.LecturerRepository) *AchievementService {
	return &AchievementService{
		achievementRepo: achievementRepo,
		studentRepo:     studentRepo,
		lecturerRepo:    lecturerRepo,
	}
}

// FR-003: Submit Prestasi - Handler untuk mahasiswa submit prestasi
func (s *AchievementService) HandleCreateAchievementRequest(c *fiber.Ctx) error {
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

func (s *AchievementService) HandleGetStudentAchievements(c *fiber.Ctx) error {
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

func (s *AchievementService) HandleGetStudentAchievementReferences(c *fiber.Ctx) error {
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

func (s *AchievementService) HandleGetAchievementByID(c *fiber.Ctx) error {
	idParam := c.Params("id")

	// Validate and convert ID
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid achievement ID",
		})
	}

	// Get achievement
	achievement, err := s.GetAchievementByID(id)
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

func (s *AchievementService) HandleUpdateAchievementRequest(c *fiber.Ctx) error {
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

func (s *AchievementService) HandleDeleteAchievementRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	idParam := c.Params("id")

	// Validate and convert ID
	id, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid achievement ID",
		})
	}

	// Process deletion
	err = s.DeleteAchievement(userID, id)
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

func (s *AchievementService) HandleSubmitAchievementRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	achievementID := c.Params("achievement_id")

	err := s.SubmitAchievementForVerification(userID, achievementID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Achievement submitted for verification",
	})
}

func (s *AchievementService) HandleGetPendingVerifications(c *fiber.Ctx) error {
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

func (s *AchievementService) HandleVerifyAchievementRequest(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	refIDParam := c.Params("reference_id")
	
	var req VerifyAchievementRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if req.Status != "verified" && req.Status != "rejected" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Status must be 'verified' or 'rejected'",
		})
	}

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

	// Process verification
	err = s.VerifyAchievement(userID, refID, &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Achievement verification updated",
	})
}

// HandleUploadAttachment - Handle file upload for achievement attachments
func (s *AchievementService) HandleUploadAttachment(c *fiber.Ctx) error {
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

func (s *AchievementService) SubmitAchievementForVerification(studentID string, achievementID string) error {
	// Find the reference
	references, err := s.achievementRepo.GetReferencesByStudentID(studentID)
	if err != nil {
		return err
	}

	var targetRef *model.AchievementReference
	for _, ref := range references {
		if ref.AchievementID == achievementID {
			targetRef = &ref
			break
		}
	}

	if targetRef == nil {
		return errors.New("achievement not found")
	}

	if targetRef.Status != "draft" {
		return errors.New("achievement already submitted")
	}

	// Update status to submitted
	targetRef.Status = "submitted"
	targetRef.SubmittedAt = time.Now()

	return s.achievementRepo.UpdateReference(targetRef)
}

func (s *AchievementService) VerifyAchievement(lecturerID string, referenceID primitive.ObjectID, req *VerifyAchievementRequest) error {
	// Get the reference
	reference, err := s.achievementRepo.GetReferenceByID(referenceID)
	if err != nil {
		return errors.New("achievement reference not found")
	}

	if reference.Status != "submitted" {
		return errors.New("achievement is not in submitted status")
	}

	// Check if lecturer is the advisor of the student
	student, err := s.studentRepo.GetByID(reference.StudentID)
	if err != nil {
		return errors.New("student not found")
	}

	lecturer, err := s.lecturerRepo.GetByUserID(lecturerID)
	if err != nil {
		return errors.New("lecturer not found")
	}

	if student.AdvisorID != lecturer.ID {
		return errors.New("you can only verify achievements of your advisees")
	}

	// Update reference
	reference.Status = req.Status
	reference.VerifiedBy = lecturerID
	reference.VerifiedAt = time.Now()
	if req.Status == "rejected" {
		reference.RejectionNote = req.RejectionNote
	}

	return s.achievementRepo.UpdateReference(reference)
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
		references, err := s.achievementRepo.GetReferencesByStudentID(student.ID)
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

func (s *AchievementService) GetAchievementByID(id primitive.ObjectID) (*model.Achievement, error) {
	return s.achievementRepo.GetByID(id)
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

func (s *AchievementService) DeleteAchievement(studentID string, achievementID primitive.ObjectID) error {
	// Get existing achievement
	achievement, err := s.achievementRepo.GetByID(achievementID)
	if err != nil {
		return errors.New("achievement not found")
	}

	// Check ownership
	if achievement.StudentID != studentID {
		return errors.New("unauthorized")
	}

	return s.achievementRepo.Delete(achievementID)
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