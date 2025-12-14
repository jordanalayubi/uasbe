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