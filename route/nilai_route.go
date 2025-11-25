package route

import (
	"UASBE/app/model"
	"UASBE/app/service"
	"UASBE/middleware"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type NilaiHandler struct {
	nilaiService service.NilaiService
}

func NewNilaiHandler(nilaiService service.NilaiService) *NilaiHandler {
	return &NilaiHandler{nilaiService: nilaiService}
}

func (h *NilaiHandler) Create(c *fiber.Ctx) error {
	var nilai model.Nilai
	if err := c.BodyParser(&nilai); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Message: "Invalid request body",
		})
	}

	if err := h.nilaiService.Create(&nilai); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(model.Response{
		Success: true,
		Message: "Nilai created successfully",
		Data:    nilai,
	})
}

func (h *NilaiHandler) GetAll(c *fiber.Ctx) error {
	nilai, err := h.nilaiService.GetAll()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.Response{
			Success: false,
			Message: err.Error(),
		})
	}

	return c.JSON(model.Response{
		Success: true,
		Message: "Success",
		Data:    nilai,
	})
}

func (h *NilaiHandler) GetByID(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Message: "Invalid ID",
		})
	}

	nilai, err := h.nilaiService.GetByID(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(model.Response{
			Success: false,
			Message: "Nilai not found",
		})
	}

	return c.JSON(model.Response{
		Success: true,
		Message: "Success",
		Data:    nilai,
	})
}

func (h *NilaiHandler) GetByMahasiswaID(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("mahasiswa_id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Message: "Invalid Mahasiswa ID",
		})
	}

	nilai, err := h.nilaiService.GetByMahasiswaID(uint(id))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.Response{
			Success: false,
			Message: err.Error(),
		})
	}

	return c.JSON(model.Response{
		Success: true,
		Message: "Success",
		Data:    nilai,
	})
}

func (h *NilaiHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Message: "Invalid ID",
		})
	}

	var nilai model.Nilai
	if err := c.BodyParser(&nilai); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Message: "Invalid request body",
		})
	}

	if err := h.nilaiService.Update(uint(id), &nilai); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Message: err.Error(),
		})
	}

	return c.JSON(model.Response{
		Success: true,
		Message: "Nilai updated successfully",
		Data:    nilai,
	})
}

func (h *NilaiHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Message: "Invalid ID",
		})
	}

	if err := h.nilaiService.Delete(uint(id)); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Message: err.Error(),
		})
	}

	return c.JSON(model.Response{
		Success: true,
		Message: "Nilai deleted successfully",
	})
}

func SetupNilaiRoutes(app *fiber.App, nilaiService service.NilaiService) {
	handler := NewNilaiHandler(nilaiService)
	
	nilai := app.Group("/api/nilai")
	nilai.Use(middleware.AuthMiddleware)
	
	nilai.Get("/", handler.GetAll)
	nilai.Get("/:id", handler.GetByID)
	nilai.Get("/mahasiswa/:mahasiswa_id", handler.GetByMahasiswaID)
	nilai.Post("/", middleware.RoleMiddleware("admin"), handler.Create)
	nilai.Put("/:id", middleware.RoleMiddleware("admin"), handler.Update)
	nilai.Delete("/:id", middleware.RoleMiddleware("admin"), handler.Delete)
}
