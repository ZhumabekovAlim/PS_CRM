package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"ps_club_backend/internal/database"
	"ps_club_backend/internal/models"

	"github.com/gin-gonic/gin"
)

// CreateOrder handles the creation of a new order with its items
func CreateOrder(c *gin.Context) {
	var orderInput struct {
		ClientID       *int64               `json:"client_id"`
		BookingID      *int64               `json:"booking_id"`
		StaffID        *int64               `json:"staff_id"`
		TableID        *int64               `json:"table_id"`
		Status         string               `json:"status"`
		PaymentMethod  *string              `json:"payment_method"`
		Notes          *string              `json:"notes"`
		OrderItems     []models.OrderItem `json:"order_items" binding:"required,dive"` // dive validates each item in slice
		DiscountAmount float64              `json:"discount_amount"`
	}

	if err := c.ShouldBindJSON(&orderInput); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	db := database.GetDB()
	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start database transaction"})
		return
	}
	defer tx.Rollback() // Rollback if not committed

	// Calculate total amount from order items
	var totalAmount float64
	for i, item := range orderInput.OrderItems {
		var price float64
		var currentStock sql.NullInt64
		var itemName string
		err := tx.QueryRow("SELECT price, current_stock, name FROM pricelist_items WHERE id = $1 AND is_available = TRUE", item.PricelistItemID).Scan(&price, &currentStock, &itemName)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Pricelist item with ID " + strconv.FormatInt(item.PricelistItemID, 10) + " not found or not available"})
			return
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch price for item " + strconv.FormatInt(item.PricelistItemID, 10)})
			return
		}
		orderInput.OrderItems[i].UnitPrice = price
		orderInput.OrderItems[i].TotalPrice = price * float64(item.Quantity)
		totalAmount += orderInput.OrderItems[i].TotalPrice

		// Inventory check and update (if item is stock-tracked)
		if currentStock.Valid {
			if currentStock.Int64 < int64(item.Quantity) {
				c.JSON(http.StatusConflict, gin.H{"error": "Not enough stock for item: " + itemName + " (ID: " + strconv.FormatInt(item.PricelistItemID, 10) + "). Available: " + strconv.FormatInt(currentStock.Int64, 10)})
				return
			}
			// Decrease stock
			_, err = tx.Exec("UPDATE pricelist_items SET current_stock = current_stock - $1 WHERE id = $2", item.Quantity, item.PricelistItemID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update stock for item " + itemName})
				return
			}
			// Create inventory movement record
			_, err = tx.Exec(`INSERT INTO inventory_movements (pricelist_item_id, staff_id, movement_type, quantity_changed, reason, movement_date)
                             VALUES ($1, $2, $3, $4, $5, $6)`,
				item.PricelistItemID, orderInput.StaffID, "sale", -item.Quantity, "Order creation", time.Now())
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record inventory movement for item " + itemName})
				return
			}
		}
	}

	finalAmount := totalAmount - orderInput.DiscountAmount
	if finalAmount < 0 {
		finalAmount = 0 // Cannot be negative
	}

	orderStatus := "pending"
	if orderInput.Status != "" {
		orderStatus = orderInput.Status
	}

	order := models.Order{
		ClientID:       orderInput.ClientID,
		BookingID:      orderInput.BookingID,
		StaffID:        orderInput.StaffID,
		TableID:        orderInput.TableID,
		OrderTime:      time.Now(),
		Status:         orderStatus,
		TotalAmount:    totalAmount,
		DiscountAmount: orderInput.DiscountAmount,
		FinalAmount:    finalAmount,
		PaymentMethod:  orderInput.PaymentMethod,
		Notes:          orderInput.Notes,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	orderQuery := `INSERT INTO orders 
	               (client_id, booking_id, staff_id, table_id, order_time, status, total_amount, discount_amount, final_amount, payment_method, notes, created_at, updated_at)
	               VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) 
	               RETURNING id, order_time, created_at, updated_at`
	err = tx.QueryRow(orderQuery,
		order.ClientID, order.BookingID, order.StaffID, order.TableID, order.OrderTime, order.Status,
		order.TotalAmount, order.DiscountAmount, order.FinalAmount, order.PaymentMethod, order.Notes,
		order.CreatedAt, order.UpdatedAt,
	).Scan(&order.ID, &order.OrderTime, &order.CreatedAt, &order.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order: " + err.Error()})
		return
	}

	// Insert Order Items
	orderItemQuery := `INSERT INTO order_items 
	                   (order_id, pricelist_item_id, quantity, unit_price, total_price, notes, created_at, updated_at)
	                   VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id, created_at, updated_at`
	order.OrderItems = []models.OrderItem{}
	for _, item := range orderInput.OrderItems {
		item.OrderID = order.ID
		item.CreatedAt = time.Now()
		item.UpdatedAt = time.Now()
		err := tx.QueryRow(orderItemQuery,
			item.OrderID, item.PricelistItemID, item.Quantity, item.UnitPrice, item.TotalPrice, item.Notes,
			item.CreatedAt, item.UpdatedAt,
		).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order item: " + err.Error()})
			return
		}
		order.OrderItems = append(order.OrderItems, item)
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusCreated, order)
}

// GetOrders handles fetching all orders with filters
func GetOrders(c *gin.Context) {
	db := database.GetDB()
	baseQuery := `
        SELECT
            o.id, o.client_id, o.booking_id, o.staff_id, o.table_id, o.order_time, o.status,
            o.total_amount, o.discount_amount, o.final_amount, o.payment_method, o.notes, o.created_at, o.updated_at,
            c.full_name as client_name, c.phone_number as client_phone,
            gt.name as table_name,
            u.full_name as staff_name
        FROM orders o
        LEFT JOIN clients c ON o.client_id = c.id
        LEFT JOIN game_tables gt ON o.table_id = gt.id
        LEFT JOIN staff_members sm ON o.staff_id = sm.id
        LEFT JOIN users u ON sm.user_id = u.id
    `
	var conditions []string
	var args []interface{}
	argCounter := 1

	clientID := c.Query("client_id")
	if clientID != "" {
		conditions = append(conditions, "o.client_id = $"+strconv.Itoa(argCounter))
		args = append(args, clientID)
		argCounter++
	}
	staffID := c.Query("staff_id")
	if staffID != "" {
		conditions = append(conditions, "o.staff_id = $"+strconv.Itoa(argCounter))
		args = append(args, staffID)
		argCounter++
	}
	tableID := c.Query("table_id")
	if tableID != "" {
		conditions = append(conditions, "o.table_id = $"+strconv.Itoa(argCounter))
		args = append(args, tableID)
		argCounter++
	}
	status := c.Query("status")
	if status != "" {
		conditions = append(conditions, "o.status = $"+strconv.Itoa(argCounter))
		args = append(args, status)
		argCounter++
	}
	date := c.Query("date") // YYYY-MM-DD
	if date != "" {
		parsedDate, err := time.Parse("2006-01-02", date)
		if err == nil {
			startOfDay := time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), 0, 0, 0, 0, parsedDate.Location())
			endOfDay := startOfDay.AddDate(0, 0, 1).Add(-time.Nanosecond)
			conditions = append(conditions, "o.order_time BETWEEN $"+strconv.Itoa(argCounter)+" AND $"+strconv.Itoa(argCounter+1))
			args = append(args, startOfDay, endOfDay)
			argCounter += 2
		}
	}

	if len(conditions) > 0 {
		baseQuery += " WHERE " + string(join(conditions, " AND "))
	}
	baseQuery += " ORDER BY o.order_time DESC"

	rows, err := db.Query(baseQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders: " + err.Error()})
		return
	}
	defer rows.Close()

	orders := []models.Order{}
	for rows.Next() {
		var o models.Order
		var clientName, clientPhone, tableName, staffName sql.NullString
		if err := rows.Scan(
			&o.ID, &o.ClientID, &o.BookingID, &o.StaffID, &o.TableID, &o.OrderTime, &o.Status,
			&o.TotalAmount, &o.DiscountAmount, &o.FinalAmount, &o.PaymentMethod, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
			&clientName, &clientPhone, &tableName, &staffName,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan order: " + err.Error()})
			return
		}
		if o.ClientID != nil {
			o.Client = &models.Client{ID: *o.ClientID}
			if clientName.Valid { o.Client.FullName = clientName.String }
			if clientPhone.Valid { o.Client.PhoneNumber = &clientPhone.String }
		}
		if o.TableID != nil {
			o.GameTable = &models.GameTable{ID: *o.TableID}
			if tableName.Valid { o.GameTable.Name = tableName.String }
		}
		if o.StaffID != nil {
			o.StaffMember = &models.StaffMember{ID: *o.StaffID}
			if staffName.Valid { o.StaffMember.User = &models.User{FullName: &staffName.String} }
		}
		orders = append(orders, o)
	}
	c.JSON(http.StatusOK, orders)
}

// GetOrderByID handles fetching a single order by ID with its items
func GetOrderByID(c *gin.Context) {
	idStr := c.Param("id")
	orderID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	db := database.GetDB()
	var o models.Order
	var clientName, clientPhone, tableName, staffName sql.NullString

	orderQuery := `
        SELECT
            o.id, o.client_id, o.booking_id, o.staff_id, o.table_id, o.order_time, o.status,
            o.total_amount, o.discount_amount, o.final_amount, o.payment_method, o.notes, o.created_at, o.updated_at,
            c.full_name as client_name, c.phone_number as client_phone,
            gt.name as table_name,
            u.full_name as staff_name
        FROM orders o
        LEFT JOIN clients c ON o.client_id = c.id
        LEFT JOIN game_tables gt ON o.table_id = gt.id
        LEFT JOIN staff_members sm ON o.staff_id = sm.id
        LEFT JOIN users u ON sm.user_id = u.id
        WHERE o.id = $1`
	err = db.QueryRow(orderQuery, orderID).Scan(
		&o.ID, &o.ClientID, &o.BookingID, &o.StaffID, &o.TableID, &o.OrderTime, &o.Status,
		&o.TotalAmount, &o.DiscountAmount, &o.FinalAmount, &o.PaymentMethod, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
		&clientName, &clientPhone, &tableName, &staffName,
	)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch order: " + err.Error()})
		return
	}

	if o.ClientID != nil {
		o.Client = &models.Client{ID: *o.ClientID}
		if clientName.Valid { o.Client.FullName = clientName.String }
		if clientPhone.Valid { o.Client.PhoneNumber = &clientPhone.String }
	}
	if o.TableID != nil {
		o.GameTable = &models.GameTable{ID: *o.TableID}
		if tableName.Valid { o.GameTable.Name = tableName.String }
	}
	if o.StaffID != nil {
		o.StaffMember = &models.StaffMember{ID: *o.StaffID}
		if staffName.Valid { o.StaffMember.User = &models.User{FullName: &staffName.String} }
	}

	// Fetch Order Items
	itemRows, err := db.Query(`
        SELECT oi.id, oi.order_id, oi.pricelist_item_id, oi.quantity, oi.unit_price, oi.total_price, oi.notes, oi.created_at, oi.updated_at,
               pi.name as item_name, pi.sku as item_sku
        FROM order_items oi
        JOIN pricelist_items pi ON oi.pricelist_item_id = pi.id
        WHERE oi.order_id = $1 ORDER BY oi.id`, orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch order items: " + err.Error()})
		return
	}
	defer itemRows.Close()

	o.OrderItems = []models.OrderItem{}
	for itemRows.Next() {
		var item models.OrderItem
		var itemName, itemSKU sql.NullString
		if err := itemRows.Scan(
			&item.ID, &item.OrderID, &item.PricelistItemID, &item.Quantity, &item.UnitPrice, &item.TotalPrice, &item.Notes, &item.CreatedAt, &item.UpdatedAt,
			&itemName, &itemSKU,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan order item: " + err.Error()})
			return
		}
		item.PricelistItem = &models.PricelistItem{ID: item.PricelistItemID}
		if itemName.Valid { item.PricelistItem.Name = itemName.String }
		if itemSKU.Valid { item.PricelistItem.SKU = &itemSKU.String }
		o.OrderItems = append(o.OrderItems, item)
	}

	c.JSON(http.StatusOK, o)
}

// UpdateOrderStatus handles updating the status of an order
func UpdateOrderStatus(c *gin.Context) {
	idStr := c.Param("id")
	orderID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var statusUpdate struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&statusUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// TODO: Add logic if status change implies other actions (e.g., payment processing, inventory return for 'cancelled')

	db := database.GetDB()
	query := `UPDATE orders SET status = $1, updated_at = $2 WHERE id = $3 
	          RETURNING id, client_id, booking_id, staff_id, table_id, order_time, status, 
	          total_amount, discount_amount, final_amount, payment_method, notes, created_at, updated_at`

	updatedAt := time.Now()
	var o models.Order
	err = db.QueryRow(query, statusUpdate.Status, updatedAt, orderID).Scan(
		&o.ID, &o.ClientID, &o.BookingID, &o.StaffID, &o.TableID, &o.OrderTime, &o.Status,
		&o.TotalAmount, &o.DiscountAmount, &o.FinalAmount, &o.PaymentMethod, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found to update"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order status: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, o)
}

// DeleteOrder handles deleting an order (use with caution, usually orders are cancelled or archived)
func DeleteOrder(c *gin.Context) {
	idStr := c.Param("id")
	orderID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	db := database.GetDB()
	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Before deleting order, consider implications: inventory should be returned if order was 'completed' or 'pending'
	// For simplicity, this example doesn't automatically handle inventory rollback on delete.
	// That logic would typically be in a service layer or more complex handler.

	// First delete order items
	_, err = tx.Exec("DELETE FROM order_items WHERE order_id = $1", orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete order items: " + err.Error()})
		return
	}

	// Then delete the order itself
	result, err := tx.Exec("DELETE FROM orders WHERE id = $1", orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete order: " + err.Error()})
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found to delete"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order and its items deleted successfully"})
}

