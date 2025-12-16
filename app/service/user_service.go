package service

import (
	"UASBE/app/model"
	"UASBE/app/repository"
	"errors"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	userRepo     *repository.UserRepository
	studentRepo  *repository.StudentRepository
	lecturerRepo *repository.LecturerRepository
}

type CreateUserRequest struct {
	Username     string `json:"username"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	FullName     string `json:"full_name"`
	Role         string `json:"role"` // "admin", "student", "lecturer"
	IsActive     bool   `json:"is_active"`
	
	// Student specific fields
	StudentID    string `json:"student_id,omitempty"`
	ProgramStudy string `json:"program_study,omitempty"`
	AcademicYear string `json:"academic_year,omitempty"`
	AdvisorID    string `json:"advisor_id,omitempty"`
	
	// Lecturer specific fields
	LecturerID   string `json:"lecturer_id,omitempty"`
	Department   string `json:"department,omitempty"`
}

type UpdateUserRequest struct {
	Username     string `json:"username,omitempty"`
	Email        string `json:"email,omitempty"`
	Password     string `json:"password,omitempty"`
	FullName     string `json:"full_name,omitempty"`
	Role         string `json:"role,omitempty"`
	IsActive     *bool  `json:"is_active,omitempty"`
	
	// Student specific fields
	StudentID    string `json:"student_id,omitempty"`
	ProgramStudy string `json:"program_study,omitempty"`
	AcademicYear string `json:"academic_year,omitempty"`
	AdvisorID    string `json:"advisor_id,omitempty"`
	
	// Lecturer specific fields
	LecturerID   string `json:"lecturer_id,omitempty"`
	Department   string `json:"department,omitempty"`
}

func NewUserService(userRepo *repository.UserRepository, studentRepo *repository.StudentRepository, lecturerRepo *repository.LecturerRepository) *UserService {
	return &UserService{
		userRepo:     userRepo,
		studentRepo:  studentRepo,
		lecturerRepo: lecturerRepo,
	}
}

// FR-009 Step 1: Create user
func (s *UserService) CreateUserRequest(c *fiber.Ctx) error {
	userRole := c.Locals("role").(string)
	
	// Only admin can create users
	if userRole != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only admin can create users",
		})
	}

	var req CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Username == "" || req.Email == "" || req.Password == "" || req.FullName == "" || req.Role == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Username, email, password, full_name, and role are required",
		})
	}

	user, err := s.CreateUser(&req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    user,
	})
}

// FR-009 Step 1: Update user
func (s *UserService) UpdateUserRequest(c *fiber.Ctx) error {
	userRole := c.Locals("role").(string)
	
	// Only admin can update users
	if userRole != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only admin can update users",
		})
	}

	userID := c.Params("id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "User ID is required",
		})
	}

	var req UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	user, err := s.UpdateUser(userID, &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    user,
	})
}

// FR-009 Step 1: Delete user
func (s *UserService) DeleteUserRequest(c *fiber.Ctx) error {
	userRole := c.Locals("role").(string)
	
	// Only admin can delete users
	if userRole != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only admin can delete users",
		})
	}

	userID := c.Params("id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "User ID is required",
		})
	}

	err := s.DeleteUser(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "User deleted successfully",
	})
}

// Get all users
func (s *UserService) GetAllUsersRequest(c *fiber.Ctx) error {
	userRole := c.Locals("role").(string)
	
	// Only admin can view all users
	if userRole != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only admin can view all users",
		})
	}

	users, err := s.userRepo.GetAll()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get users",
		})
	}

	// Remove passwords from response
	for i := range users {
		users[i].Password = ""
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    users,
	})
}

// Get user by ID
func (s *UserService) GetUserByIDRequest(c *fiber.Ctx) error {
	userRole := c.Locals("role").(string)
	
	// Only admin can view user details
	if userRole != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Only admin can view user details",
		})
	}

	userID := c.Params("id")
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Remove password from response
	user.Password = ""

	return c.JSON(fiber.Map{
		"success": true,
		"data":    user,
	})
}

// Business logic methods
func (s *UserService) CreateUser(req *CreateUserRequest) (*model.User, error) {
	// Check if username already exists
	_, err := s.userRepo.GetByUsername(req.Username)
	if err == nil {
		return nil, errors.New("username already exists")
	}

	// Check if email already exists
	_, err = s.userRepo.GetByEmail(req.Email)
	if err == nil {
		return nil, errors.New("email already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Get role ID
	roleID, err := s.getRoleIDByName(req.Role)
	if err != nil {
		return nil, err
	}

	// Create user
	user := &model.User{
		Username:  req.Username,
		Email:     req.Email,
		Password:  string(hashedPassword),
		FullName:  req.FullName,
		RoleID:    roleID,
		IsActive:  req.IsActive,
	}

	err = s.userRepo.Create(user)
	if err != nil {
		return nil, err
	}

	// FR-009 Step 3: Set student/lecturer profile
	err = s.createRoleProfile(user, req)
	if err != nil {
		// Rollback user creation
		s.userRepo.Delete(user.ID)
		return nil, err
	}

	// Remove password from response
	user.Password = ""
	return user, nil
}

func (s *UserService) UpdateUser(userID string, req *UpdateUserRequest) (*model.User, error) {
	// Get existing user
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Update fields if provided
	if req.Username != "" {
		user.Username = req.Username
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		user.Password = string(hashedPassword)
	}
	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	// FR-009 Step 2: Assign role if changed
	if req.Role != "" {
		roleID, err := s.getRoleIDByName(req.Role)
		if err != nil {
			return nil, err
		}
		user.RoleID = roleID
	}

	err = s.userRepo.Update(user)
	if err != nil {
		return nil, err
	}

	// Update role profile if needed
	if req.Role != "" {
		err = s.updateRoleProfile(user, req)
		if err != nil {
			return nil, err
		}
	}

	// Remove password from response
	user.Password = ""
	return user, nil
}

func (s *UserService) DeleteUser(userID string) error {
	return s.userRepo.Delete(userID)
}

func (s *UserService) createRoleProfile(user *model.User, req *CreateUserRequest) error {
	switch req.Role {
	case "student":
		if req.StudentID == "" || req.ProgramStudy == "" || req.AcademicYear == "" {
			return errors.New("student_id, program_study, and academic_year are required for students")
		}
		
		student := &model.Student{
			UserID:       user.ID,
			StudentID:    req.StudentID,
			ProgramStudy: req.ProgramStudy,
			AcademicYear: req.AcademicYear,
		}
		
		// FR-009 Step 4: Set advisor untuk mahasiswa
		if req.AdvisorID != "" {
			student.AdvisorID = req.AdvisorID
		}
		
		return s.studentRepo.Create(student)
		
	case "lecturer":
		if req.LecturerID == "" || req.Department == "" {
			return errors.New("lecturer_id and department are required for lecturers")
		}
		
		lecturer := &model.Lecturer{
			UserID:     user.ID,
			LecturerID: req.LecturerID,
			Department: req.Department,
		}
		
		return s.lecturerRepo.Create(lecturer)
	}
	
	return nil
}

func (s *UserService) updateRoleProfile(user *model.User, req *UpdateUserRequest) error {
	// This is a simplified implementation
	// In a real app, you'd handle role changes more carefully
	return nil
}

func (s *UserService) getRoleIDByName(roleName string) (string, error) {
	db := s.userRepo.GetDB()
	if db == nil {
		return "", errors.New("database connection not available")
	}
	
	var roleID string
	query := "SELECT id FROM roles WHERE name = $1"
	err := db.QueryRow(query, roleName).Scan(&roleID)
	if err != nil {
		return "", errors.New("role not found")
	}
	
	return roleID, nil
}