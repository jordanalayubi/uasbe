package main

import (
	"UASBE/database"
	"database/sql"
	"log"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Connect to database
	database.Connect()
	defer database.Disconnect()

	db := database.GetPostgresDB()
	if db == nil {
		log.Fatal("Failed to connect to PostgreSQL")
	}

	// Add FR-011 permissions
	err := addStatisticsPermissions(db)
	if err != nil {
		log.Fatal("Failed to add FR-011 permissions:", err)
	}

	log.Println("FR-011 permissions setup completed successfully!")
}

func addStatisticsPermissions(db *sql.DB) error {
	// Define permissions for FR-011
	permissions := []struct {
		name        string
		resource    string
		action      string
		description string
	}{
		{
			name:        "statistics:view_own",
			resource:    "statistics",
			action:      "view_own",
			description: "View own achievement statistics",
		},
		{
			name:        "statistics:view_advisee",
			resource:    "statistics",
			action:      "view_advisee",
			description: "View advisee achievement statistics",
		},
		{
			name:        "statistics:view_all",
			resource:    "statistics",
			action:      "view_all",
			description: "View all achievement statistics",
		},
	}

	// Add permissions
	for _, perm := range permissions {
		// Check if permission already exists
		var count int
		checkQuery := "SELECT COUNT(*) FROM permissions WHERE name = $1"
		err := db.QueryRow(checkQuery, perm.name).Scan(&count)
		if err != nil {
			return err
		}

		if count > 0 {
			log.Printf("Permission '%s' already exists", perm.name)
			continue
		}

		// Insert the permission
		insertPermissionQuery := `
			INSERT INTO permissions (id, name, resource, action, description)
			VALUES (gen_random_uuid(), $1, $2, $3, $4)
		`
		
		_, err = db.Exec(insertPermissionQuery, perm.name, perm.resource, perm.action, perm.description)
		if err != nil {
			return err
		}
		log.Printf("Added permission: %s", perm.name)
	}

	// Assign permissions to roles
	rolePermissions := map[string][]string{
		"student": {"statistics:view_own"},
		"lecturer": {"statistics:view_own", "statistics:view_advisee"},
		"admin": {"statistics:view_own", "statistics:view_advisee", "statistics:view_all"},
	}

	for roleName, permNames := range rolePermissions {
		// Get role ID
		var roleID string
		getRoleQuery := "SELECT id FROM roles WHERE name = $1"
		err := db.QueryRow(getRoleQuery, roleName).Scan(&roleID)
		if err != nil {
			log.Printf("Warning: Role '%s' not found: %v", roleName, err)
			continue
		}

		for _, permName := range permNames {
			// Get permission ID
			var permissionID string
			getPermissionQuery := "SELECT id FROM permissions WHERE name = $1"
			err := db.QueryRow(getPermissionQuery, permName).Scan(&permissionID)
			if err != nil {
				log.Printf("Warning: Permission '%s' not found: %v", permName, err)
				continue
			}

			// Check if role_permission already exists
			var rolePermCount int
			checkRolePermQuery := "SELECT COUNT(*) FROM role_permissions WHERE role_id = $1 AND permission_id = $2"
			err = db.QueryRow(checkRolePermQuery, roleID, permissionID).Scan(&rolePermCount)
			if err != nil {
				return err
			}

			if rolePermCount > 0 {
				log.Printf("Role '%s' already has permission '%s'", roleName, permName)
				continue
			}

			// Assign permission to role
			assignPermissionQuery := `
				INSERT INTO role_permissions (role_id, permission_id)
				VALUES ($1, $2)
			`
			
			_, err = db.Exec(assignPermissionQuery, roleID, permissionID)
			if err != nil {
				return err
			}
			log.Printf("Assigned permission '%s' to role '%s'", permName, roleName)
		}
	}

	return nil
}