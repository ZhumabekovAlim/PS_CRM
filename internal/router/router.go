package router

import (
	"database/sql"
	"time" // Added for JWT expiration

	"ps_club_backend/internal/handlers"
	"ps_club_backend/internal/middleware"
	"ps_club_backend/internal/repositories" // Added for AuthRepository
	"ps_club_backend/internal/services"
	"github.com/gin-gonic/gin"
)

// Setup initializes the routing for the application.
func Setup(engine *gin.Engine, db *sql.DB) {
	// Initialize Repositories
	authRepo := repositories.NewAuthRepository(db)
	pricelistRepo := repositories.NewPricelistRepository(db)
	inventoryMvRepo := repositories.NewInventoryMovementRepository(db)
	orderRepo := repositories.NewOrderRepository(db)
	clientRepo := repositories.NewClientRepository(db)
	staffRepo := repositories.NewStaffRepository(db)
	bookingRepo := repositories.NewBookingRepository(db) // Added BookingRepository
	// TODO: Initialize other repositories here

	// Initialize Services
	// For JWT Secret: In a real app, load from config/env. Using a placeholder for now.
	jwtSecret := "your-very-secure-jwt-secret-replace-it" // Replace with actual secret management
	jwtExpiration := time.Hour * 72                       // Example: 72 hours

	authService := services.NewAuthService(authRepo, db, jwtSecret, jwtExpiration)
	pricelistService := services.NewPricelistService(pricelistRepo, db)
	inventoryMvService := services.NewInventoryMovementService(inventoryMvRepo, pricelistRepo, db)
	orderService := services.NewOrderService(orderRepo, pricelistRepo, inventoryMvRepo, db)
	clientService := services.NewClientService(clientRepo, db)
	staffService := services.NewStaffService(staffRepo, authRepo, db)
	bookingService := services.NewBookingService(bookingRepo, clientRepo, staffRepo, db) // Added BookingService
	// TODO: Initialize other services here as they are created

	// Initialize Handlers
	authHandler := handlers.NewAuthHandler(authService)
	pricelistHandler := handlers.NewPricelistHandler(pricelistService)
	inventoryMvHandler := handlers.NewInventoryMovementHandler(inventoryMvService)
	orderHandler := handlers.NewOrderHandler(orderService)
	clientHandler := handlers.NewClientHandler(clientService)
	staffHandler := handlers.NewStaffHandler(staffService)
	bookingHandler := handlers.NewBookingHandler(bookingService) // Added BookingHandler
	// TODO: Initialize other handlers here as they are refactored

	apiV1 := engine.Group("/api/v1")

	// Setup public authentication routes
	// Note: Original SetupAuthRoutes(apiV1, authHandler) might be split if some auth routes are public
	// and some (like /me, /logout) are authenticated. For this example, assuming all auth routes are passed authHandler.
	// If /register and /login are public and don't need middleware.AuthMiddleware() applied to their group:
	// publicAuthRoutes := apiV1.Group("/auth")
	// SetupPublicAuthRoutes(publicAuthRoutes, authHandler) // e.g. for /register, /login

	// Setup authenticated routes
	authenticated := apiV1.Group("")
	authenticated.Use(middleware.AuthMiddleware())
	{
		// Assuming /auth/me, /auth/logout are authenticated:
		SetupAuthenticatedAuthRoutes(authenticated.Group("/auth"), authHandler) // Grouping auth routes under /auth path
		
		SetupOrderRoutes(authenticated, orderHandler)
		SetupPricelistCategoryRoutes(authenticated, pricelistHandler)
		SetupPricelistItemRoutes(authenticated, pricelistHandler)
		SetupInventoryMovementRoutes(authenticated, inventoryMvHandler)
		SetupClientRoutes(authenticated, clientHandler)
		SetupStaffRoutes(authenticated, staffHandler)
		SetupShiftRoutes(authenticated, staffHandler)
		SetupBookingRoutes(authenticated, bookingHandler) // Updated to pass bookingHandler

		// Placeholder for other route setups, assuming they are also authenticated
		SetupBarItemRoutes(authenticated)           // Still uses old direct handlers
		SetupHookahItemRoutes(authenticated)        // Still uses old direct handlers
		SetupGameTableRoutes(authenticated)         // Pass handler when available
		SetupSettingsRoutes(authenticated)          // Pass handler when available
		SetupReportRoutes(authenticated)            // Pass handler when available
		SetupDashboardRoutes(authenticated)         // Pass handler when available
	}

	// If /auth/register and /auth/login are truly public (no AuthMiddleware):
	// Re-define SetupAuthRoutes to split public and private, or have two functions.
	// Example:
	authPublicRoutes := apiV1.Group("/auth")
	SetupPublicAuthRoutes(authPublicRoutes, authHandler) // For /register, /login
}

// Helper for clarity if splitting auth routes (example, actual split logic is in SetupAuthRoutes)
func SetupPublicAuthRoutes(group *gin.RouterGroup, authHandler *handlers.AuthHandler) {
    group.POST("/register", authHandler.RegisterUser)
    group.POST("/login", authHandler.LoginUser)
    group.POST("/refresh-token", authHandler.RefreshToken)
}

func SetupAuthenticatedAuthRoutes(group *gin.RouterGroup, authHandler *handlers.AuthHandler) {
    group.POST("/logout", authHandler.LogoutUser)
    group.GET("/me", authHandler.GetCurrentUser)
}
