package model

import (
	"time"
)

type Role struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type RolePermission struct {
	ID           string `json:"id" db:"id"`
	RoleID       string `json:"role_id" db:"role_id"`
	PermissionID string `json:"permission_id" db:"permission_id"`
}