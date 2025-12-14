package repository

import (
	"UASBE/app/model"
	"UASBE/database"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository() *UserRepository {
	return &UserRepository{
		db: database.GetPostgresDB(),
	}
}

func (r *UserRepository) Create(user *model.User) error {
	user.ID = uuid.New().String()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	
	query := `
		INSERT INTO users (id, username, email, password_hash, full_name, role_id, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	
	_, err := r.db.Exec(query, user.ID, user.Username, user.Email, user.Password, 
		user.FullName, user.RoleID, user.IsActive, user.CreatedAt, user.UpdatedAt)
	
	return err
}

func (r *UserRepository) GetByID(id string) (*model.User, error) {
	var user model.User
	
	query := `
		SELECT id, username, email, password_hash, full_name, role_id, is_active, created_at, updated_at
		FROM users WHERE id = $1
	`
	
	err := r.db.QueryRow(query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.Password,
		&user.FullName, &user.RoleID, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &user, nil
}

func (r *UserRepository) GetByUsername(username string) (*model.User, error) {
	var user model.User
	
	query := `
		SELECT id, username, email, password_hash, full_name, role_id, is_active, created_at, updated_at
		FROM users WHERE username = $1
	`
	
	err := r.db.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.Password,
		&user.FullName, &user.RoleID, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &user, nil
}

func (r *UserRepository) GetByEmail(email string) (*model.User, error) {
	var user model.User
	
	query := `
		SELECT id, username, email, password_hash, full_name, role_id, is_active, created_at, updated_at
		FROM users WHERE email = $1
	`
	
	err := r.db.QueryRow(query, email).Scan(
		&user.ID, &user.Username, &user.Email, &user.Password,
		&user.FullName, &user.RoleID, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &user, nil
}

func (r *UserRepository) Update(user *model.User) error {
	user.UpdatedAt = time.Now()
	
	query := `
		UPDATE users 
		SET username = $2, email = $3, password_hash = $4, full_name = $5, 
		    role_id = $6, is_active = $7, updated_at = $8
		WHERE id = $1
	`
	
	_, err := r.db.Exec(query, user.ID, user.Username, user.Email, user.Password,
		user.FullName, user.RoleID, user.IsActive, user.UpdatedAt)
	
	return err
}

func (r *UserRepository) Delete(id string) error {
	query := "DELETE FROM users WHERE id = $1"
	_, err := r.db.Exec(query, id)
	return err
}

func (r *UserRepository) GetAll() ([]model.User, error) {
	var users []model.User
	
	query := `
		SELECT id, username, email, password_hash, full_name, role_id, is_active, created_at, updated_at
		FROM users ORDER BY created_at DESC
	`
	
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var user model.User
		err := rows.Scan(
			&user.ID, &user.Username, &user.Email, &user.Password,
			&user.FullName, &user.RoleID, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	
	return users, nil
}

func (r *UserRepository) GetDB() *sql.DB {
	return r.db
}

// GetByUsernameOrEmail gets user by username or email for login
func (r *UserRepository) GetByUsernameOrEmail(credential string) (*model.User, error) {
	var user model.User
	
	query := `
		SELECT id, username, email, password_hash, full_name, role_id, is_active, created_at, updated_at
		FROM users WHERE username = $1 OR email = $1
	`
	
	err := r.db.QueryRow(query, credential).Scan(
		&user.ID, &user.Username, &user.Email, &user.Password,
		&user.FullName, &user.RoleID, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &user, nil
}

// GetUserWithRoleAndPermissions gets user with role and permissions for login response
func (r *UserRepository) GetUserWithRoleAndPermissions(userID string) (*model.User, *model.Role, []model.Permission, error) {
	// Get user
	user, err := r.GetByID(userID)
	if err != nil {
		return nil, nil, nil, err
	}

	// Get role
	var role model.Role
	roleQuery := `
		SELECT id, name, description, created_at
		FROM roles WHERE id = $1
	`
	
	err = r.db.QueryRow(roleQuery, user.RoleID).Scan(
		&role.ID, &role.Name, &role.Description, &role.CreatedAt,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	// Get permissions
	permissionsQuery := `
		SELECT p.id, p.name, p.resource, p.action, p.description
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = $1
	`
	
	rows, err := r.db.Query(permissionsQuery, user.RoleID)
	if err != nil {
		return user, &role, nil, nil // Return user and role even if permissions fail
	}
	defer rows.Close()

	var permissions []model.Permission
	for rows.Next() {
		var permission model.Permission
		err := rows.Scan(
			&permission.ID, &permission.Name, &permission.Resource,
			&permission.Action, &permission.Description,
		)
		if err != nil {
			continue
		}
		permissions = append(permissions, permission)
	}

	return user, &role, permissions, nil
}