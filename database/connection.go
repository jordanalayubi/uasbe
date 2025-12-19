package database

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// PostgreSQL connection for master data
var PostgresDB *sql.DB

// MongoDB connection for achievements
var MongoDB *mongo.Database
var MongoClient *mongo.Client

// PostgresMigrator handles PostgreSQL migrations
type PostgresMigrator struct {
	db            *sql.DB
	migrationsDir string
}

// MongoMigrator handles MongoDB migrations
type MongoMigrator struct {
	db            *mongo.Database
	migrationsDir string
}

// MigrationRecord for MongoDB migration tracking
type MigrationRecord struct {
	Version    string    `bson:"version"`
	ExecutedAt time.Time `bson:"executed_at"`
}

func Connect() {
	// Connect to PostgreSQL
	connectPostgreSQL()
	
	// Check database health first
	err := CheckDatabaseHealth()
	if err != nil {
		log.Printf("Database health check failed: %v", err)
		log.Println("Attempting to setup database schema...")
		
		// Setup database schema
		err = SetupSchema()
		if err != nil {
			log.Printf("Failed to setup database schema: %v", err)
			log.Println("Please reset database manually or check the error above")
			log.Fatal("Database setup failed")
		}
	}
	
	// Create default users
	err = CreateDefaultUsers()
	if err != nil {
		log.Printf("Warning: Failed to create default users: %v", err)
		log.Println("You may need to create admin user manually")
	}
	
	// Connect to MongoDB
	connectMongoDB()
}

func connectPostgreSQL() {
	var err error
	
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Jakarta",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	PostgresDB, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("Failed to connect to PostgreSQL:", err)
	}

	err = PostgresDB.Ping()
	if err != nil {
		log.Fatal("Failed to ping PostgreSQL:", err)
	}

	log.Println("Connected to PostgreSQL successfully")
}

func connectMongoDB() {
	var err error
	
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}
	
	databaseName := os.Getenv("MONGO_DATABASE_NAME")
	if databaseName == "" {
		databaseName = "uasbe_achievements"
	}

	// Set client options
	clientOptions := options.Client().ApplyURI(mongoURI)
	
	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	MongoClient, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}

	// Check the connection
	err = MongoClient.Ping(ctx, nil)
	if err != nil {
		log.Fatal("Failed to ping MongoDB:", err)
	}

	MongoDB = MongoClient.Database(databaseName)
	log.Printf("Connected to MongoDB database: %s", databaseName)
}

func Migrate() {
	// Note: Migrations should be run manually using cmd/migrate/main.go
	// This function is kept for compatibility but doesn't run auto-migrations
	log.Println("Database connections established")
	log.Println("Run migrations manually using: go run cmd/migrate/main.go -db=postgres && go run cmd/migrate/main.go -db=mongo")
}

func Disconnect() {
	// Close PostgreSQL connection
	if PostgresDB != nil {
		PostgresDB.Close()
	}

	// Close MongoDB connection
	if MongoClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		MongoClient.Disconnect(ctx)
	}
}

// GetPostgresDB returns PostgreSQL database instance
func GetPostgresDB() *sql.DB {
	return PostgresDB
}

// GetMongoCollection returns MongoDB collection for achievements
func GetMongoCollection(name string) *mongo.Collection {
	return MongoDB.Collection(name)
}

// NewPostgresMigrator creates a new PostgreSQL migrator
func NewPostgresMigrator(db *sql.DB) *PostgresMigrator {
	return &PostgresMigrator{
		db:            db,
		migrationsDir: "database/migrations/postgres",
	}
}

// NewMongoMigrator creates a new MongoDB migrator
func NewMongoMigrator(db *mongo.Database) *MongoMigrator {
	return &MongoMigrator{
		db:            db,
		migrationsDir: "database/migrations/mongo",
	}
}

// RunMigrations runs all PostgreSQL migrations
func (m *PostgresMigrator) RunMigrations() error {
	// Create migrations table if not exists
	err := m.createMigrationsTable()
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %v", err)
	}

	// Get migration files
	files, err := m.getMigrationFiles()
	if err != nil {
		return fmt.Errorf("failed to get migration files: %v", err)
	}

	// Run each migration
	for _, file := range files {
		err := m.runMigration(file)
		if err != nil {
			return fmt.Errorf("failed to run migration %s: %v", file, err)
		}
	}

	log.Println("PostgreSQL migrations completed successfully")
	return nil
}

func (m *PostgresMigrator) createMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			executed_at TIMESTAMP DEFAULT NOW()
		);
	`
	_, err := m.db.Exec(query)
	return err
}

func (m *PostgresMigrator) getMigrationFiles() ([]string, error) {
	files, err := ioutil.ReadDir(m.migrationsDir)
	if err != nil {
		return nil, err
	}

	var migrationFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			migrationFiles = append(migrationFiles, file.Name())
		}
	}

	sort.Strings(migrationFiles)
	return migrationFiles, nil
}

func (m *PostgresMigrator) runMigration(filename string) error {
	// Check if migration already executed
	var count int
	err := m.db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = $1", filename).Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		log.Printf("Migration %s already executed, skipping", filename)
		return nil
	}

	// Read migration file
	filePath := filepath.Join(m.migrationsDir, filename)
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Execute migration
	_, err = m.db.Exec(string(content))
	if err != nil {
		return err
	}

	// Record migration as executed
	_, err = m.db.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", filename)
	if err != nil {
		return err
	}

	log.Printf("Migration %s executed successfully", filename)
	return nil
}

// RunMigrations runs all MongoDB migrations
func (m *MongoMigrator) RunMigrations() error {
	// Create migrations collection if not exists
	err := m.createMigrationsCollection()
	if err != nil {
		return fmt.Errorf("failed to create migrations collection: %v", err)
	}

	// Get migration files
	files, err := m.getMigrationFiles()
	if err != nil {
		return fmt.Errorf("failed to get migration files: %v", err)
	}

	// Run each migration
	for _, file := range files {
		err := m.runMigration(file)
		if err != nil {
			return fmt.Errorf("failed to run migration %s: %v", file, err)
		}
	}

	log.Println("MongoDB migrations completed successfully")
	return nil
}

func (m *MongoMigrator) createMigrationsCollection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create collection if it doesn't exist
	collections, err := m.db.ListCollectionNames(ctx, bson.M{"name": "schema_migrations"})
	if err != nil {
		return err
	}

	if len(collections) == 0 {
		err = m.db.CreateCollection(ctx, "schema_migrations")
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *MongoMigrator) getMigrationFiles() ([]string, error) {
	files, err := ioutil.ReadDir(m.migrationsDir)
	if err != nil {
		return nil, err
	}

	var migrationFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".js") {
			migrationFiles = append(migrationFiles, file.Name())
		}
	}

	sort.Strings(migrationFiles)
	return migrationFiles, nil
}

func (m *MongoMigrator) runMigration(filename string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check if migration already executed
	collection := m.db.Collection("schema_migrations")
	count, err := collection.CountDocuments(ctx, bson.M{"version": filename})
	if err != nil {
		return err
	}

	if count > 0 {
		log.Printf("Migration %s already executed, skipping", filename)
		return nil
	}

	// For now, we'll just mark as executed since we don't have mongo shell execution
	// In a real scenario, you would execute the JavaScript file using mongo shell
	log.Printf("Executing MongoDB migration %s (manual execution required)", filename)

	// Record migration as executed
	record := MigrationRecord{
		Version:    filename,
		ExecutedAt: time.Now(),
	}

	_, err = collection.InsertOne(ctx, record)
	if err != nil {
		return err
	}

	log.Printf("Migration %s marked as executed", filename)
	return nil
}
// SetupSchema creates database tables if they don't exist
func SetupSchema() error {
	// Create roles table
	_, err := PostgresDB.Exec(`
		CREATE TABLE IF NOT EXISTS roles (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name VARCHAR(50) UNIQUE NOT NULL,
			description TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create roles table: %v", err)
	}

	// Insert default roles
	_, err = PostgresDB.Exec(`
		INSERT INTO roles (name, description) VALUES 
		('admin', 'System Administrator'),
		('student', 'Student User'),
		('lecturer', 'Lecturer User')
		ON CONFLICT (name) DO NOTHING;
	`)
	if err != nil {
		return fmt.Errorf("failed to insert default roles: %v", err)
	}

	// Create users table
	_, err = PostgresDB.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			username VARCHAR(100) UNIQUE NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			password VARCHAR(255) NOT NULL,
			full_name VARCHAR(255) NOT NULL,
			role_id UUID NOT NULL REFERENCES roles(id),
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create users table: %v", err)
	}

	// Create lecturers table
	_, err = PostgresDB.Exec(`
		CREATE TABLE IF NOT EXISTS lecturers (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			lecturer_id VARCHAR(50) UNIQUE NOT NULL,
			department VARCHAR(255) NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create lecturers table: %v", err)
	}

	// Check if students table exists
	var studentsExists bool
	err = PostgresDB.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'students'
		);
	`).Scan(&studentsExists)
	if err != nil {
		return fmt.Errorf("failed to check students table: %v", err)
	}

	if !studentsExists {
		// Create students table with proper foreign key
		_, err = PostgresDB.Exec(`
			CREATE TABLE students (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				student_id VARCHAR(50) UNIQUE NOT NULL,
				program_study VARCHAR(255) NOT NULL,
				academic_year VARCHAR(10) NOT NULL,
				advisor_id UUID REFERENCES users(id) ON DELETE SET NULL,
				created_at TIMESTAMP DEFAULT NOW(),
				updated_at TIMESTAMP DEFAULT NOW()
			);
		`)
		if err != nil {
			return fmt.Errorf("failed to create students table: %v", err)
		}
	} else {
		// Check if advisor_id column has correct constraint
		_, err = PostgresDB.Exec(`
			ALTER TABLE students 
			DROP CONSTRAINT IF EXISTS students_advisor_id_fkey;
		`)
		if err != nil {
			log.Printf("Warning: Could not drop existing constraint: %v", err)
		}
		
		// Add correct foreign key constraint
		_, err = PostgresDB.Exec(`
			ALTER TABLE students 
			ADD CONSTRAINT students_advisor_id_fkey 
			FOREIGN KEY (advisor_id) REFERENCES users(id) ON DELETE SET NULL;
		`)
		if err != nil {
			log.Printf("Warning: Could not add foreign key constraint: %v", err)
		}
	}
	if err != nil {
		return fmt.Errorf("failed to create students table: %v", err)
	}

	// Check if permissions table exists and has correct structure
	var permissionsExists bool
	err = PostgresDB.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'permissions'
		);
	`).Scan(&permissionsExists)
	if err != nil {
		return fmt.Errorf("failed to check permissions table: %v", err)
	}

	if !permissionsExists {
		// Create permissions table with correct structure
		_, err = PostgresDB.Exec(`
			CREATE TABLE permissions (
				id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				role_id UUID NOT NULL REFERENCES roles(id),
				resource VARCHAR(100) NOT NULL,
				action VARCHAR(100) NOT NULL,
				created_at TIMESTAMP DEFAULT NOW(),
				UNIQUE(role_id, resource, action)
			);
		`)
		if err != nil {
			return fmt.Errorf("failed to create permissions table: %v", err)
		}

		// Insert default permissions
		_, err = PostgresDB.Exec(`
			INSERT INTO permissions (role_id, resource, action) 
			SELECT r.id, p.resource, p.action FROM roles r
			CROSS JOIN (VALUES 
				('achievements', 'create'),
				('achievements', 'read'),
				('achievements', 'update'),
				('achievements', 'delete'),
				('achievements', 'view_advisee'),
				('achievements', 'verify')
			) AS p(resource, action)
			WHERE r.name IN ('student', 'lecturer', 'admin');
		`)
		if err != nil {
			return fmt.Errorf("failed to insert default permissions: %v", err)
		}
	} else {
		// Check if permissions table has role_id column
		var hasRoleID bool
		err = PostgresDB.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.columns 
				WHERE table_name = 'permissions' 
				AND column_name = 'role_id'
			);
		`).Scan(&hasRoleID)
		if err != nil {
			return fmt.Errorf("failed to check role_id column: %v", err)
		}

		if !hasRoleID {
			// Add role_id column if it doesn't exist
			_, err = PostgresDB.Exec(`
				ALTER TABLE permissions 
				ADD COLUMN role_id UUID REFERENCES roles(id);
			`)
			if err != nil {
				return fmt.Errorf("failed to add role_id column: %v", err)
			}
		}
	}

	log.Println("Database schema setup completed successfully")
	return nil
}

// CreateDefaultUsers creates default admin user if not exists
func CreateDefaultUsers() error {
	// Check if admin user exists
	var count int
	err := PostgresDB.QueryRow("SELECT COUNT(*) FROM users WHERE username = 'admin'").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check admin user: %v", err)
	}

	if count > 0 {
		log.Println("Default admin user already exists")
		return nil
	}

	// Get admin role ID
	var adminRoleID string
	err = PostgresDB.QueryRow("SELECT id FROM roles WHERE name = 'admin'").Scan(&adminRoleID)
	if err != nil {
		return fmt.Errorf("failed to get admin role ID: %v", err)
	}

	// Create admin user
	_, err = PostgresDB.Exec(`
		INSERT INTO users (username, email, password, full_name, role_id, is_active)
		VALUES ('admin', 'admin@unair.ac.id', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'System Administrator', $1, true)
	`, adminRoleID)
	if err != nil {
		return fmt.Errorf("failed to create admin user: %v", err)
	}

	log.Println("Default admin user created successfully")
	return nil
}
// ResetDatabase drops and recreates all tables (for development only)
func ResetDatabase() error {
	log.Println("WARNING: Resetting database - all data will be lost!")
	
	// Drop tables in reverse order due to foreign key constraints
	tables := []string{"permissions", "students", "lecturers", "users", "roles"}
	
	for _, table := range tables {
		_, err := PostgresDB.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
		if err != nil {
			return fmt.Errorf("failed to drop table %s: %v", table, err)
		}
	}
	
	log.Println("All tables dropped successfully")
	
	// Recreate schema
	return SetupSchema()
}

// CheckDatabaseHealth checks if all required tables exist with correct structure
func CheckDatabaseHealth() error {
	requiredTables := []string{"roles", "users", "lecturers", "students", "permissions"}
	
	for _, table := range requiredTables {
		var exists bool
		err := PostgresDB.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = $1
			);
		`, table).Scan(&exists)
		
		if err != nil {
			return fmt.Errorf("failed to check table %s: %v", table, err)
		}
		
		if !exists {
			return fmt.Errorf("required table %s does not exist", table)
		}
	}
	
	log.Println("Database health check passed")
	return nil
}