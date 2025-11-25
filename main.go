package main

import (
	"UASBE/app/repository"
	"UASBE/app/service"
	"UASBE/database"
	"UASBE/route"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Connect to database
	database.Connect()

	// Auto migrate
	database.Migrate()

	// Initialize repositories
	userRepo := repository.NewUserRepository(database.DB)
	mahasiswaRepo := repository.NewMahasiswaRepository(database.DB)
	matakuliahRepo := repository.NewMataKuliahRepository(database.DB)
	nilaiRepo := repository.NewNilaiRepository(database.DB)

	// Initialize services
	authService := service.NewAuthService(userRepo)
	mahasiswaService := service.NewMahasiswaService(mahasiswaRepo, userRepo)
	matakuliahService := service.NewMataKuliahService(matakuliahRepo)
	nilaiService := service.NewNilaiService(nilaiRepo, mahasiswaRepo, matakuliahRepo)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName: "UAS Mahasiswa API v1.0",
	})

	// Middleware
	app.Use(cors.New())
	app.Use(logger.New())

	// Routes
	route.SetupAuthRoutes(app, authService)
	route.SetupMahasiswaRoutes(app, mahasiswaService)
	route.SetupMataKuliahRoutes(app, matakuliahService)
	route.SetupNilaiRoutes(app, nilaiService)

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "Server is running",
		})
	})

	// Start server
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server running on port %s", port)
	log.Fatal(app.Listen(":" + port))
}
