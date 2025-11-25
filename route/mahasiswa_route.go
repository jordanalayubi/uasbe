package route

import (
	"UASBE/app/model"
	"UASBE/app/service"
	"UASBE/middleware"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type MahasiswaHandler struct {
	mahasiswaService service.MahasiswaService
}

func NewMahasiswaHandler(mahasiswaService service.MahasiswaService) *MahasiswaHandler {
	return &MahasiswaHandler{mahasiswaService: mahasiswaService}
}

type CreateMahasiswaRequest struct {
	NIM      string `json:"nim" validate:"required"`
	Nama     string `json:"nama" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Jurusan  string `json:"jurusan" validate:"required"`
	Angkatan int    `json:"angkatan" validate:"required"`
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

func (h *MahasiswaHandler) Create(c *fiber.Ctx) error {
	var req CreateMahasiswaRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Message: "Invalid request body",
		})
	}

	mahasiswa := &model.Mahasiswa{
		NIM:      req.NIM,
		Nama:     req.Nama,
		Email:    req.Email,
		Jurusan:  req.Jurusan,
		Angkatan: req.Angkatan,
	}

	if err := h.mahasiswaService.Create(mahasiswa, req.Username, req.Password); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(model.Response{
		Success: true,
		Message: "Mahasiswa created successfully",
		Data:    mahasiswa,
	})
}

func (h *MahasiswaHandler) GetAll(c *fiber.Ctx) error {
	mahasiswa, err := h.mahasiswaService.GetAll()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.Response{
			Success: false,
			Message: err.Error(),
		})
	}

	return c.JSON(model.Response{
		Success: true,
		Message: "Success",
		Data:    mahasiswa,
	})
}

func (h *MahasiswaHandler) GetByID(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Message: "Invalid ID",
		})
	}

	mahasiswa, err := h.mahasiswaService.GetByID(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(model.Response{
			Success: false,
			Message: "Mahasiswa not found",
		})
	}

	return c.JSON(model.Response{
		Success: true,
		Message: "Success",
		Data:    mahasiswa,
	})
}

func (h *MahasiswaHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Message: "Invalid ID",
		})
	}

	var mahasiswa model.Mahasiswa
	if err := c.BodyParser(&mahasiswa); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Message: "Invalid request body",
		})
	}

	if err := h.mahasiswaService.Update(uint(id), &mahasiswa); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Message: err.Error(),
		})
	}

	return c.JSON(model.Response{
		Success: true,
		Message: "Mahasiswa updated successfully",
		Data:    mahasiswa,
	})
}

func (h *MahasiswaHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Message: "Invalid ID",
		})
	}

	if err := h.mahasiswaService.Delete(uint(id)); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Message: err.Error(),
		})
	}

	return c.JSON(model.Response{
		Success: true,
		Message: "Mahasiswa deleted successfully",
	})
}

func SetupMahasiswaRoutes(app *fiber.App, mahasiswaService service.MahasiswaService) {
	handler := NewMahasiswaHandler(mahasiswaService)
	
	mahasiswa := app.Group("/api/mahasiswa")
	mahasiswa.Use(middleware.AuthMiddleware)
	
	mahasiswa.Get("/", handler.GetAll)
	mahasiswa.Get("/:id", handler.GetByID)
	mahasiswa.Post("/", middleware.RoleMiddleware("admin"), handler.Create)
	mahasiswa.Put("/:id", middleware.RoleMiddleware("admin"), handler.Update)
	mahasiswa.Delete("/:id", middleware.RoleMiddleware("admin"), handler.Delete)
}
