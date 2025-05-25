package router

import (
	"ps_club_backend/internal/handlers"
	"ps_club_backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// SetupAuthRoutes sets up the authentication routes.
func SetupAuthRoutes(apiGroup *gin.RouterGroup, authHandler *handlers.AuthHandler) {
	authRoutes := apiGroup.Group("/auth")
	{
		authRoutes.POST("/register", authHandler.RegisterUser)
		authRoutes.POST("/login", authHandler.LoginUser)
		authRoutes.POST("/refresh-token", authHandler.RefreshToken) // Will use the placeholder from AuthHandler

		authRequiredRoutes := authRoutes.Group("")
		authRequiredRoutes.Use(middleware.AuthMiddleware()) // Apply AuthMiddleware to this sub-group
		{
			authRequiredRoutes.POST("/logout", authHandler.LogoutUser)
			authRequiredRoutes.GET("/me", authHandler.GetCurrentUser)
		}
	}
}

// SetupOrderRoutes sets up the order routes.
func SetupOrderRoutes(authenticatedGroup *gin.RouterGroup, orderHandler *handlers.OrderHandler) {
	orderRoutes := authenticatedGroup.Group("/orders")
	orderRoutes.Use(middleware.RoleAuthMiddleware("Admin", "Staff"))
	{
		orderRoutes.POST("", orderHandler.CreateOrder)
		orderRoutes.GET("", orderHandler.GetOrders)
		orderRoutes.GET("/:id", orderHandler.GetOrderByID)
		orderRoutes.PATCH("/:id/status", orderHandler.UpdateOrderStatus)
		orderRoutes.DELETE("/:id", orderHandler.DeleteOrder)
	}
}

// SetupPricelistCategoryRoutes sets up the pricelist category routes.
func SetupPricelistCategoryRoutes(authenticatedGroup *gin.RouterGroup, pricelistHandler *handlers.PricelistHandler) {
	pricelistCategoryRoutes := authenticatedGroup.Group("/pricelist-categories")
	pricelistCategoryRoutes.Use(middleware.RoleAuthMiddleware("Admin", "Staff"))
	{
		pricelistCategoryRoutes.POST("", pricelistHandler.CreatePricelistCategory)
		pricelistCategoryRoutes.GET("", pricelistHandler.GetPricelistCategories)
		pricelistCategoryRoutes.GET("/:id", pricelistHandler.GetPricelistCategoryByID)
		pricelistCategoryRoutes.PUT("/:id", pricelistHandler.UpdatePricelistCategory)
		pricelistCategoryRoutes.DELETE("/:id", pricelistHandler.DeletePricelistCategory)
	}
}

// SetupPricelistItemRoutes sets up the pricelist item routes.
func SetupPricelistItemRoutes(authenticatedGroup *gin.RouterGroup, pricelistHandler *handlers.PricelistHandler) {
	pricelistItemRoutes := authenticatedGroup.Group("/pricelist-items")
	pricelistItemRoutes.Use(middleware.RoleAuthMiddleware("Admin", "Staff"))
	{
		pricelistItemRoutes.POST("", pricelistHandler.CreatePricelistItem)
		pricelistItemRoutes.GET("", pricelistHandler.GetPricelistItems)
		pricelistItemRoutes.GET("/:id", pricelistHandler.GetPricelistItemByID)
		pricelistItemRoutes.PUT("/:id", pricelistHandler.UpdatePricelistItem)
		pricelistItemRoutes.DELETE("/:id", pricelistHandler.DeletePricelistItem)
	}
}

// SetupBarItemRoutes sets up the bar item routes.
func SetupBarItemRoutes(authenticatedGroup *gin.RouterGroup /*, handler *handlers.BarItemHandler*/) {
	barItemRoutes := authenticatedGroup.Group("/bar-items")
	barItemRoutes.Use(middleware.RoleAuthMiddleware("Admin", "Staff"))
	{
		barItemRoutes.POST("", handlers.CreateBarItem)
		barItemRoutes.GET("", handlers.GetBarItems)
		barItemRoutes.GET("/:id", handlers.GetBarItemByID)
		barItemRoutes.PUT("/:id", handlers.UpdateBarItem)
		barItemRoutes.DELETE("/:id", handlers.DeleteBarItem)
	}
}

// SetupHookahItemRoutes sets up the hookah item routes.
func SetupHookahItemRoutes(authenticatedGroup *gin.RouterGroup /*, handler *handlers.HookahItemHandler*/) {
	hookahItemRoutes := authenticatedGroup.Group("/hookah-items")
	hookahItemRoutes.Use(middleware.RoleAuthMiddleware("Admin", "Staff"))
	{
		hookahItemRoutes.POST("", handlers.CreateHookahItem)
		hookahItemRoutes.GET("", handlers.GetHookahItems)
		hookahItemRoutes.GET("/:id", handlers.GetHookahItemByID)
		hookahItemRoutes.PUT("/:id", handlers.UpdateHookahItem)
		hookahItemRoutes.DELETE("/:id", handlers.DeleteHookahItem)
	}
}

// SetupClientRoutes sets up the client routes.
func SetupClientRoutes(authenticatedGroup *gin.RouterGroup, clientHandler *handlers.ClientHandler) {
	clientRoutes := authenticatedGroup.Group("/clients")
	clientRoutes.Use(middleware.RoleAuthMiddleware("Admin", "Staff"))
	{
		clientRoutes.POST("", clientHandler.CreateClient)
		clientRoutes.GET("", clientHandler.GetClients)
		clientRoutes.GET("/:id", clientHandler.GetClientByID)
		clientRoutes.PUT("/:id", clientHandler.UpdateClient)
		clientRoutes.DELETE("/:id", clientHandler.DeleteClient)
	}
}

// SetupStaffRoutes sets up the staff routes.
// Note: RoleAuthMiddleware is applied specifically for write and read operations.
func SetupStaffRoutes(authenticatedGroup *gin.RouterGroup, staffHandler *handlers.StaffHandler) {
	staffWriteRoutes := authenticatedGroup.Group("/staff")
	staffWriteRoutes.Use(middleware.RoleAuthMiddleware("Admin")) // Admin only for POST, PUT, DELETE
	{
		staffWriteRoutes.POST("", staffHandler.CreateStaffMember)
		staffWriteRoutes.PUT("/:id", staffHandler.UpdateStaffMember)
		staffWriteRoutes.DELETE("/:id", staffHandler.DeleteStaffMember)
	}

	// GET routes with Admin or Staff roles
	authenticatedGroup.GET("/staff", middleware.RoleAuthMiddleware("Admin", "Staff"), staffHandler.GetStaffMembers)
	authenticatedGroup.GET("/staff/:id", middleware.RoleAuthMiddleware("Admin", "Staff"), staffHandler.GetStaffMemberByID)
}

// SetupShiftRoutes sets up the shift routes.
func SetupShiftRoutes(authenticatedGroup *gin.RouterGroup, staffHandler *handlers.StaffHandler) {
	shiftRoutes := authenticatedGroup.Group("/shifts")
	shiftRoutes.Use(middleware.RoleAuthMiddleware("Admin", "Staff"))
	{
		shiftRoutes.POST("", staffHandler.CreateShift)
		shiftRoutes.GET("", staffHandler.GetShifts)
		shiftRoutes.GET("/:id", staffHandler.GetShiftByID)
		shiftRoutes.PUT("/:id", staffHandler.UpdateShift)
		shiftRoutes.DELETE("/:id", staffHandler.DeleteShift)
	}
}

// SetupGameTableRoutes sets up the game table routes.
func SetupGameTableRoutes(authenticatedGroup *gin.RouterGroup /*, handler *handlers.GameTableHandler*/) {
	gameTableRoutes := authenticatedGroup.Group("/tables")
	gameTableRoutes.Use(middleware.RoleAuthMiddleware("Admin", "Staff"))
	{
		gameTableRoutes.POST("", handlers.CreateGameTable)
		gameTableRoutes.GET("", handlers.GetGameTables)
		gameTableRoutes.GET("/:id", handlers.GetGameTableByID)
		gameTableRoutes.PUT("/:id", handlers.UpdateGameTable)
		gameTableRoutes.DELETE("/:id", handlers.DeleteGameTable)
	}
}

// SetupBookingRoutes sets up the booking routes.
func SetupBookingRoutes(authenticatedGroup *gin.RouterGroup, bookingHandler *handlers.BookingHandler) {
	bookingRoutes := authenticatedGroup.Group("/bookings")
	bookingRoutes.Use(middleware.RoleAuthMiddleware("Admin", "Staff"))
	{
		bookingRoutes.POST("", bookingHandler.CreateBooking)
		bookingRoutes.GET("", bookingHandler.GetBookings)
		bookingRoutes.GET("/:id", bookingHandler.GetBookingByID)
		bookingRoutes.PUT("/:id", bookingHandler.UpdateBooking)
		bookingRoutes.DELETE("/:id", bookingHandler.DeleteBooking)
		bookingRoutes.PATCH("/:id/cancel", bookingHandler.CancelBooking)
		bookingRoutes.PATCH("/:id/complete", bookingHandler.CompleteBooking)
	}
}

// SetupInventoryMovementRoutes sets up the inventory movement routes.
func SetupInventoryMovementRoutes(authenticatedGroup *gin.RouterGroup, inventoryMvHandler *handlers.InventoryMovementHandler) {
	inventoryMovementRoutes := authenticatedGroup.Group("/inventory-movements")
	inventoryMovementRoutes.Use(middleware.RoleAuthMiddleware("Admin", "Staff"))
	{
		inventoryMovementRoutes.POST("", inventoryMvHandler.CreateInventoryMovement)
		inventoryMovementRoutes.GET("", inventoryMvHandler.GetInventoryMovements)
	}
}

// SetupSettingsRoutes sets up the application settings routes.
func SetupSettingsRoutes(authenticatedGroup *gin.RouterGroup /*, handler *handlers.SettingsHandler*/) {
	settingsRoutes := authenticatedGroup.Group("/settings")
	settingsRoutes.Use(middleware.RoleAuthMiddleware("Admin"))
	{
		settingsRoutes.GET("", handlers.GetApplicationSettings)
		settingsRoutes.POST("", handlers.CreateOrUpdateApplicationSetting)
		settingsRoutes.GET("/:key", handlers.GetApplicationSettingByKey)
		settingsRoutes.DELETE("/:key", handlers.DeleteApplicationSettingByKey)
	}
}

// SetupReportRoutes sets up the report routes.
func SetupReportRoutes(authenticatedGroup *gin.RouterGroup /*, handler *handlers.ReportHandler*/) {
	reportRoutes := authenticatedGroup.Group("/reports")
	reportRoutes.Use(middleware.RoleAuthMiddleware("Admin", "Staff"))
	{
		reportRoutes.GET("/sales", handlers.GetSalesReports)
		reportRoutes.GET("/bookings", handlers.GetBookingReports)
		reportRoutes.GET("/inventory", handlers.GetInventoryReports)
	}
}

// SetupDashboardRoutes sets up the dashboard routes.
func SetupDashboardRoutes(authenticatedGroup *gin.RouterGroup /*, handler *handlers.DashboardHandler*/) {
	dashboardRoutes := authenticatedGroup.Group("/dashboard")
	dashboardRoutes.Use(middleware.RoleAuthMiddleware("Admin", "Staff"))
	{
		dashboardRoutes.GET("/summary", handlers.GetDashboardSummary)
	}
}
