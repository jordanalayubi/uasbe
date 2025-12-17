package service

import (
	"UASBE/app/model"
	"UASBE/app/repository"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo     *repository.UserRepository
	studentRepo  *repository.StudentRepository
	lecturerRepo *repository.LecturerRepository
	jwtSecret    string
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token       string             `json:"token"`
	User        *model.User        `json:"user"`
	Role        *model.Role        `json:"role"`
	Permissions []model.Permission `json:"permissions"`
	ExpiresAt   time.Time          `json:"expires_at"`
}

type RegisterRequest struct {
	Username     string `json:"username"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	FullName     string `json:"full_name"`
	Role         string `json:"role"` // "admin", "student", "lecturer"
	StudentID    string `json:"student_id,omitempty"`
	LecturerID   string `json:"lecturer_id,omitempty"`
	ProgramStudy string `json:"program_study,omitempty"`
	AcademicYear string `json:"academic_year,omitempty"`
	Department   string `json:"department,omitempty"`
}

func NewAuthService(userRepo *repository.UserRepository, studentRepo *repository.StudentRepository, lecturerRepo *repository.LecturerRepository, jwtSecret string) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		studentRepo:  studentRepo,
		lecturerRepo: lecturerRepo,
		jwtSecret:    jwtSecret,
	}
}

func (s *AuthService) Login(req *LoginRequest) (*LoginResponse, error) {
	// FR-001 Step 1: User mengirim kredensial
	// Support login with username or email
	user, err := s.getUserByCredential(req.Username)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// FR-001 Step 2: Sistem memvalidasi kredensial
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// FR-001 Step 3: Sistem mengecek status aktif user
	if !user.IsActive {
		return nil, errors.New("account is deactivated")
	}

	// Get user role and permissions
	role, permissions, err := s.getUserRoleAndPermissions(user.RoleID)
	if err != nil {
		return nil, errors.New("failed to get user role")
	}

	// FR-001 Step 4: Sistem generate JWT token dengan role dan permissions
	token, expiresAt, err := s.generateTokenWithRoleAndPermissions(user.ID, user.Username, role, permissions)
	if err != nil {
		return nil, errors.New("failed to generate token")
	}

	// FR-001 Step 5: Return token dan user profile
	// Remove password from response
	user.Password = ""

	return &LoginResponse{
		Token:       token,
		User:        user,
		Role:        role,
		Permissions: permissions,
		ExpiresAt:   expiresAt,
	}, nil
}

// Helper method to get user by username or email
func (s *AuthService) getUserByCredential(credential string) (*model.User, error) {
	// Try username first
	user, err := s.userRepo.GetByUsername(credential)
	if err != nil {
		// Try email if username fails
		user, err = s.userRepo.GetByEmail(credential)
		if err != nil {
			return nil, errors.New("user not found with credential: " + credential)
		}
	}
	return user, nil
}

// Helper method to get user role and permissions
func (s *AuthService) getUserRoleAndPermissions(roleID string) (*model.Role, []model.Permission, error) {
	db := s.userRepo.GetDB()
	if db == nil {
		return nil, nil, errors.New("database connection not available")
	}

	// Get role
	var role model.Role
	roleQuery := `SELECT id, name, description, created_at FROM roles WHERE id = $1`
	err := db.QueryRow(roleQuery, roleID).Scan(&role.ID, &role.Name, &role.Description, &role.CreatedAt)
	if err != nil {
		return nil, nil, err
	}

	// Get permissions
	permissionsQuery := `
		SELECT p.id, p.name, p.resource, p.action, p.description
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = $1
	`
	
	rows, err := db.Query(permissionsQuery, roleID)
	if err != nil {
		return &role, nil, nil // Return role even if permissions fail
	}
	defer rows.Close()

	var permissions []model.Permission
	for rows.Next() {
		var permission model.Permission
		err := rows.Scan(&permission.ID, &permission.Name, &permission.Resource, &permission.Action, &permission.Description)
		if err != nil {
			continue
		}
		permissions = append(permissions, permission)
	}

	return &role, permissions, nil
}

func (s *AuthService) Register(req *RegisterRequest) (*model.User, error) {
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

	// Get role ID based on role name
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
		IsActive:  true,
	}

	err = s.userRepo.Create(user)
	if err != nil {
		return nil, err
	}

	// Create additional profile based on role
	switch req.Role {
	case "student":
		student := &model.Student{
			UserID:       user.ID,
			StudentID:    req.StudentID,
			ProgramStudy: req.ProgramStudy,
			AcademicYear: req.AcademicYear,
		}
		err = s.studentRepo.Create(student)
		if err != nil {
			// Rollback user creation if student creation fails
			s.userRepo.Delete(user.ID)
			return nil, err
		}

	case "lecturer":
		lecturer := &model.Lecturer{
			UserID:     user.ID,
			LecturerID: req.LecturerID,
			Department: req.Department,
		}
		err = s.lecturerRepo.Create(lecturer)
		if err != nil {
			// Rollback user creation if lecturer creation fails
			s.userRepo.Delete(user.ID)
			return nil, err
		}
	}

	// Remove password from response
	user.Password = ""
	return user, nil
}

func (s *AuthService) generateToken(userID string, username string) (string, time.Time, error) {
	expiresAt := time.Now().Add(24 * time.Hour)

	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"exp":      expiresAt.Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

func (s *AuthService) generateTokenWithRoleAndPermissions(userID string, username string, role *model.Role, permissions []model.Permission) (string, time.Time, error) {
	expiresAt := time.Now().Add(24 * time.Hour)

	// Create permission strings for JWT
	var permissionStrings []string
	for _, perm := range permissions {
		permissionStrings = append(permissionStrings, perm.Resource+":"+perm.Action)
	}

	claims := jwt.MapClaims{
		"user_id":     userID,
		"username":    username,
		"role":        role.Name,
		"role_id":     role.ID,
		"permissions": permissionStrings,
		"exp":         expiresAt.Unix(),
		"iat":         time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

func (s *AuthService) ValidateToken(tokenString string) (*jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return &claims, nil
	}

	return nil, errors.New("invalid token")
}

// LoginRequest handles user login
// @Summary User Login
// @Description Authenticate user with username/email and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param credentials body LoginRequest true "Login credentials"
// @Success 200 {object} map[string]interface{} "Login successful"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Router /auth/login [post]
func (s *AuthService) LoginRequest(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": "Invalid request body",
			"message": "Please provide valid JSON data",
			"code": "INVALID_REQUEST_BODY",
		})
	}

	// Validate request
	if req.Username == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": "Validation failed",
			"message": "Username and password are required",
			"code": "MISSING_CREDENTIALS",
			"details": fiber.Map{
				"username": req.Username == "",
				"password": req.Password == "",
			},
		})
	}

	// Process login
	response, err := s.Login(&req)
	if err != nil {
		// Enhanced error responses
		var errorCode string
		var message string
		
		switch err.Error() {
		case "invalid credentials":
			errorCode = "INVALID_CREDENTIALS"
			message = "Username/email or password is incorrect"
		case "account is deactivated":
			errorCode = "ACCOUNT_DEACTIVATED"
			message = "Your account has been deactivated. Please contact administrator"
		default:
			errorCode = "LOGIN_FAILED"
			message = "Login failed. Please try again"
		}
		
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error": err.Error(),
			"message": message,
			"code": errorCode,
			"timestamp": time.Now(),
		})
	}

	// Get additional user profile info
	var profileInfo fiber.Map
	if response.Role.Name == "student" {
		student, err := s.studentRepo.GetByUserID(response.User.ID)
		if err == nil {
			profileInfo = fiber.Map{
				"type": "student",
				"student_id": student.StudentID,
				"program_study": student.ProgramStudy,
				"academic_year": student.AcademicYear,
				"advisor_id": student.AdvisorID,
			}
		}
	} else if response.Role.Name == "lecturer" {
		lecturer, err := s.lecturerRepo.GetByUserID(response.User.ID)
		if err == nil {
			profileInfo = fiber.Map{
				"type": "lecturer",
				"lecturer_id": lecturer.LecturerID,
				"department": lecturer.Department,
			}
		}
	} else {
		profileInfo = fiber.Map{
			"type": "admin",
		}
	}

	// Enhanced success response
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Login successful",
		"code": "LOGIN_SUCCESS",
		"data": fiber.Map{
			"token": response.Token,
			"expires_at": response.ExpiresAt,
			"expires_in_seconds": int(time.Until(response.ExpiresAt).Seconds()),
			"user": fiber.Map{
				"id": response.User.ID,
				"username": response.User.Username,
				"email": response.User.Email,
				"full_name": response.User.FullName,
				"is_active": response.User.IsActive,
				"created_at": response.User.CreatedAt,
			},
			"role": fiber.Map{
				"id": response.Role.ID,
				"name": response.Role.Name,
				"description": response.Role.Description,
			},
			"permissions": response.Permissions,
			"profile": profileInfo,
		},
		"session_info": fiber.Map{
			"login_time": time.Now(),
			"ip_address": c.IP(),
			"user_agent": c.Get("User-Agent"),
		},
		"next_steps": []string{
			"Use the token in Authorization header for subsequent requests",
			"Token format: Bearer <token>",
			"Token expires at: " + response.ExpiresAt.Format("2006-01-02 15:04:05"),
		},
	})
}

func (s *AuthService) RegisterRequest(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if req.Username == "" || req.Email == "" || req.Password == "" || req.FullName == "" || req.Role == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "All fields are required",
		})
	}

	// Validate role-specific fields
	if req.Role == "student" && (req.StudentID == "" || req.ProgramStudy == "" || req.AcademicYear == "") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Student ID, program study, and academic year are required for students",
		})
	}

	if req.Role == "lecturer" && (req.LecturerID == "" || req.Department == "") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Lecturer ID and department are required for lecturers",
		})
	}

	// Process registration
	user, err := s.Register(&req)
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

func (s *AuthService) getRoleIDByName(roleName string) (string, error) {
	// Query PostgreSQL for role ID
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