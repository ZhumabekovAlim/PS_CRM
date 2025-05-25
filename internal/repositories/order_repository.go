package repositories

import (
	"database/sql"
	"errors"
	"fmt"
	"ps_club_backend/internal/models"
	"strings"
	"time"

	"github.com/lib/pq" // For pq.Error
)

// OrderRepository defines the interface for order-related database operations.
type OrderRepository interface {
	// Order methods
	CreateOrder(executor SQLExecutor, order *models.Order) (int64, error)
	GetOrderByID(orderID int64) (*models.Order, error) // Basic order details
	GetOrders(filters models.OrderFilters) ([]models.Order, int, error) // orders, total count, error
	UpdateOrderStatus(executor SQLExecutor, orderID int64, newStatus string, updatedAt time.Time) error
	DeleteOrder(executor SQLExecutor, orderID int64) (int64, error) // Returns rows affected or error

	// OrderItem methods
	CreateOrderItem(executor SQLExecutor, item *models.OrderItem) (int64, error)
	GetOrderItemsByOrderID(orderID int64) ([]models.OrderItem, error)
	DeleteOrderItemsByOrderID(executor SQLExecutor, orderID int64) (int64, error) // Returns rows affected or error
}

type orderRepository struct {
	db *sql.DB
}

// NewOrderRepository creates a new instance of OrderRepository.
func NewOrderRepository(db *sql.DB) OrderRepository {
	return &orderRepository{db: db}
}

// --- Order Methods ---

func (r *orderRepository) CreateOrder(executor SQLExecutor, order *models.Order) (int64, error) {
	query := `INSERT INTO orders 
	            (client_id, booking_id, staff_id, table_id, order_time, status, 
	             total_amount, discount_amount, final_amount, payment_method, notes, 
	             created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) 
	          RETURNING id`
	
	if order.OrderTime.IsZero() { order.OrderTime = time.Now() }
	if order.CreatedAt.IsZero() { order.CreatedAt = time.Now() }
	if order.UpdatedAt.IsZero() { order.UpdatedAt = time.Now() }

	err := executor.QueryRow(query,
		order.ClientID, order.BookingID, order.StaffID, order.TableID, order.OrderTime, order.Status,
		order.TotalAmount, order.DiscountAmount, order.FinalAmount, order.PaymentMethod, order.Notes,
		order.CreatedAt, order.UpdatedAt,
	).Scan(&order.ID)

	if err != nil {
		return 0, fmt.Errorf("%w: creating order: %v", ErrDatabaseError, err)
	}
	return order.ID, nil
}

func (r *orderRepository) GetOrderByID(orderID int64) (*models.Order, error) {
	order := &models.Order{}
	query := `SELECT id, client_id, booking_id, staff_id, table_id, order_time, status, 
	                 total_amount, discount_amount, final_amount, payment_method, notes, 
	                 created_at, updated_at 
	          FROM orders 
	          WHERE id = $1`
	err := r.db.QueryRow(query, orderID).Scan(
		&order.ID, &order.ClientID, &order.BookingID, &order.StaffID, &order.TableID, &order.OrderTime, &order.Status,
		&order.TotalAmount, &order.DiscountAmount, &order.FinalAmount, &order.PaymentMethod, &order.Notes,
		&order.CreatedAt, &order.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("%w: getting order by ID %d: %v", ErrDatabaseError, orderID, err)
	}
	return order, nil
}

func (r *orderRepository) GetOrders(filters models.OrderFilters) ([]models.Order, int, error) {
	orders := []models.Order{}
	totalCount := 0

	var queryBuilder strings.Builder
	queryBuilder.WriteString(`
        SELECT
            o.id, o.client_id, o.booking_id, o.staff_id, o.table_id, o.order_time, o.status,
            o.total_amount, o.discount_amount, o.final_amount, o.payment_method, o.notes, 
            o.created_at, o.updated_at,
            c.full_name as client_name, c.phone_number as client_phone,
            gt.name as table_name,
            u.full_name as staff_name,
            COUNT(*) OVER() as total_count
        FROM orders o
        LEFT JOIN clients c ON o.client_id = c.id
        LEFT JOIN game_tables gt ON o.table_id = gt.id
        LEFT JOIN staff_members sm ON o.staff_id = sm.id
        LEFT JOIN users u ON sm.user_id = u.id
    `)

	var conditions []string
	var args []interface{}
	argCounter := 1

	if filters.ClientID != nil {
		conditions = append(conditions, fmt.Sprintf("o.client_id = $%d", argCounter))
		args = append(args, *filters.ClientID)
		argCounter++
	}
	if filters.StaffID != nil {
		conditions = append(conditions, fmt.Sprintf("o.staff_id = $%d", argCounter))
		args = append(args, *filters.StaffID)
		argCounter++
	}
	if filters.TableID != nil {
		conditions = append(conditions, fmt.Sprintf("o.table_id = $%d", argCounter))
		args = append(args, *filters.TableID)
		argCounter++
	}
	if filters.Status != nil && *filters.Status != "" {
		conditions = append(conditions, fmt.Sprintf("o.status = $%d", argCounter))
		args = append(args, *filters.Status)
		argCounter++
	}
	if filters.Date != nil && *filters.Date != "" {
		parsedDate, err := time.Parse("2006-01-02", *filters.Date)
		if err == nil {
			startOfDay := time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), 0, 0, 0, 0, parsedDate.Location())
			endOfDay := startOfDay.AddDate(0, 0, 1).Add(-time.Nanosecond)
			conditions = append(conditions, fmt.Sprintf("o.order_time BETWEEN $%d AND $%d", argCounter, argCounter+1))
			args = append(args, startOfDay, endOfDay)
			argCounter += 2
		}
	}

	if len(conditions) > 0 {
		queryBuilder.WriteString(" WHERE " + strings.Join(conditions, " AND "))
	}
	queryBuilder.WriteString(" ORDER BY o.order_time DESC")

	if filters.PageSize > 0 {
		queryBuilder.WriteString(fmt.Sprintf(" LIMIT $%d", argCounter))
		args = append(args, filters.PageSize)
		argCounter++
		if filters.Page > 0 {
			offset := (filters.Page - 1) * filters.PageSize
			queryBuilder.WriteString(fmt.Sprintf(" OFFSET $%d", argCounter))
			args = append(args, offset)
			// argCounter++ // Not needed as this is the last placeholder
		}
	}

	rows, err := r.db.Query(queryBuilder.String(), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: querying orders: %v", ErrDatabaseError, err)
	}
	defer rows.Close()

	for rows.Next() {
		var o models.Order
		var clientName, clientPhone, tableName, staffName sql.NullString
		
		var client models.Client
		var gameTable models.GameTable
		var staffMember models.StaffMember
		var user models.User

		err := rows.Scan(
			&o.ID, &o.ClientID, &o.BookingID, &o.StaffID, &o.TableID, &o.OrderTime, &o.Status,
			&o.TotalAmount, &o.DiscountAmount, &o.FinalAmount, &o.PaymentMethod, &o.Notes,
			&o.CreatedAt, &o.UpdatedAt,
			&clientName, &clientPhone, &tableName, &staffName,
			&totalCount,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("%w: scanning order: %v", ErrDatabaseError, err)
		}

		if o.ClientID != nil {
			client.ID = *o.ClientID
			if clientName.Valid { client.FullName = clientName.String }
			if clientPhone.Valid { phone := clientPhone.String; client.PhoneNumber = &phone }
			o.Client = &client
		}
		if o.TableID != nil {
			gameTable.ID = *o.TableID
			if tableName.Valid { gameTable.Name = tableName.String }
			o.GameTable = &gameTable
		}
		if o.StaffID != nil {
			staffMember.ID = *o.StaffID
			if staffName.Valid { name := staffName.String; user.FullName = &name }
			staffMember.User = &user
			o.StaffMember = &staffMember
		}
		orders = append(orders, o)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("%w: iterating order rows: %v", ErrDatabaseError, err)
	}
	return orders, totalCount, nil
}

func (r *orderRepository) UpdateOrderStatus(executor SQLExecutor, orderID int64, newStatus string, updatedAt time.Time) error {
	query := `UPDATE orders SET status = $1, updated_at = $2 WHERE id = $3`
	result, err := executor.Exec(query, newStatus, updatedAt, orderID)
	if err != nil {
		return fmt.Errorf("%w: updating order status for ID %d: %v", ErrDatabaseError, orderID, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: getting rows affected for order status update ID %d: %v", ErrDatabaseError, orderID, err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *orderRepository) DeleteOrder(executor SQLExecutor, orderID int64) (int64, error) {
	query := `DELETE FROM orders WHERE id = $1`
	result, err := executor.Exec(query, orderID)
	if err != nil {
		return 0, fmt.Errorf("%w: deleting order ID %d: %v", ErrDatabaseError, orderID, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("%w: getting rows affected for deleting order ID %d: %v", ErrDatabaseError, orderID, err)
	}
	if rowsAffected == 0 {
		return 0, ErrNotFound 
	}
	return rowsAffected, nil
}

// --- OrderItem Methods ---

func (r *orderRepository) CreateOrderItem(executor SQLExecutor, item *models.OrderItem) (int64, error) {
	query := `INSERT INTO order_items 
	            (order_id, pricelist_item_id, quantity, unit_price, total_price, notes, 
	             created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	          RETURNING id`
	if item.CreatedAt.IsZero() { item.CreatedAt = time.Now() }
	if item.UpdatedAt.IsZero() { item.UpdatedAt = time.Now() }
	
	err := executor.QueryRow(query,
		item.OrderID, item.PricelistItemID, item.Quantity, item.UnitPrice, item.TotalPrice, item.Notes,
		item.CreatedAt, item.UpdatedAt,
	).Scan(&item.ID)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23503" { 
			return 0, fmt.Errorf("%w: creating order item (constraint: %s): %v", ErrDatabaseError, pqErr.Constraint, err)
		}
		return 0, fmt.Errorf("%w: creating order item: %v", ErrDatabaseError, err)
	}
	return item.ID, nil
}

func (r *orderRepository) GetOrderItemsByOrderID(orderID int64) ([]models.OrderItem, error) {
	items := []models.OrderItem{}
	query := `
		SELECT 
		    oi.id, oi.order_id, oi.pricelist_item_id, oi.quantity, oi.unit_price, 
		    oi.total_price, oi.notes, oi.created_at, oi.updated_at,
		    pi.name as item_name, pi.sku as item_sku, pi.tracks_stock as item_tracks_stock
		FROM order_items oi
		JOIN pricelist_items pi ON oi.pricelist_item_id = pi.id
		WHERE oi.order_id = $1
		ORDER BY oi.id`

	rows, err := r.db.Query(query, orderID)
	if err != nil {
		return nil, fmt.Errorf("%w: querying order items for order ID %d: %v", ErrDatabaseError, orderID, err)
	}
	defer rows.Close()

	for rows.Next() {
		var item models.OrderItem
		var pricelistItem models.PricelistItem 
		var itemName, itemSKU sql.NullString
		var itemTracksStock sql.NullBool

		err := rows.Scan(
			&item.ID, &item.OrderID, &item.PricelistItemID, &item.Quantity, &item.UnitPrice,
			&item.TotalPrice, &item.Notes, &item.CreatedAt, &item.UpdatedAt,
			&itemName, &itemSKU, &itemTracksStock,
		)
		if err != nil {
			return nil, fmt.Errorf("%w: scanning order item for order ID %d: %v", ErrDatabaseError, orderID, err)
		}
		
		pricelistItem.ID = item.PricelistItemID 
		if itemName.Valid { pricelistItem.Name = itemName.String }
		if itemSKU.Valid { sku := itemSKU.String; pricelistItem.SKU = &sku }
		if itemTracksStock.Valid { pricelistItem.TracksStock = itemTracksStock.Bool }
		item.PricelistItem = &pricelistItem

		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: iterating order item rows for order ID %d: %v", ErrDatabaseError, orderID, err)
	}
	return items, nil
}

func (r *orderRepository) DeleteOrderItemsByOrderID(executor SQLExecutor, orderID int64) (int64, error) {
	query := `DELETE FROM order_items WHERE order_id = $1`
	result, err := executor.Exec(query, orderID)
	if err != nil {
		return 0, fmt.Errorf("%w: deleting order items for order ID %d: %v", ErrDatabaseError, orderID, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("%w: getting rows affected for deleting order items for order ID %d: %v", ErrDatabaseError, orderID, err)
	}
	return rowsAffected, nil
}
