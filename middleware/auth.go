package middleware

import (
	"UASBE/app/service"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// AuthMiddleware - Basic JWT authentication
func AuthMiddleware(authService *service.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// FR-002 Step 1: Ekstrak JWT dari header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authorization header required",
			})
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid authorization format",
			})
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		// FR-002 Step 2: Validasi token
		claims, err := authService.ValidateToken(token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token",
			})
		}

		// Extract user information from claims
		userID, ok := (*claims)["user_id"].(string)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token claims",
			})
		}

		// Store user information in context
		c.Locals("user_id", userID)
		c.Locals("username", (*claims)["username"])
		c.Locals("role", (*claims)["role"])
		c.Locals("role_id", (*claims)["role_id"])
		
		// Store permissions if available
		if permissions, ok := (*claims)["permissions"].([]interface{}); ok {
			var permissionStrings []string
			for _, perm := range permissions {
				if permStr, ok := perm.(string); ok {
					permissionStrings = append(permissionStrings, permStr)
				}
			}
			c.Locals("permissions", permissionStrings)
		}

		return c.Next()
	}
}

// RBACMiddleware - Role-Based Access Control middleware
func RBACMiddleware(authService *service.AuthService, requiredPermission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// FR-002 Step 3: Load user permissions dari token (sudah di-cache di JWT)
		permissions, ok := c.Locals("permissions").([]string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "No permissions found",
			})
		}

		// FR-002 Step 4: Check apakah user memiliki permission yang diperlukan
		hasPermission := false
		for _, permission := range permissions {
			if permission == requiredPermission {
				hasPermission = true
				break
			}
		}

		// FR-002 Step 5: Allow/deny request
		if !hasPermission {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Insufficient permissions",
				"required_permission": requiredPermission,
			})
		}

		return c.Next()
	}
}

// RoleMiddleware - Simple role-based middleware
func RoleMiddleware(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole, ok := c.Locals("role").(string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "No role found",
			})
		}

		// Check if user role is in allowed roles
		for _, role := range allowedRoles {
			if userRole == role {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Insufficient role permissions",
			"user_role": userRole,
			"allowed_roles": allowedRoles,
		})
	}
}

// PermissionMiddleware - Permission-based middleware (alternative to RBAC)
func PermissionMiddleware(authService *service.AuthService, resource, action string) fiber.Handler {
	requiredPermission := resource + ":" + action
	return RBACMiddleware(authService, requiredPermission)
}

// AdminOnlyMiddleware - Shortcut for admin-only endpoints
func AdminOnlyMiddleware() fiber.Handler {
	return RoleMiddleware("admin")
}

// LecturerOrAdminMiddleware - For lecturer and admin access
func LecturerOrAdminMiddleware() fiber.Handler {
	return RoleMiddleware("admin", "lecturer")
}