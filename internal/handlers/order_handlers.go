package handlers

import (
	"errors"
	"net/http"
	"strconv"
	// "time" // No longer directly used for business logic here

	// "ps_club_backend/internal/database" // No longer directly used
	"ps_club_backend/internal/models" // May still be used for response DTOs if not fully replaced by service DTOs
	"ps_club_backend/internal/services"
	"ps_club_backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

// OrderHandler holds the order service.
type OrderHandler struct {
	orderService services.OrderService
}

// NewOrderHandler creates a new OrderHandler.
func NewOrderHandler(os services.OrderService) *OrderHandler {
	return &OrderHandler{orderService: os}
}

// CreateOrder handles the creation of a new order with its items
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req services.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.LogError(err, "CreateOrder: Failed to bind JSON")
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid request payload: "+err.Error(), err.Error()))
		return
	}

	// Assuming StaffID might come from authenticated user context in a real app
	// For now, CreateOrderRequest requires it. If it's not in the request but from auth:
	// userID, exists := c.Get("userID") // Example: if using middleware to set userID
	// if !exists {
	// 	 utils.RespondWithError(c, utils.NewAPIError(http.StatusUnauthorized, utils.ErrCodeUnauthorized, "User not authenticated", "Missing user ID"))
	// 	 return
	// }
	// req.StaffID = userID.(int64) // Cast appropriately

	createdOrder, err := h.orderService.CreateOrder(req)
	if err != nil {
		utils.LogError(err, "CreateOrder: Error from orderService.CreateOrder")
		if errors.Is(err, services.ErrPricelistItemNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusNotFound, utils.ErrCodeNotFound, "One or more pricelist items not found or unavailable.", err.Error()))
		} else if errors.Is(err, services.ErrInsufficientStock) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusConflict, utils.ErrCodeConflict, "Insufficient stock for one or more items.", err.Error()))
		} else if errors.Is(err, services.ErrInvalidOrderStatus) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid order status provided.", err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to create order.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusCreated, createdOrder)
}

// GetOrders handles fetching all orders with filters
func (h *OrderHandler) GetOrders(c *gin.Context) {
	var filters models.OrderFilters // Changed from services.OrderFilters to models.OrderFilters

	// Parse query parameters
	if clientIDStr := c.Query("client_id"); clientIDStr != "" {
		clientID, err := strconv.ParseInt(clientIDStr, 10, 64)
		if err == nil {
			filters.ClientID = &clientID
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid client_id format.", err.Error()))
			return
		}
	}
	if staffIDStr := c.Query("staff_id"); staffIDStr != "" {
		staffID, err := strconv.ParseInt(staffIDStr, 10, 64)
		if err == nil {
			filters.StaffID = &staffID
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid staff_id format.", err.Error()))
			return
		}
	}
	if tableIDStr := c.Query("table_id"); tableIDStr != "" {
		tableID, err := strconv.ParseInt(tableIDStr, 10, 64)
		if err == nil {
			filters.TableID = &tableID
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid table_id format.", err.Error()))
			return
		}
	}
	if status := c.Query("status"); status != "" {
		filters.Status = &status
	}
	if date := c.Query("date"); date != "" {
		filters.Date = &date
	}
	if pageStr := c.Query("page"); pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err == nil && page > 0 {
			filters.Page = page
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid page format.", "page must be a positive integer"))
			return
		}
	} else {
		filters.Page = 1 // Default to page 1
	}
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		pageSize, err := strconv.Atoi(pageSizeStr)
		if err == nil && pageSize > 0 {
			filters.PageSize = pageSize
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid page_size format.", "page_size must be a positive integer"))
			return
		}
	} else {
		filters.PageSize = 10 // Default page size
	}

	// The GetOrders method in OrderService now returns (orders []models.Order, totalCount int, err error)
	// The handler needs to adapt to this.
	orders, totalCount, err := h.orderService.GetOrders(filters)
	if err != nil {
		utils.LogError(err, "GetOrders: Error from orderService.GetOrders")
		// Check if it's a specific validation error for date format from service
		// This specific error check might need to be more robust if the error message changes
		if filters.Date != nil && err.Error() == "invalid date filter format: "+*filters.Date+", expected YYYY-MM-DD" {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid date format. Use YYYY-MM-DD.", err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to fetch orders.", "Internal error"))
		}
		return
	}

	// TODO: The service currently returns []models.Order.
	// If pagination is added to service to return total count, the response structure here might change.
	// For now, just returning the slice of orders. A common pattern is to return a struct like:
	if orders == nil { // Ensure we return an empty list instead of null if no orders found
		orders = []models.Order{}
	}
	c.JSON(http.StatusOK, gin.H{
		"data":      orders,
		"total":     totalCount,
		"page":      filters.Page,
		"page_size": filters.PageSize,
	})
}

// GetOrderByID handles fetching a single order by ID with its items
func (h *OrderHandler) GetOrderByID(c *gin.Context) {
	idStr := c.Param("id")
	orderID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid order ID format.", err.Error()))
		return
	}

	order, err := h.orderService.GetOrderByID(orderID)
	if err != nil {
		utils.LogError(err, "GetOrderByID: Error from orderService.GetOrderByID for ID "+idStr)
		if errors.Is(err, services.ErrOrderNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusNotFound, utils.ErrCodeNotFound, "Order not found.", err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to fetch order.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusOK, order)
}

// UpdateOrderStatus handles updating the status of an order
func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	idStr := c.Param("id")
	orderID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid order ID format.", err.Error()))
		return
	}

	var req services.UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.LogError(err, "UpdateOrderStatus: Failed to bind JSON for ID "+idStr)
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid request payload: "+err.Error(), err.Error()))
		return
	}

	updatedOrder, err := h.orderService.UpdateOrderStatus(orderID, req)
	if err != nil {
		utils.LogError(err, "UpdateOrderStatus: Error from orderService.UpdateOrderStatus for ID "+idStr)
		if errors.Is(err, services.ErrOrderNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusNotFound, utils.ErrCodeNotFound, "Order not found to update.", err.Error()))
		} else if errors.Is(err, services.ErrInvalidOrderStatus) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid order status provided.", err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to update order status.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusOK, updatedOrder)
}

// DeleteOrder handles deleting an order
func (h *OrderHandler) DeleteOrder(c *gin.Context) {
	idStr := c.Param("id")
	orderID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid order ID format.", err.Error()))
		return
	}

	err = h.orderService.DeleteOrder(orderID)
	if err != nil {
		utils.LogError(err, "DeleteOrder: Error from orderService.DeleteOrder for ID "+idStr)
		if errors.Is(err, services.ErrOrderNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusNotFound, utils.ErrCodeNotFound, "Order not found to delete.", err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to delete order.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Order and its items deleted successfully"})
	// Or c.Status(http.StatusNoContent) if no message body is preferred for DELETE success
}
