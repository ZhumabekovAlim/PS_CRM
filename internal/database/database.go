package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq" // PostgreSQL driver
)

var DB *sql.DB

// Config holds database configuration
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// InitDB initializes the database connection
func InitDB(cfg Config) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Error opening database: %q", err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatalf("Error connecting to database: %q", err)
	}

	fmt.Println("Successfully connected to the database!")

	// Optional: Run migrations or table creation scripts here
	// err = applySchema(DB) // Schema already applied manually, commenting out to prevent exit
    // if err != nil {
    //     log.Fatalf("Error applying database schema: %q", err)
    // }
}

// applySchema reads and executes the db_schema.sql file
func applySchema(db *sql.DB) error {
    schemaFile := "/home/ubuntu/final_project/ps_club_backend/db_schema.sql" // Path to your schema file
    content, err := os.ReadFile(schemaFile)
    if err != nil {
        return fmt.Errorf("could not read schema file %s: %w", schemaFile, err)
    }

    _, err = db.Exec(string(content))
    if err != nil {
        return fmt.Errorf("could not execute schema script: %w", err)
    }
    fmt.Println("Database schema applied successfully!")
    return nil
}

// GetDB returns the database connection pool
func GetDB() *sql.DB {
	return DB
}

