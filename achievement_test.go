package main

import (
	"UASBE/app/model"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ==================== MOCK REPOSITORIES ====================

// MockAchievementRepository adalah repository tiruan untuk testing
type MockAchievementRepository struct {
	achievements map[primitive.ObjectID]*model.Achievement
	references   map[primitive.ObjectID]*model.AchievementReference
}

// NewMockAchievementRepository membuat mock repository baru
func NewMockAchievementRepository() *MockAchievementRepository {
	return &MockAchievementRepository{
		achievements: make(map[primitive.ObjectID]*model.Achievement),
		references:   make(map[primitive.ObjectID]*model.AchievementReference),
	}
}

// Create implement AchievementRepository.Create
func (m *MockAchievementRepository) Create(achievement *model.Achievement) error {
	if achievement.Title == "" {
		return errors.New("achievement title cannot be empty")
	}
	if achievement.Category == "" {
		return errors.New("achievement category cannot be empty")
	}
	
	achievement.ID = primitive.NewObjectID()
	achievement.CreatedAt = time.Now()
	achievement.UpdatedAt = time.Now()
	
	m.achievements[achievement.ID] = achievement
	return nil
}

// CreateReference implement AchievementRepository.CreateReference
func (m *MockAchievementRepository) CreateReference(ref *model.AchievementReference) error {
	if ref.StudentID == "" {
		return errors.New("student ID cannot be empty")
	}
	if ref.AchievementID == "" {
		return errors.New("achievement ID cannot be empty")
	}
	
	ref.ID = primitive.NewObjectID()
	ref.CreatedAt = time.Now()
	ref.UpdatedAt = time.Now()
	
	m.references[ref.ID] = ref
	return nil
}

// GetByID implement AchievementRepository.GetByID
func (m *MockAchievementRepository) GetByID(id primitive.ObjectID) (*model.Achievement, error) {
	if achievement, exists := m.achievements[id]; exists {
		return achievement, nil
	}
	return nil, errors.New("achievement not found")
}

// GetByIDActive implement AchievementRepository.GetByIDActive
func (m *MockAchievementRepository) GetByIDActive(id primitive.ObjectID) (*model.Achievement, error) {
	if achievement, exists := m.achievements[id]; exists {
		if achievement.DeletedAt == nil {
			return achievement, nil
		}
	}
	return nil, errors.New("achievement not found")
}

// GetReferenceByID implement AchievementRepository.GetReferenceByID
func (m *MockAchievementRepository) GetReferenceByID(id primitive.ObjectID) (*model.AchievementReference, error) {
	if ref, exists := m.references[id]; exists {
		return ref, nil
	}
	return nil, errors.New("reference not found")
}

// GetByStudentID implement AchievementRepository.GetByStudentID
func (m *MockAchievementRepository) GetByStudentID(studentID string) ([]model.Achievement, error) {
	var achievements []model.Achievement
	for _, achievement := range m.achievements {
		if achievement.StudentID == studentID && achievement.DeletedAt == nil {
			achievements = append(achievements, *achievement)
		}
	}
	return achievements, nil
}

// GetReferencesByStudentID implement AchievementRepository.GetReferencesByStudentID
func (m *MockAchievementRepository) GetReferencesByStudentID(studentID string) ([]model.AchievementReference, error) {
	var references []model.AchievementReference
	for _, ref := range m.references {
		if ref.StudentID == studentID {
			references = append(references, *ref)
		}
	}
	return references, nil
}

// GetReferencesByStatus implement AchievementRepository.GetReferencesByStatus
func (m *MockAchievementRepository) GetReferencesByStatus(status string) ([]model.AchievementReference, error) {
	var references []model.AchievementReference
	for _, ref := range m.references {
		if ref.Status == status {
			references = append(references, *ref)
		}
	}
	return references, nil
}

// Update implement AchievementRepository.Update
func (m *MockAchievementRepository) Update(achievement *model.Achievement) error {
	if _, exists := m.achievements[achievement.ID]; !exists {
		return errors.New("achievement not found")
	}
	achievement.UpdatedAt = time.Now()
	m.achievements[achievement.ID] = achievement
	return nil
}

// UpdateReference implement AchievementRepository.UpdateReference
func (m *MockAchievementRepository) UpdateReference(ref *model.AchievementReference) error {
	if _, exists := m.references[ref.ID]; !exists {
		return errors.New("reference not found")
	}
	ref.UpdatedAt = time.Now()
	m.references[ref.ID] = ref
	return nil
}

// GetReferenceByAchievementID implement AchievementRepository.GetReferenceByAchievementID
func (m *MockAchievementRepository) GetReferenceByAchievementID(achievementID string) (*model.AchievementReference, error) {
	for _, ref := range m.references {
		if ref.AchievementID == achievementID {
			return ref, nil
		}
	}
	return nil, errors.New("reference not found")
}

// MockStudentRepository adalah repository tiruan untuk testing
type MockStudentRepository struct {
	students map[string]*model.Student
}

// NewMockStudentRepository membuat mock repository baru
func NewMockStudentRepository() *MockStudentRepository {
	return &MockStudentRepository{
		students: make(map[string]*model.Student),
	}
}

// GetByUserID implement StudentRepository.GetByUserID
func (m *MockStudentRepository) GetByUserID(userID string) (*model.Student, error) {
	if student, exists := m.students[userID]; exists {
		return student, nil
	}
	return nil, errors.New("student not found")
}

// GetByAdvisorID implement StudentRepository.GetByAdvisorID
func (m *MockStudentRepository) GetByAdvisorID(advisorID string) ([]model.Student, error) {
	var students []model.Student
	for _, student := range m.students {
		if student.AdvisorID == advisorID {
			students = append(students, *student)
		}
	}
	return students, nil
}

// Update implement StudentRepository.Update
func (m *MockStudentRepository) Update(student *model.Student) error {
	if _, exists := m.students[student.UserID]; !exists {
		return errors.New("student not found")
	}
	m.students[student.UserID] = student
	return nil
}

// GetAll implement StudentRepository.GetAll
func (m *MockStudentRepository) GetAll() ([]model.Student, error) {
	var students []model.Student
	for _, student := range m.students {
		students = append(students, *student)
	}
	return students, nil
}

// AddStudent helper method untuk testing
func (m *MockStudentRepository) AddStudent(student *model.Student) {
	m.students[student.UserID] = student
}

// MockLecturerRepository adalah repository tiruan untuk testing
type MockLecturerRepository struct {
	lecturers map[string]*model.Lecturer
}

// NewMockLecturerRepository membuat mock repository baru
func NewMockLecturerRepository() *MockLecturerRepository {
	return &MockLecturerRepository{
		lecturers: make(map[string]*model.Lecturer),
	}
}

// GetByUserID implement LecturerRepository.GetByUserID
func (m *MockLecturerRepository) GetByUserID(userID string) (*model.Lecturer, error) {
	if lecturer, exists := m.lecturers[userID]; exists {
		return lecturer, nil
	}
	return nil, errors.New("lecturer not found")
}

// GetAll implement LecturerRepository.GetAll
func (m *MockLecturerRepository) GetAll() ([]model.Lecturer, error) {
	var lecturers []model.Lecturer
	for _, lecturer := range m.lecturers {
		lecturers = append(lecturers, *lecturer)
	}
	return lecturers, nil
}

// AddLecturer helper method untuk testing
func (m *MockLecturerRepository) AddLecturer(lecturer *model.Lecturer) {
	m.lecturers[lecturer.UserID] = lecturer
}

// MockNotificationService adalah notification service tiruan untuk testing
type MockNotificationService struct {
	notifications []string
}

// NewMockNotificationService membuat mock notification service baru
func NewMockNotificationService() *MockNotificationService {
	return &MockNotificationService{
		notifications: make([]string, 0),
	}
}

// SendNotification mock implementation
func (m *MockNotificationService) SendNotification(userID, message string) error {
	m.notifications = append(m.notifications, userID+": "+message)
	return nil
}

// ==================== BUSINESS LOGIC SERVICES ====================

// AchievementBusinessLogic menangani business logic untuk achievement
type AchievementBusinessLogic struct {
	achievementRepo *MockAchievementRepository
	studentRepo     *MockStudentRepository
	lecturerRepo    *MockLecturerRepository
	notificationSvc *MockNotificationService
}

// NewAchievementBusinessLogic create instance
func NewAchievementBusinessLogic(
	achievementRepo *MockAchievementRepository,
	studentRepo *MockStudentRepository,
	lecturerRepo *MockLecturerRepository,
	notificationSvc *MockNotificationService,
) *AchievementBusinessLogic {
	return &AchievementBusinessLogic{
		achievementRepo: achievementRepo,
		studentRepo:     studentRepo,
		lecturerRepo:    lecturerRepo,
		notificationSvc: notificationSvc,
	}
}

// CreateAchievement membuat achievement baru
func (s *AchievementBusinessLogic) CreateAchievement(studentID, category, title, description string) (*model.Achievement, *model.AchievementReference, error) {
	// Validasi input
	if studentID == "" {
		return nil, nil, errors.New("student ID cannot be empty")
	}
	if category == "" {
		return nil, nil, errors.New("category cannot be empty")
	}
	if title == "" {
		return nil, nil, errors.New("title cannot be empty")
	}
	if description == "" {
		return nil, nil, errors.New("description cannot be empty")
	}

	// Validasi kategori
	validCategories := []string{"competition", "research", "community_service", "academic", "organization"}
	isValidCategory := false
	for _, validCat := range validCategories {
		if category == validCat {
			isValidCategory = true
			break
		}
	}
	if !isValidCategory {
		return nil, nil, errors.New("invalid category")
	}

	// Cek apakah student ada
	_, err := s.studentRepo.GetByUserID(studentID)
	if err != nil {
		return nil, nil, errors.New("student not found")
	}

	// Buat achievement
	achievement := &model.Achievement{
		StudentID:   studentID,
		Category:    category,
		Title:       title,
		Description: description,
	}

	err = s.achievementRepo.Create(achievement)
	if err != nil {
		return nil, nil, err
	}

	// Buat reference
	reference := &model.AchievementReference{
		StudentID:     studentID,
		AchievementID: achievement.ID.Hex(),
		Status:        "draft",
	}

	err = s.achievementRepo.CreateReference(reference)
	if err != nil {
		return nil, nil, err
	}

	return achievement, reference, nil
}

// SubmitForVerification submit achievement untuk verifikasi
func (s *AchievementBusinessLogic) SubmitForVerification(studentID, achievementID string) (*model.AchievementReference, error) {
	// Cari reference berdasarkan achievement ID dan student ID
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

	// Cek status harus draft
	if targetRef.Status != "draft" {
		return nil, errors.New("only draft achievements can be submitted")
	}

	// Update status
	targetRef.Status = "submitted"
	targetRef.SubmittedAt = time.Now()

	err = s.achievementRepo.UpdateReference(targetRef)
	if err != nil {
		return nil, err
	}

	return targetRef, nil
}

// VerifyAchievement verify atau reject achievement
func (s *AchievementBusinessLogic) VerifyAchievement(lecturerID string, referenceID primitive.ObjectID, status, rejectionNote string) (*model.AchievementReference, error) {
	// Validasi status
	if status != "verified" && status != "rejected" {
		return nil, errors.New("status must be 'verified' or 'rejected'")
	}

	// Jika reject, harus ada rejection note
	if status == "rejected" && rejectionNote == "" {
		return nil, errors.New("rejection note is required when rejecting")
	}

	// Cek apakah lecturer ada
	_, err := s.lecturerRepo.GetByUserID(lecturerID)
	if err != nil {
		return nil, errors.New("lecturer not found")
	}

	// Ambil reference
	reference, err := s.achievementRepo.GetReferenceByID(referenceID)
	if err != nil {
		return nil, err
	}

	// Cek status harus submitted
	if reference.Status != "submitted" {
		return nil, errors.New("only submitted achievements can be verified")
	}

	// Update reference
	reference.Status = status
	reference.VerifiedBy = lecturerID
	reference.VerifiedAt = time.Now()
	if status == "rejected" {
		reference.RejectionNote = rejectionNote
	}

	err = s.achievementRepo.UpdateReference(reference)
	if err != nil {
		return nil, err
	}

	// Send notification
	s.notificationSvc.SendNotification(reference.StudentID, "Achievement "+status)

	return reference, nil
}

// SoftDeleteAchievement soft delete achievement
func (s *AchievementBusinessLogic) SoftDeleteAchievement(studentID string, achievementID primitive.ObjectID) error {
	// Ambil achievement
	achievement, err := s.achievementRepo.GetByID(achievementID)
	if err != nil {
		return err
	}

	// Cek ownership
	if achievement.StudentID != studentID {
		return errors.New("unauthorized")
	}

	// Ambil reference
	reference, err := s.achievementRepo.GetReferenceByAchievementID(achievementID.Hex())
	if err != nil {
		return err
	}

	// Cek status harus draft
	if reference.Status != "draft" {
		return errors.New("only draft achievements can be deleted")
	}

	// Soft delete achievement
	now := time.Now()
	achievement.DeletedAt = &now

	err = s.achievementRepo.Update(achievement)
	if err != nil {
		return err
	}

	// Update reference status
	reference.Status = "deleted"
	err = s.achievementRepo.UpdateReference(reference)
	if err != nil {
		return err
	}

	return nil
}

// GetStudentAchievements ambil semua achievement student
func (s *AchievementBusinessLogic) GetStudentAchievements(studentID string) ([]model.Achievement, error) {
	return s.achievementRepo.GetByStudentID(studentID)
}

// ValidateStatusTransition validasi transisi status
func ValidateStatusTransition(from, to string) bool {
	validTransitions := map[string][]string{
		"draft":     {"submitted", "deleted"},
		"submitted": {"verified", "rejected"},
		"verified":  {},
		"rejected":  {},
		"deleted":   {},
	}

	allowedTransitions, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, allowed := range allowedTransitions {
		if allowed == to {
			return true
		}
	}

	return false
}

// ==================== TEST CASES ====================

// TestCreateAchievement menguji pembuatan achievement
func TestCreateAchievement(t *testing.T) {
	// SETUP
	mockAchievementRepo := NewMockAchievementRepository()
	mockStudentRepo := NewMockStudentRepository()
	mockLecturerRepo := NewMockLecturerRepository()
	mockNotificationSvc := NewMockNotificationService()

	service := NewAchievementBusinessLogic(mockAchievementRepo, mockStudentRepo, mockLecturerRepo, mockNotificationSvc)

	// Setup test data
	student := &model.Student{
		ID:           "student-internal-123",
		UserID:       "student-123",
		StudentID:    "12345678",
		ProgramStudy: "D4 Teknik Informatika",
		AcademicYear: "2023",
	}
	mockStudentRepo.AddStudent(student)

	tests := []struct {
		name        string
		studentID   string
		category    string
		title       string
		description string
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "Valid Achievement",
			studentID:   "student-123",
			category:    "competition",
			title:       "Programming Contest Winner",
			description: "Won first place in national programming contest",
			wantErr:     false,
			errMsg:      "",
		},
		{
			name:        "Empty Student ID",
			studentID:   "",
			category:    "competition",
			title:       "Test Achievement",
			description: "Test description",
			wantErr:     true,
			errMsg:      "student ID cannot be empty",
		},
		{
			name:        "Empty Category",
			studentID:   "student-123",
			category:    "",
			title:       "Test Achievement",
			description: "Test description",
			wantErr:     true,
			errMsg:      "category cannot be empty",
		},
		{
			name:        "Empty Title",
			studentID:   "student-123",
			category:    "competition",
			title:       "",
			description: "Test description",
			wantErr:     true,
			errMsg:      "title cannot be empty",
		},
		{
			name:        "Empty Description",
			studentID:   "student-123",
			category:    "competition",
			title:       "Test Achievement",
			description: "",
			wantErr:     true,
			errMsg:      "description cannot be empty",
		},
		{
			name:        "Invalid Category",
			studentID:   "student-123",
			category:    "invalid_category",
			title:       "Test Achievement",
			description: "Test description",
			wantErr:     true,
			errMsg:      "invalid category",
		},
		{
			name:        "Student Not Found",
			studentID:   "nonexistent-student",
			category:    "competition",
			title:       "Test Achievement",
			description: "Test description",
			wantErr:     true,
			errMsg:      "student not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			achievement, reference, err := service.CreateAchievement(tt.studentID, tt.category, tt.title, tt.description)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateAchievement() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("error = %v, want %v", err.Error(), tt.errMsg)
				return
			}

			if !tt.wantErr {
				if achievement == nil {
					t.Errorf("got nil achievement, want achievement")
					return
				}
				if reference == nil {
					t.Errorf("got nil reference, want reference")
					return
				}

				// Verify achievement properties
				if achievement.StudentID != tt.studentID {
					t.Errorf("StudentID = %v, want %v", achievement.StudentID, tt.studentID)
				}
				if achievement.Category != tt.category {
					t.Errorf("Category = %v, want %v", achievement.Category, tt.category)
				}
				if achievement.Title != tt.title {
					t.Errorf("Title = %v, want %v", achievement.Title, tt.title)
				}

				// Verify reference properties
				if reference.StudentID != tt.studentID {
					t.Errorf("Reference StudentID = %v, want %v", reference.StudentID, tt.studentID)
				}
				if reference.Status != "draft" {
					t.Errorf("Status = %v, want %v", reference.Status, "draft")
				}
				if reference.AchievementID != achievement.ID.Hex() {
					t.Errorf("AchievementID = %v, want %v", reference.AchievementID, achievement.ID.Hex())
				}
			}
		})
	}
}

// TestSubmitForVerification menguji submit achievement untuk verifikasi
func TestSubmitForVerification(t *testing.T) {
	// SETUP
	mockAchievementRepo := NewMockAchievementRepository()
	mockStudentRepo := NewMockStudentRepository()
	mockLecturerRepo := NewMockLecturerRepository()
	mockNotificationSvc := NewMockNotificationService()

	service := NewAchievementBusinessLogic(mockAchievementRepo, mockStudentRepo, mockLecturerRepo, mockNotificationSvc)

	// Setup test data
	student := &model.Student{
		ID:           "student-internal-123",
		UserID:       "student-123",
		StudentID:    "12345678",
		ProgramStudy: "D4 Teknik Informatika",
	}
	mockStudentRepo.AddStudent(student)

	// Create achievement first
	achievement, reference, err := service.CreateAchievement("student-123", "competition", "Test Achievement", "Test description")
	if err != nil {
		t.Fatalf("Failed to create achievement: %v", err)
	}

	achievementID := achievement.ID.Hex()

	tests := []struct {
		name          string
		studentID     string
		achievementID string
		wantErr       bool
		expectedStatus string
	}{
		{
			name:          "Valid Submission",
			studentID:     "student-123",
			achievementID: achievementID,
			wantErr:       false,
			expectedStatus: "submitted",
		},
		{
			name:          "Invalid Achievement ID",
			studentID:     "student-123",
			achievementID: "nonexistent-id",
			wantErr:       true,
		},
		{
			name:          "Wrong Student",
			studentID:     "wrong-student",
			achievementID: achievementID,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset reference status to draft for each test
			if tt.name == "Valid Submission" {
				reference.Status = "draft"
				mockAchievementRepo.UpdateReference(reference)
			}

			updatedRef, err := service.SubmitForVerification(tt.studentID, tt.achievementID)

			if (err != nil) != tt.wantErr {
				t.Errorf("SubmitForVerification() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if updatedRef == nil {
					t.Errorf("Expected updated reference but got nil")
					return
				}

				if updatedRef.Status != tt.expectedStatus {
					t.Errorf("Status = %v, want %v", updatedRef.Status, tt.expectedStatus)
				}

				if updatedRef.SubmittedAt.IsZero() {
					t.Errorf("SubmittedAt should be set")
				}
			}
		})
	}
}

// TestVerifyAchievement menguji verifikasi achievement
func TestVerifyAchievement(t *testing.T) {
	// SETUP
	mockAchievementRepo := NewMockAchievementRepository()
	mockStudentRepo := NewMockStudentRepository()
	mockLecturerRepo := NewMockLecturerRepository()
	mockNotificationSvc := NewMockNotificationService()

	service := NewAchievementBusinessLogic(mockAchievementRepo, mockStudentRepo, mockLecturerRepo, mockNotificationSvc)

	// Setup test data
	student := &model.Student{
		ID:           "student-internal-123",
		UserID:       "student-123",
		StudentID:    "12345678",
		ProgramStudy: "D4 Teknik Informatika",
	}
	mockStudentRepo.AddStudent(student)

	lecturer := &model.Lecturer{
		ID:           "lecturer-internal-123",
		UserID:       "lecturer-123",
		LecturerID:   "19800101",
		Department:   "Teknik Informatika",
	}
	mockLecturerRepo.AddLecturer(lecturer)

	// Create and submit achievement
	achievement, reference, err := service.CreateAchievement("student-123", "competition", "Test Achievement", "Test description")
	if err != nil {
		t.Fatalf("Failed to create achievement: %v", err)
	}

	_, err = service.SubmitForVerification("student-123", achievement.ID.Hex())
	if err != nil {
		t.Fatalf("Failed to submit achievement: %v", err)
	}

	tests := []struct {
		name          string
		lecturerID    string
		referenceID   primitive.ObjectID
		status        string
		rejectionNote string
		wantErr       bool
	}{
		{
			name:          "Valid Verification",
			lecturerID:    "lecturer-123",
			referenceID:   reference.ID,
			status:        "verified",
			rejectionNote: "",
			wantErr:       false,
		},
		{
			name:          "Valid Rejection with Note",
			lecturerID:    "lecturer-123",
			referenceID:   reference.ID,
			status:        "rejected",
			rejectionNote: "Documents incomplete",
			wantErr:       false,
		},
		{
			name:          "Invalid Status",
			lecturerID:    "lecturer-123",
			referenceID:   reference.ID,
			status:        "invalid",
			rejectionNote: "",
			wantErr:       true,
		},
		{
			name:          "Rejection without Note",
			lecturerID:    "lecturer-123",
			referenceID:   reference.ID,
			status:        "rejected",
			rejectionNote: "",
			wantErr:       true,
		},
		{
			name:          "Lecturer Not Found",
			lecturerID:    "nonexistent-lecturer",
			referenceID:   reference.ID,
			status:        "verified",
			rejectionNote: "",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset reference status to submitted for each test
			reference.Status = "submitted"
			mockAchievementRepo.UpdateReference(reference)

			updatedRef, err := service.VerifyAchievement(tt.lecturerID, tt.referenceID, tt.status, tt.rejectionNote)

			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyAchievement() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if updatedRef == nil {
					t.Errorf("Expected updated reference but got nil")
					return
				}

				if updatedRef.Status != tt.status {
					t.Errorf("Status = %v, want %v", updatedRef.Status, tt.status)
				}

				if updatedRef.VerifiedBy != tt.lecturerID {
					t.Errorf("VerifiedBy = %v, want %v", updatedRef.VerifiedBy, tt.lecturerID)
				}

				if updatedRef.VerifiedAt.IsZero() {
					t.Errorf("VerifiedAt should be set")
				}

				if tt.status == "rejected" && updatedRef.RejectionNote != tt.rejectionNote {
					t.Errorf("RejectionNote = %v, want %v", updatedRef.RejectionNote, tt.rejectionNote)
				}
			}
		})
	}
}

// TestSoftDeleteAchievement menguji soft delete achievement
func TestSoftDeleteAchievement(t *testing.T) {
	// SETUP
	mockAchievementRepo := NewMockAchievementRepository()
	mockStudentRepo := NewMockStudentRepository()
	mockLecturerRepo := NewMockLecturerRepository()
	mockNotificationSvc := NewMockNotificationService()

	service := NewAchievementBusinessLogic(mockAchievementRepo, mockStudentRepo, mockLecturerRepo, mockNotificationSvc)

	// Setup test data
	student := &model.Student{
		ID:           "student-internal-123",
		UserID:       "student-123",
		StudentID:    "12345678",
		ProgramStudy: "D4 Teknik Informatika",
	}
	mockStudentRepo.AddStudent(student)

	// Create achievement
	achievement, reference, err := service.CreateAchievement("student-123", "competition", "Test Achievement", "Test description")
	if err != nil {
		t.Fatalf("Failed to create achievement: %v", err)
	}

	achievementID := achievement.ID

	tests := []struct {
		name          string
		studentID     string
		achievementID primitive.ObjectID
		status        string
		wantErr       bool
	}{
		{
			name:          "Valid Delete Draft",
			studentID:     "student-123",
			achievementID: achievementID,
			status:        "draft",
			wantErr:       false,
		},
		{
			name:          "Cannot Delete Submitted",
			studentID:     "student-123",
			achievementID: achievementID,
			status:        "submitted",
			wantErr:       true,
		},
		{
			name:          "Wrong Student",
			studentID:     "wrong-student",
			achievementID: achievementID,
			status:        "draft",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set reference status
			reference.Status = tt.status
			mockAchievementRepo.UpdateReference(reference)

			err := service.SoftDeleteAchievement(tt.studentID, tt.achievementID)

			if (err != nil) != tt.wantErr {
				t.Errorf("SoftDeleteAchievement() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify achievement is soft deleted
				updatedAchievement, _ := mockAchievementRepo.GetByID(tt.achievementID)
				if updatedAchievement.DeletedAt == nil {
					t.Errorf("Achievement should be soft deleted")
				}

				// Verify reference status is updated
				updatedRef, _ := mockAchievementRepo.GetReferenceByAchievementID(tt.achievementID.Hex())
				if updatedRef.Status != "deleted" {
					t.Errorf("Reference status = %v, want %v", updatedRef.Status, "deleted")
				}
			}
		})
	}
}

// TestValidateStatusTransition menguji validasi transisi status
func TestValidateStatusTransition(t *testing.T) {
	tests := []struct {
		name       string
		fromStatus string
		toStatus   string
		wantValid  bool
	}{
		{"Draft to Submitted", "draft", "submitted", true},
		{"Submitted to Verified", "submitted", "verified", true},
		{"Submitted to Rejected", "submitted", "rejected", true},
		{"Draft to Verified", "draft", "verified", false}, // Invalid transition
		{"Verified to Draft", "verified", "draft", false}, // Invalid transition
		{"Rejected to Submitted", "rejected", "submitted", false}, // Invalid transition
		{"Draft to Deleted", "draft", "deleted", true},
		{"Submitted to Deleted", "submitted", "deleted", false}, // Cannot delete submitted
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := ValidateStatusTransition(tt.fromStatus, tt.toStatus)

			if isValid != tt.wantValid {
				t.Errorf("ValidateStatusTransition(%v, %v) = %v, want %v",
					tt.fromStatus, tt.toStatus, isValid, tt.wantValid)
			}
		})
	}
}

// TestGetStudentAchievements menguji pengambilan achievement mahasiswa
func TestGetStudentAchievements(t *testing.T) {
	// SETUP
	mockAchievementRepo := NewMockAchievementRepository()
	mockStudentRepo := NewMockStudentRepository()
	mockLecturerRepo := NewMockLecturerRepository()
	mockNotificationSvc := NewMockNotificationService()

	service := NewAchievementBusinessLogic(mockAchievementRepo, mockStudentRepo, mockLecturerRepo, mockNotificationSvc)

	// Setup test data
	student := &model.Student{
		ID:           "student-internal-123",
		UserID:       "student-123",
		StudentID:    "12345678",
		ProgramStudy: "D4 Teknik Informatika",
	}
	mockStudentRepo.AddStudent(student)

	// Create multiple achievements
	achievements := []struct {
		title    string
		category string
	}{
		{"Achievement 1", "competition"},
		{"Achievement 2", "research"},
		{"Achievement 3", "academic"},
	}

	for _, ach := range achievements {
		service.CreateAchievement("student-123", ach.category, ach.title, "Test description")
	}

	tests := []struct {
		name          string
		studentID     string
		expectedCount int
		wantErr       bool
	}{
		{
			name:          "Valid Student",
			studentID:     "student-123",
			expectedCount: 3,
			wantErr:       false,
		},
		{
			name:          "Nonexistent Student",
			studentID:     "nonexistent-student",
			expectedCount: 0,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			achievements, err := service.GetStudentAchievements(tt.studentID)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetStudentAchievements() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(achievements) != tt.expectedCount {
				t.Errorf("Achievement count = %v, want %v", len(achievements), tt.expectedCount)
			}
		})
	}
}