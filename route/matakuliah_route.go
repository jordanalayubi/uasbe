package route

import (
	"UASBE/app/model"
	"UASBE/app/service"
	"UASBE/middleware"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type MataKuliahHandler struct {
	matakuliahService service.MataKuliahService
}

func NewMataKuliahHandler(matakuliahService service.MataKuliahService) *MataKuliahHandler {
	return &MataKuliahHandler{matakuliahService: matakuliahService}
}

func (h *MataKuliahHandler) Create(c *fiber.Ctx) error {
	var matakuliah model.MataKuliah
	if err := c.BodyParser(&matakuliah); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Message: "Invalid request body",
		})
	}

	if err := h.matakuliahService.Create(&matakuliah); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(model.Response{
		Success: true,
		Message: "Mata Kuliah created successfully",
		Data:    matakuliah,
	})
}

func (h *MataKuliahHandler) GetAll(c *fiber.Ctx) error {
	matakuliah, err := h.matakuliahService.GetAll()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(model.Response{
			Success: false,
			Message: err.Error(),
		})
	}

	return c.JSON(model.Response{
		Success: true,
		Message: "Success",
		Data:    matakuliah,
	})
}

func (h *MataKuliahHandler) GetByID(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Message: "Invalid ID",
		})
	}

	matakuliah, err := h.matakuliahService.GetByID(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(model.Response{
			Success: false,
			Message: "Mata Kuliah not found",
		})
	}

	return c.JSON(model.Response{
		Success: true,
		Message: "Success",
		Data:    matakuliah,
	})
}

func (h *MataKuliahHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Message: "Invalid ID",
		})
	}

	var matakuliah model.MataKuliah
	if err := c.BodyParser(&matakuliah); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Message: "Invalid request body",
		})
	}

	if err := h.matakuliahService.Update(uint(id), &matakuliah); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Message: err.Error(),
		})
	}

	return c.JSON(model.Response{
		Success: true,
		Message: "Mata Kuliah updated successfully",
		Data:    matakuliah,
	})
}

func (h *MataKuliahHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Message: "Invalid ID",
		})
	}

	if err := h.matakuliahService.Delete(uint(id)); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(model.Response{
			Success: false,
			Message: err.Error(),
		})
	}

	return c.JSON(model.Response{
		Success: true,
		Message: "Mata Kuliah deleted successfully",
	})
}

func SetupMataKuliahRoutes(app *fiber.App, matakuliahService service.MataKuliahService) {
	handler := NewMataKuliahHandler(matakuliahService)
	
	matakuliah := app.Group("/api/matakuliah")
	matakuliah.Use(middleware.AuthMiddleware)
	
	matakuliah.Get("/", handler.GetAll)
	matakuliah.Get("/:id", handler.GetByID)
	matakuliah.Post("/", middleware.RoleMiddleware("admin"), handler.Create)
	matakuliah.Put("/:id", middleware.RoleMiddleware("admin"), handler.Update)
	matakuliah.Delete("/:id", middleware.RoleMiddleware("admin"), handler.Delete)
}
