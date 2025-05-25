package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq" // PostgreSQL driver
)

var DB *sql.DB

// InitDB initializes the database connection
func InitDB(host, port, user, password, dbname, sslmode, dbSchemaPath string) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)

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
	// err = applySchema(DB, dbSchemaPath) // Schema already applied manually, commenting out to prevent exit
	// if err != nil {
	//     log.Fatalf("Error applying database schema: %q", err)
	// }
}

// applySchema reads and executes the db_schema.sql file
func applySchema(db *sql.DB, schemaPath string) error {
	if schemaPath == "" {
		log.Println("No schema path provided, skipping schema application.")
		return nil
	}
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("could not read schema file %s: %w", schemaPath, err)
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

