package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"ps_club_backend/internal/services"
	"ps_club_backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

// InventoryMovementHandler holds the inventory movement service.
type InventoryMovementHandler struct {
	inventoryMvService services.InventoryMovementService
}

// NewInventoryMovementHandler creates a new InventoryMovementHandler.
func NewInventoryMovementHandler(ims services.InventoryMovementService) *InventoryMovementHandler {
	return &InventoryMovementHandler{inventoryMvService: ims}
}

// CreateInventoryMovement handles the creation of a new inventory movement.
func (h *InventoryMovementHandler) CreateInventoryMovement(c *gin.Context) {
	var req services.CreateInventoryMovementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.LogError(err, "CreateInventoryMovement: Failed to bind JSON")
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid request payload: "+err.Error(), err.Error()))
		return
	}

	// Extract authenticated staff ID from context
	authStaffIDRaw, exists := c.Get("userID") // Assuming "userID" is set by AuthMiddleware
	if !exists {
		utils.LogError(errors.New("userID not found in context"), "CreateInventoryMovement: userID not in context")
		utils.RespondWithError(c, utils.NewAPIError(http.StatusUnauthorized, utils.ErrCodeUnauthorized, "User not authenticated.", "Missing user ID in context"))
		return
	}
	authStaffID, ok := authStaffIDRaw.(int64)
	if !ok {
		utils.LogError(errors.New("userID is not of type int64"), "CreateInventoryMovement: userID type assertion failed")
		utils.RespondWithError(c, utils.NewAPIError(http.StatusUnauthorized, utils.ErrCodeUnauthorized, "User ID format incorrect.", "Invalid user ID format in context"))
		return
	}

	movement, err := h.inventoryMvService.CreateMovement(req, authStaffID)
	if err != nil {
		utils.LogError(err, "CreateInventoryMovement: Error from inventoryMvService.CreateMovement")
		if errors.Is(err, services.ErrInvalidMovementType) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid movement type provided.", err.Error()))
		} else if errors.Is(err, services.ErrMovementItemNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusNotFound, utils.ErrCodeNotFound, "Pricelist item for movement not found.", err.Error()))
		} else if errors.Is(err, services.ErrMovementItemNotTracked) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeBadRequest, "Pricelist item does not track stock.", err.Error()))
		} else if errors.Is(err, services.ErrValidation) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Validation failed: "+err.Error(), err.Error()))
		} else if errors.Is(err, services.ErrMovementCreationFailed) || errors.Is(err, services.ErrStockUpdateFailed) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to process inventory movement due to an internal issue.", err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to create inventory movement.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusCreated, movement)
}

// GetInventoryMovements handles fetching all inventory movements with filters and pagination.
func (h *InventoryMovementHandler) GetInventoryMovements(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page <= 0 { page = 1 }
	if pageSize <= 0 { pageSize = 10 }

	var itemID *int64
	if itemIDStr := c.Query("item_id"); itemIDStr != "" {
		id, err := strconv.ParseInt(itemIDStr, 10, 64)
		if err == nil {
			itemID = &id
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid item_id format.", err.Error()))
			return
		}
	}

	var staffID *int64
	if staffIDStr := c.Query("staff_id"); staffIDStr != "" {
		id, err := strconv.ParseInt(staffIDStr, 10, 64)
		if err == nil {
			staffID = &id
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid staff_id format.", err.Error()))
			return
		}
	}

	var movementType *string
	if movementTypeStr := c.Query("movement_type"); movementTypeStr != "" {
		movementType = &movementTypeStr
	}

	movements, totalCount, err := h.inventoryMvService.GetMovements(itemID, staffID, movementType, page, pageSize)
	if err != nil {
		utils.LogError(err, "GetInventoryMovements: Error from inventoryMvService.GetMovements")
		utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to fetch inventory movements.", "Internal error"))
		return
	}

	if movements == nil { // Ensure response is an empty list, not null
	    movements = []models.InventoryMovement{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  movements,
		"total": totalCount,
		"page": page,
		"page_size": pageSize,
	})
}

// Remove or comment out old standalone functions if they existed:
// func CreateInventoryMovement(c *gin.Context) { ... }
// func GetInventoryMovements(c *gin.Context) { ... }
