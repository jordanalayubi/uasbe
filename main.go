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
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Connect to database
	database.Connect()



	// Get JWT secret from environment
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key-change-in-production"
		log.Println("Warning: Using default JWT secret. Please set JWT_SECRET in production.")
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository()
	studentRepo := repository.NewStudentRepository()
	lecturerRepo := repository.NewLecturerRepository()
	achievementRepo := repository.NewAchievementRepository()

	// Initialize services
	authService := service.NewAuthService(userRepo, studentRepo, lecturerRepo, jwtSecret)
	achievementService := service.NewAchievementService(achievementRepo, studentRepo, lecturerRepo)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName: "UAS Achievement System API v1.0",
		BodyLimit: 10 * 1024 * 1024, // 10MB for file uploads
	})

	// Middleware
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE",
	}))
	app.Use(logger.New())

	// Static file serving for uploads
	app.Static("/uploads", "./uploads")

	// Routes
	route.SetupAuthRoutes(app, authService)
	route.SetupAchievementRoutes(app, achievementService, authService)
	route.SetupAdminRoutes(app, authService)
	route.SetupTestRoutes(app, authService)

	// Health check
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "UAS Achievement System API is running",
			"version": "1.0",
		})
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "Server is running",
		})
	})

	// Graceful shutdown
	defer database.Disconnect()

	// Start server
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Server running on port %s", port)
	log.Fatal(app.Listen(":" + port))
}
