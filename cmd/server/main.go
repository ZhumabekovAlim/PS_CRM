package main

import (
	"log"
	"net/http"

	"ps_club_backend/internal/database"
	"ps_club_backend/internal/handlers"
	"ps_club_backend/internal/middleware"
	"ps_club_backend/pkg/utils" // Import utils for logger

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize Logger
	utils.InitLogger() // Initialize zerolog

	// Initialize Database
	dbCfg := database.Config{
		Host:     "localhost",
		Port:     "5432",
		User:     "ps_club_user",
		Password: "ps_club_password",
		DBName:   "ps_club_crm_db",
		SSLMode:  "disable",
	}
	database.InitDB(dbCfg)

	router := gin.Default()

	// Add GinLogger middleware for request logging
	router.Use(utils.GinLogger())

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000", "http://localhost:3001"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	config.AllowCredentials = true
	router.Use(cors.New(config))

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	apiV1 := router.Group("/api/v1")
	{
		// Auth routes (public)
		authRoutes := apiV1.Group("/auth")
		{
			authRoutes.POST("/register", handlers.RegisterUser)
			authRoutes.POST("/login", handlers.LoginUser)
			authRoutes.POST("/refresh-token", handlers.RefreshToken)

			authRequiredRoutes := authRoutes.Group("")
			authRequiredRoutes.Use(middleware.AuthMiddleware())
			{
				authRequiredRoutes.POST("/logout", handlers.LogoutUser)
				authRequiredRoutes.GET("/me", handlers.GetCurrentUser)
			}
		}

		authenticated := apiV1.Group("")
		authenticated.Use(middleware.AuthMiddleware())
		{
			pricelistCategoryRoutes := authenticated.Group("/pricelist-categories")
			pricelistCategoryRoutes.Use(middleware.RoleAuthMiddleware("Admin", "Staff"))
			{
				pricelistCategoryRoutes.POST("", handlers.CreatePricelistCategory)
				pricelistCategoryRoutes.GET("", handlers.GetPricelistCategories)
				pricelistCategoryRoutes.GET("/:id", handlers.GetPricelistCategoryByID)
				pricelistCategoryRoutes.PUT("/:id", handlers.UpdatePricelistCategory)
				pricelistCategoryRoutes.DELETE("/:id", handlers.DeletePricelistCategory)
			}

			pricelistItemRoutes := authenticated.Group("/pricelist-items")
			pricelistItemRoutes.Use(middleware.RoleAuthMiddleware("Admin", "Staff"))
			{
				pricelistItemRoutes.POST("", handlers.CreatePricelistItem)
				pricelistItemRoutes.GET("", handlers.GetPricelistItems)
				pricelistItemRoutes.GET("/:id", handlers.GetPricelistItemByID)
				pricelistItemRoutes.PUT("/:id", handlers.UpdatePricelistItem)
				pricelistItemRoutes.DELETE("/:id", handlers.DeletePricelistItem)
			}

			barItemRoutes := authenticated.Group("/bar-items")
			barItemRoutes.Use(middleware.RoleAuthMiddleware("Admin", "Staff"))
			{
				barItemRoutes.POST("", handlers.CreateBarItem)
				barItemRoutes.GET("", handlers.GetBarItems)
				barItemRoutes.GET("/:id", handlers.GetBarItemByID)
				barItemRoutes.PUT("/:id", handlers.UpdateBarItem)
				barItemRoutes.DELETE("/:id", handlers.DeleteBarItem)
			}

			hookahItemRoutes := authenticated.Group("/hookah-items")
			hookahItemRoutes.Use(middleware.RoleAuthMiddleware("Admin", "Staff"))
			{
				hookahItemRoutes.POST("", handlers.CreateHookahItem)
				hookahItemRoutes.GET("", handlers.GetHookahItems)
				hookahItemRoutes.GET("/:id", handlers.GetHookahItemByID)
				hookahItemRoutes.PUT("/:id", handlers.UpdateHookahItem)
				hookahItemRoutes.DELETE("/:id", handlers.DeleteHookahItem)
			}

			clientRoutes := authenticated.Group("/clients")
			clientRoutes.Use(middleware.RoleAuthMiddleware("Admin", "Staff"))
			{
				clientRoutes.POST("", handlers.CreateClient)
				clientRoutes.GET("", handlers.GetClients)
				clientRoutes.GET("/:id", handlers.GetClientByID)
				clientRoutes.PUT("/:id", handlers.UpdateClient)
				clientRoutes.DELETE("/:id", handlers.DeleteClient)
			}

			staffWriteRoutes := authenticated.Group("/staff")
			staffWriteRoutes.Use(middleware.RoleAuthMiddleware("Admin"))
			{
				staffWriteRoutes.POST("", handlers.CreateStaffMember)
				staffWriteRoutes.PUT("/:id", handlers.UpdateStaffMember)
				staffWriteRoutes.DELETE("/:id", handlers.DeleteStaffMember)
			}
			authenticated.GET("/staff", middleware.RoleAuthMiddleware("Admin", "Staff"), handlers.GetStaffMembers)
			authenticated.GET("/staff/:id", middleware.RoleAuthMiddleware("Admin", "Staff"), handlers.GetStaffMemberByID)

			shiftRoutes := authenticated.Group("/shifts")
			shiftRoutes.Use(middleware.RoleAuthMiddleware("Admin", "Staff"))
			{
				shiftRoutes.POST("", handlers.CreateShift)
				shiftRoutes.GET("", handlers.GetShifts)
				shiftRoutes.GET("/:id", handlers.GetShiftByID)
				shiftRoutes.PUT("/:id", handlers.UpdateShift)
				shiftRoutes.DELETE("/:id", handlers.DeleteShift)
			}

			gameTableRoutes := authenticated.Group("/tables")
			gameTableRoutes.Use(middleware.RoleAuthMiddleware("Admin", "Staff"))
			{
				gameTableRoutes.POST("", handlers.CreateGameTable)
				gameTableRoutes.GET("", handlers.GetGameTables)
				gameTableRoutes.GET("/:id", handlers.GetGameTableByID)
				gameTableRoutes.PUT("/:id", handlers.UpdateGameTable)
				gameTableRoutes.DELETE("/:id", handlers.DeleteGameTable)
			}

			bookingRoutes := authenticated.Group("/bookings")
			bookingRoutes.Use(middleware.RoleAuthMiddleware("Admin", "Staff"))
			{
				bookingRoutes.POST("", handlers.CreateBooking)
				bookingRoutes.GET("", handlers.GetBookings)
				bookingRoutes.GET("/:id", handlers.GetBookingByID)
				bookingRoutes.PUT("/:id", handlers.UpdateBooking)
				bookingRoutes.DELETE("/:id", handlers.DeleteBooking)
			}

			orderRoutes := authenticated.Group("/orders")
			orderRoutes.Use(middleware.RoleAuthMiddleware("Admin", "Staff"))
			{
				orderRoutes.POST("", handlers.CreateOrder)
				orderRoutes.GET("", handlers.GetOrders)
				orderRoutes.GET("/:id", handlers.GetOrderByID)
				orderRoutes.PATCH("/:id/status", handlers.UpdateOrderStatus)
				orderRoutes.DELETE("/:id", handlers.DeleteOrder)
			}

			inventoryMovementRoutes := authenticated.Group("/inventory-movements")
			inventoryMovementRoutes.Use(middleware.RoleAuthMiddleware("Admin", "Staff"))
			{
				inventoryMovementRoutes.POST("", handlers.CreateInventoryMovement)
				inventoryMovementRoutes.GET("", handlers.GetInventoryMovements)
			}

			settingsRoutes := authenticated.Group("/settings")
			settingsRoutes.Use(middleware.RoleAuthMiddleware("Admin"))
			{
				settingsRoutes.GET("", handlers.GetApplicationSettings)
				settingsRoutes.POST("", handlers.CreateOrUpdateApplicationSetting)
				settingsRoutes.GET("/:key", handlers.GetApplicationSettingByKey)
				settingsRoutes.DELETE("/:key", handlers.DeleteApplicationSettingByKey)
			}

			reportRoutes := authenticated.Group("/reports")
			reportRoutes.Use(middleware.RoleAuthMiddleware("Admin", "Staff"))
			{
				reportRoutes.GET("/sales", handlers.GetSalesReports)
				reportRoutes.GET("/bookings", handlers.GetBookingReports)
				reportRoutes.GET("/inventory", handlers.GetInventoryReports)
			}

			dashboardRoutes := authenticated.Group("/dashboard")
			dashboardRoutes.Use(middleware.RoleAuthMiddleware("Admin", "Staff"))
			{
				dashboardRoutes.GET("/summary", handlers.GetDashboardSummary)
			}
		}
	}

	port := "8080"
	// Use utils.LogInfo for server startup messages
	utils.LogInfo("Server starting", map[string]interface{}{"port": port})
	utils.LogInfo("Frontend should be configured to make API calls", map[string]interface{}{"url": "http://localhost:" + port + "/api/v1"})

	if err := router.Run(":" + port); err != nil {
		// Use utils.LogError for fatal server errors
		utils.LogError(err, "Failed to start server")
		log.Fatalf("Failed to start server: %v", err) // Keep log.Fatalf for immediate exit on critical error
	}
}
