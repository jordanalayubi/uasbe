package route

import (
	"UASBE/app/model"
	"UASBE/app/service"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req model.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Message: "Invalid request body",
		})
	}

	response, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(model.Response{
			Success: false,
			Message: err.Error(),
		})
	}

	return c.JSON(model.Response{
		Success: true,
		Message: "Login successful",
		Data:    response,
	})
}

func SetupAuthRoutes(app *fiber.App, authService service.AuthService) {
	handler := NewAuthHandler(authService)
	
	auth := app.Group("/api/auth")
	auth.Post("/login", handler.Login)
}
