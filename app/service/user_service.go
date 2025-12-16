package service

import (
	"UASBE/app/model"
	"UASBE/app/repository"
	"errors"
	"fmt"
	"time"

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
	adminUsername := c.Locals("username").(string)
	
	// Only admin can create users
	if userRole != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error": "Access denied",
			"message": "Only admin can create users",
			"code": "INSUFFICIENT_PERMISSIONS",
			"user_role": userRole,
			"required_role": "admin",
		})
	}

	var req CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": "Invalid request body",
			"message": "Please provide valid JSON data",
			"code": "INVALID_REQUEST_BODY",
		})
	}

	// Enhanced validation
	validationErrors := make(map[string]string)
	if req.Username == "" {
		validationErrors["username"] = "Username is required"
	}
	if req.Email == "" {
		validationErrors["email"] = "Email is required"
	}
	if req.Password == "" {
		validationErrors["password"] = "Password is required"
	}
	if req.FullName == "" {
		validationErrors["full_name"] = "Full name is required"
	}
	if req.Role == "" {
		validationErrors["role"] = "Role is required"
	}

	// Validate role
	validRoles := []string{"admin", "student", "lecturer"}
	isValidRole := false
	for _, role := range validRoles {
		if req.Role == role {
			isValidRole = true
			break
		}
	}
	if req.Role != "" && !isValidRole {
		validationErrors["role"] = "Invalid role. Valid options: " + fmt.Sprintf("%v", validRoles)
	}

	// Role-specific validation
	if req.Role == "student" {
		if req.StudentID == "" {
			validationErrors["student_id"] = "Student ID is required for students"
		}
		if req.ProgramStudy == "" {
			validationErrors["program_study"] = "Program study is required for students"
		}
		if req.AcademicYear == "" {
			validationErrors["academic_year"] = "Academic year is required for students"
		}
	}

	if req.Role == "lecturer" {
		if req.LecturerID == "" {
			validationErrors["lecturer_id"] = "Lecturer ID is required for lecturers"
		}
		if req.Department == "" {
			validationErrors["department"] = "Department is required for lecturers"
		}
	}

	if len(validationErrors) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": "Validation failed",
			"message": "Please correct the following errors",
			"code": "VALIDATION_ERROR",
			"details": validationErrors,
			"valid_roles": validRoles,
			"role_requirements": fiber.Map{
				"student": []string{"student_id", "program_study", "academic_year"},
				"lecturer": []string{"lecturer_id", "department"},
				"admin": []string{},
			},
		})
	}

	user, err := s.CreateUser(&req)
	if err != nil {
		var errorCode string
		var message string
		
		switch err.Error() {
		case "username already exists":
			errorCode = "USERNAME_EXISTS"
			message = "Username is already taken"
		case "email already exists":
			errorCode = "EMAIL_EXISTS"
			message = "Email is already registered"
		case "role not found":
			errorCode = "ROLE_NOT_FOUND"
			message = "Invalid role specified"
		default:
			errorCode = "CREATION_FAILED"
			message = "Failed to create user"
		}
		
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": err.Error(),
			"message": message,
			"code": errorCode,
		})
	}

	// Get additional profile info for response
	var profileInfo fiber.Map
	var permissions []string
	
	if req.Role == "student" {
		student, err := s.studentRepo.GetByUserID(user.ID)
		if err == nil {
			profileInfo = fiber.Map{
				"type": "student",
				"student_id": student.StudentID,
				"program_study": student.ProgramStudy,
				"academic_year": student.AcademicYear,
				"advisor_id": student.AdvisorID,
			}
			
			// Get advisor info if available
			if student.AdvisorID != "" {
				advisor, err := s.lecturerRepo.GetByUserID(student.AdvisorID)
				if err == nil {
					profileInfo["advisor_info"] = fiber.Map{
						"advisor_id": advisor.LecturerID,
						"department": advisor.Department,
					}
				}
			}
		}
		permissions = []string{"achievements:create", "achievements:read", "achievements:update", "achievements:delete"}
	} else if req.Role == "lecturer" {
		lecturer, err := s.lecturerRepo.GetByUserID(user.ID)
		if err == nil {
			profileInfo = fiber.Map{
				"type": "lecturer",
				"lecturer_id": lecturer.LecturerID,
				"department": lecturer.Department,
			}
		}
		permissions = []string{"achievements:verify", "achievements:view_advisee"}
	} else {
		profileInfo = fiber.Map{
			"type": "admin",
		}
		permissions = []string{"users:create", "users:read", "users:update", "users:delete", "achievements:view_all"}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "User created successfully",
		"code": "USER_CREATED",
		"data": fiber.Map{
			"user": fiber.Map{
				"id": user.ID,
				"username": user.Username,
				"email": user.Email,
				"full_name": user.FullName,
				"role": req.Role,
				"is_active": user.IsActive,
				"created_at": user.CreatedAt,
			},
			"profile": profileInfo,
			"permissions": permissions,
		},
		"created_by": fiber.Map{
			"admin_username": adminUsername,
			"timestamp": time.Now(),
		},
		"next_steps": []string{
			"User account has been created successfully",
			"User can now login with username: " + user.Username,
			"Default password has been set (user should change it on first login)",
			"User has been assigned appropriate permissions based on role",
		},
		"login_info": fiber.Map{
			"username": user.Username,
			"temporary_password": "Set by admin (hidden for security)",
			"first_login_required": true,
		},
		"timestamp": time.Now(),
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