package services

import (
	"database/sql"
	"errors"
	"fmt"
	"ps_club_backend/internal/models"
	"ps_club_backend/internal/repositories" // Added
	// "strconv" // No longer needed here
	"strings"
	"time"
)

// Custom Errors - some might be redefined or become more specific
var (
	ErrPricelistItemNotFound = errors.New("pricelist item not found or not available")
	ErrInsufficientStock     = errors.New("insufficient stock for item")
	ErrOrderNotFound         = errors.New("order not found")
	ErrInvalidOrderStatus    = errors.New("invalid order status")
	// TODO: Consider adding more specific errors for different failure scenarios
	// e.g., ErrOrderCreationConflict if some underlying data changed during creation
)

// OrderStatus constants - these remain the same
const (
	StatusPending    = "pending"
	StatusCompleted  = "completed"
	StatusCancelled  = "cancelled"
	StatusPreparing  = "preparing"
	StatusReady      = "ready"
	StatusServed     = "served"
	StatusPaid       = "paid"
	StatusRefunded   = "refunded"
)

// --- Data Transfer Objects (DTOs) --- (These remain the same as they are for service input/output)

// CreateOrderItemRequest is used for creating individual order items.
type CreateOrderItemRequest struct {
	PricelistItemID int64  `json:"pricelist_item_id" binding:"required"`
	Quantity        int    `json:"quantity" binding:"required,gt=0"`
	Notes           string `json:"notes"`
}

// CreateOrderRequest is used for creating a new order.
type CreateOrderRequest struct {
	ClientID       *int64                   `json:"client_id"`
	BookingID      *int64                   `json:"booking_id"`
	StaffID        int64                    `json:"staff_id" binding:"required"`
	TableID        *int64                   `json:"table_id"`
	Status         string                   `json:"status" binding:"required"`
	PaymentMethod  *string                  `json:"payment_method"`
	Notes          *string                  `json:"notes"`
	OrderItems     []CreateOrderItemRequest `json:"order_items" binding:"required,dive"`
	DiscountAmount *float64                 `json:"discount_amount"`
}

// OrderItemResponse represents an item within an order for API responses.
// This DTO might be adjusted or built from models.OrderItem and models.PricelistItem
type OrderItemResponse struct {
	ID              int64    `json:"id"`
	PricelistItemID int64    `json:"pricelist_item_id"`
	ItemName        string   `json:"item_name"`
	Quantity        int      `json:"quantity"`
	UnitPrice       float64  `json:"unit_price"`
	TotalPrice      float64  `json:"total_price"`
	Notes           *string  `json:"notes"`
}

// OrderResponse represents a complete order for API responses.
// This DTO will be built from models.Order and its related data.
type OrderResponse struct {
	ID             int64               `json:"id"`
	ClientName     *string             `json:"client_name,omitempty"`
	BookingID      *int64              `json:"booking_id,omitempty"`
	StaffName      *string             `json:"staff_name,omitempty"` // Changed to pointer
	TableName      *string             `json:"table_name,omitempty"`
	Status         string              `json:"status"`
	TotalAmount    float64             `json:"total_amount"`
	DiscountAmount float64             `json:"discount_amount"`
	FinalAmount    float64             `json:"final_amount"`
	PaymentMethod  *string             `json:"payment_method,omitempty"`
	Notes          *string             `json:"notes,omitempty"`
	OrderItems     []OrderItemResponse `json:"order_items"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
}

// UpdateOrderStatusRequest is used for updating the status of an order.
type UpdateOrderStatusRequest struct {
	Status string `json:"status" binding:"required"`
}
// --- End of DTOs ---


// --- OrderService Interface ---
type OrderService interface {
	CreateOrder(req CreateOrderRequest) (*models.Order, error) // Returning models.Order for now
	GetOrders(filters models.OrderFilters) ([]models.Order, int, error) // Added totalCount
	GetOrderByID(orderID int64) (*models.Order, error) // Returning models.Order with items
	UpdateOrderStatus(orderID int64, req UpdateOrderStatusRequest) (*models.Order, error)
	DeleteOrder(orderID int64) error
}

// --- orderService Implementation ---
type orderService struct {
	orderRepo        repositories.OrderRepository
	pricelistRepo    repositories.PricelistRepository
	inventoryMvRepo  repositories.InventoryMovementRepository
	db               *sql.DB // For managing transactions
}

// NewOrderService creates a new instance of OrderService.
func NewOrderService(
	or repositories.OrderRepository,
	pr repositories.PricelistRepository,
	imr repositories.InventoryMovementRepository,
	db *sql.DB,
) OrderService {
	return &orderService{
		orderRepo:        or,
		pricelistRepo:    pr,
		inventoryMvRepo:  imr,
		db:               db,
	}
}

// --- Method Implementations ---

func (s *orderService) CreateOrder(req CreateOrderRequest) (*models.Order, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start database transaction: %w", err)
	}
	defer tx.Rollback()

	var totalAmount float64
	orderItemsToCreate := make([]models.OrderItem, 0, len(req.OrderItems))

	for _, itemReq := range req.OrderItems {
		if itemReq.Quantity <= 0 {
			return nil, fmt.Errorf("%w: quantity for item ID %d must be positive", ErrValidation, itemReq.PricelistItemID)
		}
		price, stock, itemName, tracksStock, repoErr := s.pricelistRepo.GetItemPriceAndStock(itemReq.PricelistItemID)
		if repoErr != nil {
			if errors.Is(repoErr, repositories.ErrNotFound) {
				return nil, fmt.Errorf("%w: item ID %d", ErrPricelistItemNotFound, itemReq.PricelistItemID)
			}
			return nil, fmt.Errorf("failed to fetch pricelist item %d details: %w", itemReq.PricelistItemID, repoErr)
		}

		itemTotalPrice := price * float64(itemReq.Quantity)
		totalAmount += itemTotalPrice

		if tracksStock {
			if !stock.Valid || stock.Int64 < int64(itemReq.Quantity) {
				return nil, fmt.Errorf("%w %s (ID: %d). Requested: %d, Available: %d",
					ErrInsufficientStock, itemName, itemReq.PricelistItemID, itemReq.Quantity, stock.Int64)
			}
			_, repoErr = s.pricelistRepo.UpdateStock(tx, itemReq.PricelistItemID, -itemReq.Quantity)
			if repoErr != nil {
				return nil, fmt.Errorf("failed to update stock for item %s (ID: %d): %w", itemName, itemReq.PricelistItemID, repoErr)
			}
			movement := models.InventoryMovement{
				PricelistItemID: itemReq.PricelistItemID,
				StaffID:         &req.StaffID,
				MovementType:    MovementTypeSale,
				QuantityChanged: -itemReq.Quantity,
				Reason:          models.NewNullString("Order creation"),
				MovementDate:    time.Now(),
			}
			_, repoErr = s.inventoryMvRepo.CreateMovement(tx, &movement)
			if repoErr != nil {
				return nil, fmt.Errorf("failed to record inventory movement for sale of item %s (ID: %d): %w", itemName, itemReq.PricelistItemID, repoErr)
			}
		}
		orderItemsToCreate = append(orderItemsToCreate, models.OrderItem{
			PricelistItemID: itemReq.PricelistItemID,
			Quantity:        itemReq.Quantity,
			UnitPrice:       price,
			TotalPrice:      itemTotalPrice,
			Notes:           models.NewNullString(itemReq.Notes),
		})
	}

	finalAmount := totalAmount
	if req.DiscountAmount != nil {
		finalAmount = totalAmount - *req.DiscountAmount
		if finalAmount < 0 {
			finalAmount = 0
		}
	}

	if !isValidOrderStatus(req.Status) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidOrderStatus, req.Status)
	}

	order := models.Order{
		ClientID:       req.ClientID,
		BookingID:      req.BookingID,
		StaffID:        &req.StaffID,
		TableID:        req.TableID,
		Status:         req.Status,
		TotalAmount:    totalAmount,
		DiscountAmount: req.DiscountAmount, // This is already a *float64
		FinalAmount:    finalAmount,
		PaymentMethod:  req.PaymentMethod,
		Notes:          req.Notes,
		OrderTime:      time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	createdOrderID, repoErr := s.orderRepo.CreateOrder(tx, &order)
	if repoErr != nil {
		return nil, fmt.Errorf("failed to create order record: %w", repoErr)
	}
	order.ID = createdOrderID

	for _, itemModel := range orderItemsToCreate {
		itemModel.OrderID = createdOrderID // Link item to the created order
		_, repoErr = s.orderRepo.CreateOrderItem(tx, &itemModel)
		if repoErr != nil {
			return nil, fmt.Errorf("failed to create order item (pricelist_item_id: %d): %w", itemModel.PricelistItemID, repoErr)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit order transaction: %w", err)
	}

	// Fetch the full order to return, including joined data and order items
	return s.GetOrderByID(createdOrderID)
}

func (s *orderService) GetOrders(filters models.OrderFilters) ([]models.Order, int, error) {
	orders, totalCount, err := s.orderRepo.GetOrders(filters)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get orders: %w", err)
	}
	// The repository GetOrders now includes joins for client, staff, table names.
	// No need to fetch order items for the list view usually.
	return orders, totalCount, nil
}

func (s *orderService) GetOrderByID(orderID int64) (*models.Order, error) {
	order, err := s.orderRepo.GetOrderByID(orderID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to get order by ID from repository: %w", err)
	}

	items, err := s.orderRepo.GetOrderItemsByOrderID(orderID)
	if err != nil {
		// Log this error but don't necessarily fail the whole request if order header is found
		fmt.Printf("Warning: failed to get order items for order ID %d: %v\n", orderID, err)
		// Depending on requirements, might return order without items or a specific error
	}
	order.OrderItems = items

	// The s.orderRepo.GetOrderByID does not currently join related names.
	// For now, we will rely on the more detailed s.orderRepo.GetOrders for that.
	// If specific names are needed here, we'd call other repos or enhance GetOrderByID in orderRepo.
	// For the purpose of this refactor, we'll return the order as is from GetOrderByID and GetOrderItemsByOrderID.
	// The GetOrders in repo already fetches names.
	// If detailed Client, Staff, Table data is needed beyond what GetOrders provides,
	// individual repo calls would be needed here.
	// For example:
	// if order.ClientID != nil { s.clientRepo.GetClientByID(*order.ClientID) ... }
	// if order.StaffID != nil { s.staffRepo.GetStaffByID(*order.StaffID) ... }
	// if order.TableID != nil { s.tableRepo.GetTableByID(*order.TableID) ... }

	return order, nil
}

func (s *orderService) UpdateOrderStatus(orderID int64, req UpdateOrderStatusRequest) (*models.Order, error) {
	if !isValidOrderStatus(req.Status) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidOrderStatus, req.Status)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	currentOrder, err := s.orderRepo.GetOrderByID(orderID) // Get current order for status and staff ID
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to fetch order for status update: %w", err)
	}

	if req.Status == StatusCancelled && currentOrder.Status != StatusCancelled && currentOrder.Status != StatusRefunded {
		orderItems, repoErr := s.orderRepo.GetOrderItemsByOrderID(orderID)
		if repoErr != nil {
			return nil, fmt.Errorf("failed to fetch order items for stock return: %w", repoErr)
		}
		for _, item := range orderItems {
			// Need to check PricelistItem's TracksStock status
			_, _, _, tracksStock, itemDetailErr := s.pricelistRepo.GetItemPriceAndStock(item.PricelistItemID)
			if itemDetailErr != nil {
				return nil, fmt.Errorf("failed to get item details for stock return (item ID %d): %w", item.PricelistItemID, itemDetailErr)
			}

			if tracksStock {
				_, repoErr = s.pricelistRepo.UpdateStock(tx, item.PricelistItemID, item.Quantity) // Return positive quantity
				if repoErr != nil {
					return nil, fmt.Errorf("failed to return stock for item ID %d: %w", item.PricelistItemID, repoErr)
				}
				movement := models.InventoryMovement{
					PricelistItemID: item.PricelistItemID,
					StaffID:         currentOrder.StaffID, // Use staff ID from the order
					MovementType:    MovementTypeReturnCancellation,
					QuantityChanged: item.Quantity, // Positive quantity for return
					Reason:          models.NewNullString(fmt.Sprintf("Order %d cancelled", orderID)),
					MovementDate:    time.Now(),
				}
				_, repoErr = s.inventoryMvRepo.CreateMovement(tx, &movement)
				if repoErr != nil {
					return nil, fmt.Errorf("failed to record inventory movement for stock return (item ID %d): %w", item.PricelistItemID, repoErr)
				}
			}
		}
	}

	err = s.orderRepo.UpdateOrderStatus(tx, orderID, req.Status, time.Now())
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("failed to update order status in repository: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction for order status update: %w", err)
	}
	return s.GetOrderByID(orderID)
}

func (s *orderService) DeleteOrder(orderID int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	order, err := s.orderRepo.GetOrderByID(orderID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ErrOrderNotFound
		}
		return fmt.Errorf("failed to fetch order for deletion: %w", err)
	}

	if order.Status != StatusCancelled && order.Status != StatusRefunded {
		orderItems, repoErr := s.orderRepo.GetOrderItemsByOrderID(orderID)
		if repoErr != nil {
			return fmt.Errorf("failed to fetch order items for stock return on delete: %w", repoErr)
		}
		for _, item := range orderItems {
			_, _, _, tracksStock, itemDetailErr := s.pricelistRepo.GetItemPriceAndStock(item.PricelistItemID)
			if itemDetailErr != nil {
				return fmt.Errorf("failed to get item details for stock return (item ID %d): %w", item.PricelistItemID, itemDetailErr)
			}
			if tracksStock {
				_, repoErr = s.pricelistRepo.UpdateStock(tx, item.PricelistItemID, item.Quantity) // Return positive quantity
				if repoErr != nil {
					return fmt.Errorf("failed to return stock for item ID %d on delete: %w", item.PricelistItemID, repoErr)
				}
				movement := models.InventoryMovement{
					PricelistItemID: item.PricelistItemID,
					StaffID:         order.StaffID, // Use staff ID from the order
					MovementType:    MovementTypeReturnDeletion,
					QuantityChanged: item.Quantity, // Positive quantity for return
					Reason:          models.NewNullString(fmt.Sprintf("Order %d deleted", orderID)),
					MovementDate:    time.Now(),
				}
				_, repoErr = s.inventoryMvRepo.CreateMovement(tx, &movement)
				if repoErr != nil {
					return fmt.Errorf("failed to record inventory movement for stock return on delete (item ID %d): %w", item.PricelistItemID, repoErr)
				}
			}
		}
	}

	_, err = s.orderRepo.DeleteOrderItemsByOrderID(tx, orderID)
	if err != nil {
		return fmt.Errorf("failed to delete order items: %w", err)
	}

	_, err = s.orderRepo.DeleteOrder(tx, orderID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) { // Should be caught by GetOrderByID, but for safety
			return ErrOrderNotFound
		}
		return fmt.Errorf("failed to delete order: %w", err)
	}

	return tx.Commit()
}

// Helper function to validate order status (can be expanded)
func isValidOrderStatus(status string) bool {
	switch status {
	case StatusPending, StatusCompleted, StatusCancelled, StatusPreparing, StatusReady, StatusServed, StatusPaid, StatusRefunded:
		return true
	default:
		return false
	}
}
