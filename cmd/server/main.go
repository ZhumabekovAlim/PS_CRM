package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"log"
	"net/http"
	"os"
	"strings"

	"ps_club_backend/internal/database"
	// "ps_club_backend/internal/handlers" // No longer directly used for route setup here
	// "ps_club_backend/internal/middleware" // No longer directly used for route setup here
	// "ps_club_backend/internal/services" // No longer directly used for route setup here
	"ps_club_backend/internal/router" // Added for router.Setup
	"ps_club_backend/pkg/utils"       // Import utils for logger

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize Logger
	utils.InitLogger() // Initialize zerolog

	// Load database configuration from environment variables
	dbHost := utils.Getenv("DB_HOST", "localhost")
	dbPort := utils.Getenv("DB_PORT", "5432")
	dbUser := utils.Getenv("DB_USER", "ps_club_user")
	dbPassword := utils.Getenv("DB_PASSWORD", "ps_club_password")
	dbName := utils.Getenv("DB_NAME", "ps_club_crm_db")
	dbSSLMode := utils.Getenv("DB_SSLMODE", "disable")
	dbSchemaPath := utils.Getenv("DB_SCHEMA_PATH", "") // Default to empty string

	// Initialize Database
	database.InitDB(dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode, dbSchemaPath)
	utils.LogInfo("Database initialized", map[string]interface{}{"configured_from_env": true})

	router := gin.Default()

	// Add GinLogger middleware for request logging
	router.Use(utils.GinLogger())

	// CORS configuration
	corsAllowedOriginsEnv := os.Getenv("CORS_ALLOWED_ORIGINS")
	var allowedOrigins []string
	if corsAllowedOriginsEnv != "" {
		allowedOrigins = strings.Split(corsAllowedOriginsEnv, ",")
	} else {
		allowedOrigins = []string{"http://localhost:3000", "http://localhost:3001"} // Default origins
	}

	config := cors.DefaultConfig()
	config.AllowOrigins = allowedOrigins
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	config.AllowCredentials = true
	router.Use(cors.New(config))

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	// Setup all application routes
	dbConn := database.GetDB()
	router_pkg.Setup(router, dbConn)

	// Server port configuration
	port := utils.Getenv("PORT", "8080") // Default to 8080 if not set
	utils.LogInfo("Server starting", map[string]interface{}{"port": port, "configured_from_env": true})
	utils.LogInfo("Frontend should be configured to make API calls", map[string]interface{}{"url": "http://localhost:" + port + "/api/v1"})

	if err := router.Run(":" + port); err != nil {
		utils.LogError(err, "Failed to start server")
		log.Fatalf("Failed to start server: %v", err)
	}
}
