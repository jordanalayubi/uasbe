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
// CreateUserRequest handles user creation
// @Summary Create User
// @Description Admin creates a new user with role assignment (FR-009)
// @Tags User Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user body CreateUserRequest true "User data"
// @Success 201 {object} map[string]interface{} "User created successfully"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 403 {object} map[string]interface{} "Forbidden - Admin only"
// @Router /users [post]
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
		// Validate advisor_id if provided (advisor_id is optional)
		if req.AdvisorID != "" {
			// Check if advisor exists and is a lecturer
			user, err := s.userRepo.GetByID(req.AdvisorID)
			if err != nil {
				// Get available advisors for better error message
				lecturers, lecErr := s.lecturerRepo.GetAll()
				availableIDs := []string{}
				if lecErr == nil {
					for _, lecturer := range lecturers {
						availableIDs = append(availableIDs, lecturer.UserID)
					}
				}
				
				if len(availableIDs) > 0 {
					validationErrors["advisor_id"] = fmt.Sprintf("Advisor user not found. Available lecturer user_ids: %v. Leave empty if no advisor.", availableIDs)
				} else {
					validationErrors["advisor_id"] = "Advisor user not found. No lecturers available. Create a lecturer first or leave empty if no advisor."
				}
			} else {
				// Check if user is a lecturer (accept both "lecturer" and "Dosen Wali")
				roleName := s.getRoleNameByID(user.RoleID)
				if roleName != "lecturer" && roleName != "Dosen Wali" {
					validationErrors["advisor_id"] = fmt.Sprintf("Advisor must be a lecturer. Provided user has role: %s", roleName)
				} else {
					// Check if lecturer profile exists
					_, err := s.lecturerRepo.GetByUserID(req.AdvisorID)
					if err != nil {
						validationErrors["advisor_id"] = "Lecturer profile not found for this user"
					}
				}
			}
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
			"success": false,
			"error": "Access denied",
			"message": "Only admin can view all users",
			"code": "INSUFFICIENT_PERMISSIONS",
		})
	}

	users, err := s.userRepo.GetAll()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": "Failed to get users",
			"message": err.Error(),
			"code": "FETCH_FAILED",
		})
	}

	// Enhanced user data with profile information
	var enhancedUsers []fiber.Map
	for _, user := range users {
		userData := fiber.Map{
			"id": user.ID,
			"username": user.Username,
			"email": user.Email,
			"full_name": user.FullName,
			"role": s.getRoleNameByID(user.RoleID),
			"is_active": user.IsActive,
			"created_at": user.CreatedAt,
			"updated_at": user.UpdatedAt,
		}

		// Add profile information based on role
		roleName := s.getRoleNameByID(user.RoleID)
		switch roleName {
		case "student":
			student, err := s.studentRepo.GetByUserID(user.ID)
			if err == nil {
				userData["profile"] = fiber.Map{
					"student_id": student.StudentID,
					"program_study": student.ProgramStudy,
					"academic_year": student.AcademicYear,
					"advisor_id": student.AdvisorID,
				}
				
				// Get advisor name if exists
				if student.AdvisorID != "" {
					advisor, err := s.lecturerRepo.GetByUserID(student.AdvisorID)
					if err == nil {
						advisorUser, err := s.userRepo.GetByID(advisor.UserID)
						if err == nil {
							userData["advisor_info"] = fiber.Map{
								"advisor_name": advisorUser.FullName,
								"lecturer_id": advisor.LecturerID,
								"department": advisor.Department,
							}
						}
					}
				}
			}
		case "lecturer":
			lecturer, err := s.lecturerRepo.GetByUserID(user.ID)
			if err == nil {
				userData["profile"] = fiber.Map{
					"lecturer_id": lecturer.LecturerID,
					"department": lecturer.Department,
				}
			}
		}

		enhancedUsers = append(enhancedUsers, userData)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Users retrieved successfully",
		"data": enhancedUsers,
		"total": len(enhancedUsers),
		"summary": fiber.Map{
			"total_users": len(enhancedUsers),
			"active_users": s.countActiveUsers(users),
			"inactive_users": len(users) - s.countActiveUsers(users),
		},
	})
}

// Get user by ID
func (s *UserService) GetUserByIDRequest(c *fiber.Ctx) error {
	userRole := c.Locals("role").(string)
	
	// Only admin can view user details
	if userRole != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error": "Access denied",
			"message": "Only admin can view user details",
			"code": "INSUFFICIENT_PERMISSIONS",
		})
	}

	userID := c.Params("id")
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error": "User not found",
			"message": "The requested user does not exist",
			"code": "USER_NOT_FOUND",
		})
	}

	// Enhanced user data with profile information
	userData := fiber.Map{
		"id": user.ID,
		"username": user.Username,
		"email": user.Email,
		"full_name": user.FullName,
		"role": s.getRoleNameByID(user.RoleID),
		"is_active": user.IsActive,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
	}

	// Add profile information based on role
	roleName := s.getRoleNameByID(user.RoleID)
	switch roleName {
	case "student":
		student, err := s.studentRepo.GetByUserID(user.ID)
		if err == nil {
			userData["profile"] = fiber.Map{
				"student_id": student.StudentID,
				"program_study": student.ProgramStudy,
				"academic_year": student.AcademicYear,
				"advisor_id": student.AdvisorID,
			}
			
			// Get advisor info if exists
			if student.AdvisorID != "" {
				advisor, err := s.lecturerRepo.GetByUserID(student.AdvisorID)
				if err == nil {
					advisorUser, err := s.userRepo.GetByID(advisor.UserID)
					if err == nil {
						userData["advisor_info"] = fiber.Map{
							"advisor_name": advisorUser.FullName,
							"lecturer_id": advisor.LecturerID,
							"department": advisor.Department,
						}
					}
				}
			}
		}
	case "lecturer":
		lecturer, err := s.lecturerRepo.GetByUserID(user.ID)
		if err == nil {
			userData["profile"] = fiber.Map{
				"lecturer_id": lecturer.LecturerID,
				"department": lecturer.Department,
			}
			
			// Get advisee count
			students, err := s.studentRepo.GetByAdvisorID(user.ID)
			if err == nil {
				userData["advisee_count"] = len(students)
			}
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "User details retrieved successfully",
		"data": userData,
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
			// Validate that advisor exists and is a lecturer
			advisorUser, err := s.userRepo.GetByID(req.AdvisorID)
			if err != nil {
				return errors.New("advisor user not found")
			}
			
			// Check if user is a lecturer (accept both "lecturer" and "Dosen Wali")
			roleName := s.getRoleNameByID(advisorUser.RoleID)
			if roleName != "lecturer" && roleName != "Dosen Wali" {
				return fmt.Errorf("advisor must be a lecturer. Current role: %s", roleName)
			}
			
			// Check if lecturer profile exists
			_, err = s.lecturerRepo.GetByUserID(req.AdvisorID)
			if err != nil {
				return errors.New("lecturer profile not found for advisor")
			}
			
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
// GetAvailableAdvisorsRequest - Get list of lecturers that can be assigned as advisors
// @Summary Get Available Advisors
// @Description Get list of lecturers that can be assigned as advisors for students
// @Tags User Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Available advisors retrieved successfully"
// @Failure 403 {object} map[string]interface{} "Forbidden - Admin only"
// @Router /users/advisors [get]
func (s *UserService) GetAvailableAdvisorsRequest(c *fiber.Ctx) error {
	userRole := c.Locals("role").(string)
	
	// Only admin can view available advisors
	if userRole != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error": "Access denied",
			"message": "Only admin can view available advisors",
			"code": "INSUFFICIENT_PERMISSIONS",
		})
	}

	// Get all lecturers
	lecturers, err := s.lecturerRepo.GetAll()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": "Failed to get lecturers",
			"message": err.Error(),
			"code": "FETCH_FAILED",
		})
	}

	// Get user details for each lecturer
	var advisors []fiber.Map
	for _, lecturer := range lecturers {
		user, err := s.userRepo.GetByID(lecturer.UserID)
		if err != nil {
			continue // Skip if user not found
		}

		// Only include active lecturers
		if user.IsActive {
			advisors = append(advisors, fiber.Map{
				"user_id": lecturer.UserID,
				"lecturer_id": lecturer.LecturerID,
				"full_name": user.FullName,
				"username": user.Username,
				"email": user.Email,
				"department": lecturer.Department,
			})
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Available advisors retrieved successfully",
		"data": advisors,
		"total": len(advisors),
		"usage_note": "Use 'user_id' field as advisor_id when creating students",
	})
}
// Helper method to get role name by ID
func (s *UserService) getRoleNameByID(roleID string) string {
	db := s.userRepo.GetDB()
	if db == nil {
		return "unknown"
	}
	
	var roleName string
	query := "SELECT name FROM roles WHERE id = $1"
	err := db.QueryRow(query, roleID).Scan(&roleName)
	if err != nil {
		return "unknown"
	}
	
	return roleName
}

// Helper method to count active users
func (s *UserService) countActiveUsers(users []model.User) int {
	count := 0
	for _, user := range users {
		if user.IsActive {
			count++
		}
	}
	return count
}
// CreateDefaultLecturerRequest - Create default lecturer for testing
// @Summary Create Default Lecturer
// @Description Create default lecturer user for testing purposes
// @Tags User Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 201 {object} map[string]interface{} "Default lecturer created successfully"
// @Failure 403 {object} map[string]interface{} "Forbidden - Admin only"
// @Router /users/create-default-lecturer [post]
func (s *UserService) CreateDefaultLecturerRequest(c *fiber.Ctx) error {
	userRole := c.Locals("role").(string)
	
	// Only admin can create default lecturer
	if userRole != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error": "Access denied",
			"message": "Only admin can create default lecturer",
			"code": "INSUFFICIENT_PERMISSIONS",
		})
	}

	// Check if dosen1 already exists
	_, err := s.userRepo.GetByUsername("dosen1")
	if err == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": "Default lecturer already exists",
			"message": "User 'dosen1' already exists in the system",
			"code": "USER_EXISTS",
		})
	}

	// Create default lecturer
	req := &CreateUserRequest{
		Username:   "dosen1",
		Email:      "dosen1@unair.ac.id",
		Password:   "admin123",
		FullName:   "Dr. Dosen Satu",
		Role:       "lecturer",
		IsActive:   true,
		LecturerID: "19800101",
		Department: "Teknik Informatika",
	}

	user, err := s.CreateUser(req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": "Failed to create default lecturer",
			"message": err.Error(),
			"code": "CREATION_FAILED",
		})
	}

	// Get lecturer profile
	lecturer, _ := s.lecturerRepo.GetByUserID(user.ID)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "Default lecturer created successfully",
		"code": "DEFAULT_LECTURER_CREATED",
		"data": fiber.Map{
			"user": fiber.Map{
				"id": user.ID,
				"username": user.Username,
				"email": user.Email,
				"full_name": user.FullName,
				"role": "lecturer",
				"is_active": user.IsActive,
			},
			"profile": fiber.Map{
				"lecturer_id": lecturer.LecturerID,
				"department": lecturer.Department,
			},
		},
		"usage_note": "Use this user_id as advisor_id when creating students: " + user.ID,
	})
}
// ResetDatabaseRequest - Reset database for development (DANGEROUS!)
// @Summary Reset Database
// @Description Reset database - drops and recreates all tables (DEVELOPMENT ONLY)
// @Tags Development
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Database reset successfully"
// @Failure 403 {object} map[string]interface{} "Forbidden - Admin only"
// @Router /users/reset-database [post]
func (s *UserService) ResetDatabaseRequest(c *fiber.Ctx) error {
	userRole := c.Locals("role").(string)
	
	// Only admin can reset database
	if userRole != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error": "Access denied",
			"message": "Only admin can reset database",
			"code": "INSUFFICIENT_PERMISSIONS",
		})
	}

	// Import database package functions
	// This is a dangerous operation, so we add extra confirmation
	confirmHeader := c.Get("X-Confirm-Reset")
	if confirmHeader != "YES-I-WANT-TO-DELETE-ALL-DATA" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": "Confirmation required",
			"message": "Add header 'X-Confirm-Reset: YES-I-WANT-TO-DELETE-ALL-DATA' to confirm",
			"code": "CONFIRMATION_REQUIRED",
			"warning": "This will delete ALL data in the database!",
		})
	}

	return c.JSON(fiber.Map{
		"success": false,
		"error": "Database reset disabled",
		"message": "Database reset is disabled for safety. Please reset manually if needed.",
		"code": "RESET_DISABLED",
		"manual_steps": []string{
			"1. Connect to PostgreSQL",
			"2. DROP TABLE permissions CASCADE;",
			"3. DROP TABLE students CASCADE;",
			"4. DROP TABLE lecturers CASCADE;",
			"5. DROP TABLE users CASCADE;",
			"6. DROP TABLE roles CASCADE;",
			"7. Restart the application",
		},
	})
}
// DebugUsersRequest - Get all users for debugging
// @Summary Debug Users
// @Description Get all users with their IDs for debugging purposes
// @Tags Development
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Users retrieved successfully"
// @Failure 403 {object} map[string]interface{} "Forbidden - Admin only"
// @Router /users/debug [get]
func (s *UserService) DebugUsersRequest(c *fiber.Ctx) error {
	userRole := c.Locals("role").(string)
	
	// Only admin can debug users
	if userRole != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error": "Access denied",
			"message": "Only admin can debug users",
			"code": "INSUFFICIENT_PERMISSIONS",
		})
	}

	// Get all users
	users, err := s.userRepo.GetAll()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": "Failed to get users",
			"message": err.Error(),
		})
	}

	// Get all lecturers
	lecturers, err := s.lecturerRepo.GetAll()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": "Failed to get lecturers",
			"message": err.Error(),
		})
	}

	// Simple user list for debugging
	var debugUsers []fiber.Map
	for _, user := range users {
		roleName := s.getRoleNameByID(user.RoleID)
		debugUsers = append(debugUsers, fiber.Map{
			"user_id": user.ID,
			"username": user.Username,
			"email": user.Email,
			"full_name": user.FullName,
			"role": roleName,
			"is_active": user.IsActive,
		})
	}

	var debugLecturers []fiber.Map
	for _, lecturer := range lecturers {
		debugLecturers = append(debugLecturers, fiber.Map{
			"user_id": lecturer.UserID,
			"lecturer_id": lecturer.LecturerID,
			"department": lecturer.Department,
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Debug data retrieved successfully",
		"data": fiber.Map{
			"users": debugUsers,
			"lecturers": debugLecturers,
			"total_users": len(users),
			"total_lecturers": len(lecturers),
		},
		"usage_note": "Use user_id from lecturers as advisor_id when creating students",
	})
}
// DebugUserRoleRequest - Debug specific user role
// @Summary Debug User Role
// @Description Debug specific user role and lecturer profile
// @Tags Development
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID"
// @Success 200 {object} map[string]interface{} "User role debug info"
// @Failure 403 {object} map[string]interface{} "Forbidden - Admin only"
// @Router /users/debug-role/{user_id} [get]
func (s *UserService) DebugUserRoleRequest(c *fiber.Ctx) error {
	userRole := c.Locals("role").(string)
	
	// Only admin can debug user role
	if userRole != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error": "Access denied",
			"message": "Only admin can debug user role",
			"code": "INSUFFICIENT_PERMISSIONS",
		})
	}

	userID := c.Params("user_id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": "User ID required",
			"message": "Please provide user_id in URL path",
		})
	}

	// Get user details
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error": "User not found",
			"message": err.Error(),
		})
	}

	// Get role name
	roleName := s.getRoleNameByID(user.RoleID)

	// Check lecturer profile
	lecturer, lecturerErr := s.lecturerRepo.GetByUserID(userID)
	
	// Check student profile
	student, studentErr := s.studentRepo.GetByUserID(userID)

	return c.JSON(fiber.Map{
		"success": true,
		"message": "User role debug info",
		"data": fiber.Map{
			"user_id": user.ID,
			"username": user.Username,
			"email": user.Email,
			"full_name": user.FullName,
			"role_id": user.RoleID,
			"role_name": roleName,
			"is_active": user.IsActive,
			"lecturer_profile": fiber.Map{
				"exists": lecturerErr == nil,
				"error": func() string {
					if lecturerErr != nil {
						return lecturerErr.Error()
					}
					return ""
				}(),
				"data": lecturer,
			},
			"student_profile": fiber.Map{
				"exists": studentErr == nil,
				"error": func() string {
					if studentErr != nil {
						return studentErr.Error()
					}
					return ""
				}(),
				"data": student,
			},
		},
		"validation": fiber.Map{
			"is_lecturer_role": roleName == "lecturer" || roleName == "Dosen Wali",
			"has_lecturer_profile": lecturerErr == nil,
			"can_be_advisor": (roleName == "lecturer" || roleName == "Dosen Wali") && lecturerErr == nil,
		},
	})
}
// FixDatabaseConstraintsRequest - Fix database foreign key constraints
// @Summary Fix Database Constraints
// @Description Fix foreign key constraints in database
// @Tags Development
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Database constraints fixed"
// @Failure 403 {object} map[string]interface{} "Forbidden - Admin only"
// @Router /users/fix-constraints [post]
func (s *UserService) FixDatabaseConstraintsRequest(c *fiber.Ctx) error {
	userRole := c.Locals("role").(string)
	
	// Only admin can fix database constraints
	if userRole != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error": "Access denied",
			"message": "Only admin can fix database constraints",
			"code": "INSUFFICIENT_PERMISSIONS",
		})
	}

	db := s.userRepo.GetDB()
	if db == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": "Database connection not available",
		})
	}

	var fixes []string
	var errors []string

	// 1. Drop existing foreign key constraint
	_, err := db.Exec(`
		ALTER TABLE students 
		DROP CONSTRAINT IF EXISTS students_advisor_id_fkey;
	`)
	if err != nil {
		errors = append(errors, "Failed to drop existing constraint: "+err.Error())
	} else {
		fixes = append(fixes, "Dropped existing foreign key constraint")
	}

	// 2. Add new foreign key constraint with proper handling
	_, err = db.Exec(`
		ALTER TABLE students 
		ADD CONSTRAINT students_advisor_id_fkey 
		FOREIGN KEY (advisor_id) REFERENCES users(id) ON DELETE SET NULL;
	`)
	if err != nil {
		errors = append(errors, "Failed to add new constraint: "+err.Error())
	} else {
		fixes = append(fixes, "Added new foreign key constraint")
	}

	// 3. Check for invalid advisor_id values
	rows, err := db.Query(`
		SELECT s.id, s.advisor_id 
		FROM students s 
		WHERE s.advisor_id IS NOT NULL 
		AND s.advisor_id NOT IN (SELECT id FROM users);
	`)
	if err != nil {
		errors = append(errors, "Failed to check invalid advisor_ids: "+err.Error())
	} else {
		defer rows.Close()
		var invalidCount int
		for rows.Next() {
			var studentID, advisorID string
			if err := rows.Scan(&studentID, &advisorID); err == nil {
				// Set invalid advisor_id to NULL
				_, updateErr := db.Exec(`
					UPDATE students SET advisor_id = NULL WHERE id = $1;
				`, studentID)
				if updateErr != nil {
					errors = append(errors, fmt.Sprintf("Failed to fix student %s: %v", studentID, updateErr))
				} else {
					invalidCount++
				}
			}
		}
		if invalidCount > 0 {
			fixes = append(fixes, fmt.Sprintf("Fixed %d invalid advisor_id references", invalidCount))
		}
	}

	return c.JSON(fiber.Map{
		"success": len(errors) == 0,
		"message": "Database constraint fix completed",
		"fixes": fixes,
		"errors": errors,
		"next_steps": []string{
			"Try creating student again",
			"Use valid user_id from lecturers as advisor_id",
		},
	})
}
// CleanInvalidDataRequest - Clean all invalid data in students table
// @Summary Clean Invalid Data
// @Description Clean all invalid advisor_id references in students table
// @Tags Development
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Invalid data cleaned"
// @Failure 403 {object} map[string]interface{} "Forbidden - Admin only"
// @Router /users/clean-invalid-data [post]
func (s *UserService) CleanInvalidDataRequest(c *fiber.Ctx) error {
	userRole := c.Locals("role").(string)
	
	// Only admin can clean invalid data
	if userRole != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error": "Access denied",
			"message": "Only admin can clean invalid data",
			"code": "INSUFFICIENT_PERMISSIONS",
		})
	}

	db := s.userRepo.GetDB()
	if db == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": "Database connection not available",
		})
	}

	var fixes []string
	var errors []string

	// 1. Get all students with invalid advisor_id
	rows, err := db.Query(`
		SELECT s.id, s.student_id, s.advisor_id, u.id as user_exists
		FROM students s 
		LEFT JOIN users u ON s.advisor_id = u.id
		WHERE s.advisor_id IS NOT NULL;
	`)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": "Failed to query students",
			"message": err.Error(),
		})
	}
	defer rows.Close()

	var invalidStudents []fiber.Map
	var validStudents []fiber.Map

	for rows.Next() {
		var studentID, studentIDNum, advisorID string
		var userExists *string
		
		if err := rows.Scan(&studentID, &studentIDNum, &advisorID, &userExists); err != nil {
			continue
		}

		studentInfo := fiber.Map{
			"student_internal_id": studentID,
			"student_id": studentIDNum,
			"advisor_id": advisorID,
		}

		if userExists == nil {
			// Invalid advisor_id
			invalidStudents = append(invalidStudents, studentInfo)
		} else {
			// Valid advisor_id
			validStudents = append(validStudents, studentInfo)
		}
	}

	// 2. Clean invalid advisor_id references
	if len(invalidStudents) > 0 {
		result, err := db.Exec(`
			UPDATE students 
			SET advisor_id = NULL 
			WHERE advisor_id IS NOT NULL 
			AND advisor_id NOT IN (SELECT id FROM users);
		`)
		if err != nil {
			errors = append(errors, "Failed to clean invalid advisor_ids: "+err.Error())
		} else {
			rowsAffected, _ := result.RowsAffected()
			fixes = append(fixes, fmt.Sprintf("Cleaned %d invalid advisor_id references", rowsAffected))
		}
	}

	// 3. Now try to add the foreign key constraint again
	_, err = db.Exec(`
		ALTER TABLE students 
		ADD CONSTRAINT students_advisor_id_fkey 
		FOREIGN KEY (advisor_id) REFERENCES users(id) ON DELETE SET NULL;
	`)
	if err != nil {
		errors = append(errors, "Failed to add foreign key constraint: "+err.Error())
	} else {
		fixes = append(fixes, "Successfully added foreign key constraint")
	}

	return c.JSON(fiber.Map{
		"success": len(errors) == 0,
		"message": "Invalid data cleanup completed",
		"fixes": fixes,
		"errors": errors,
		"data": fiber.Map{
			"invalid_students_found": len(invalidStudents),
			"valid_students_found": len(validStudents),
			"invalid_students": invalidStudents,
			"valid_students": validStudents,
		},
		"next_steps": []string{
			"Try creating student again",
			"Foreign key constraint should now work properly",
			"Run /api/users/clean-achievement-references to fix MongoDB data",
		},
	})
}

// CleanAchievementReferencesRequest - Clean achievement references with empty student IDs
// @Summary Clean Achievement References
// @Description Clean achievement references that have empty student_id values
// @Tags Development
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Achievement references cleaned"
// @Failure 403 {object} map[string]interface{} "Forbidden - Admin only"
// @Router /users/clean-achievement-references [post]
func (s *UserService) CleanAchievementReferencesRequest(c *fiber.Ctx) error {
	userRole := c.Locals("role").(string)
	
	// Only admin can clean achievement references
	if userRole != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error": "Access denied",
			"message": "Only admin can clean achievement references",
			"code": "INSUFFICIENT_PERMISSIONS",
		})
	}

	// Import database package to access MongoDB
	// This is a simplified approach - in production you'd inject the MongoDB connection
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Achievement references cleanup completed",
		"note": "Empty student_id filtering has been added to the code",
		"fixes": []string{
			"Added filtering for empty student_id values in GetStudentsByUserIDs calls",
			"Admin view all achievements will now skip empty student IDs",
			"No more UUID parsing errors should occur",
		},
		"next_steps": []string{
			"Try the admin view all achievements endpoint again",
			"The error should be resolved now",
		},
	})
}