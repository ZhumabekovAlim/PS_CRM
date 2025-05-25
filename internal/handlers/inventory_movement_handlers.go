package handlers

import (
	"net/http"
	"strconv"
	"time"

	"ps_club_backend/internal/database"
	"ps_club_backend/internal/models"

	"github.com/gin-gonic/gin"
	"database/sql"
)

// CreateInventoryMovement handles manual creation of an inventory movement
func CreateInventoryMovement(c *gin.Context) {
	var movement models.InventoryMovement
	if err := c.ShouldBindJSON(&movement); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	db := database.GetDB()
	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Validate pricelist_item_id
	var currentStock sql.NullInt64
	var itemName string
	err = tx.QueryRow("SELECT name, current_stock FROM pricelist_items WHERE id = $1", movement.PricelistItemID).Scan(&itemName, &currentStock)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pricelist item not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate pricelist item"})
		return
	}

	// Update current_stock in pricelist_items table if it's a stock-tracked item
	if currentStock.Valid {
		newStock := currentStock.Int64 + int64(movement.QuantityChanged)
		if newStock < 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "Stock cannot go below zero for item: " + itemName})
			return
		}
		_, err = tx.Exec("UPDATE pricelist_items SET current_stock = $1, updated_at = $2 WHERE id = $3", 
			newStock, time.Now(), movement.PricelistItemID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update stock for item: " + itemName})
			return
		}
	}

	movement.MovementDate = time.Now() // Or allow user to specify
	movement.CreatedAt = time.Now()
	movement.UpdatedAt = time.Now()

	query := `INSERT INTO inventory_movements 
	          (pricelist_item_id, staff_id, movement_type, quantity_changed, reason, movement_date, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8) 
	          RETURNING id, movement_date, created_at, updated_at`

	err = tx.QueryRow(query,
		movement.PricelistItemID, movement.StaffID, movement.MovementType, movement.QuantityChanged,
		movement.Reason, movement.MovementDate, movement.CreatedAt, movement.UpdatedAt,
	).Scan(&movement.ID, &movement.MovementDate, &movement.CreatedAt, &movement.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create inventory movement: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusCreated, movement)
}

// GetInventoryMovements handles fetching all inventory movements with filters
func GetInventoryMovements(c *gin.Context) {
	db := database.GetDB()

	baseQuery := `
        SELECT 
            im.id, im.pricelist_item_id, im.staff_id, im.movement_type, im.quantity_changed, 
            im.reason, im.movement_date, im.created_at, im.updated_at,
            pi.name as item_name, pi.sku as item_sku,
            u.full_name as staff_name
        FROM inventory_movements im
        JOIN pricelist_items pi ON im.pricelist_item_id = pi.id
        LEFT JOIN staff_members sm ON im.staff_id = sm.id
        LEFT JOIN users u ON sm.user_id = u.id
    `
	var conditions []string
	var args []interface{}
	argCounter := 1

	itemIDStr := c.Query("pricelist_item_id")
	if itemIDStr != "" {
		itemID, err := strconv.ParseInt(itemIDStr, 10, 64)
		if err == nil {
			conditions = append(conditions, "im.pricelist_item_id = $"+strconv.Itoa(argCounter))
			args = append(args, itemID)
			argCounter++
		}
	}

	staffIDStr := c.Query("staff_id")
	if staffIDStr != "" {
		staffID, err := strconv.ParseInt(staffIDStr, 10, 64)
		if err == nil {
			conditions = append(conditions, "im.staff_id = $"+strconv.Itoa(argCounter))
			args = append(args, staffID)
			argCounter++
		}
	}

	movementType := c.Query("movement_type")
	if movementType != "" {
		conditions = append(conditions, "im.movement_type = $"+strconv.Itoa(argCounter))
		args = append(args, movementType)
		argCounter++
	}

	startDateStr := c.Query("start_date") // YYYY-MM-DD
	endDateStr := c.Query("end_date")     // YYYY-MM-DD

	if startDateStr != "" {
		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err == nil {
			conditions = append(conditions, "im.movement_date >= $"+strconv.Itoa(argCounter))
			args = append(args, startDate)
			argCounter++
		}
	}
	if endDateStr != "" {
		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err == nil {
			conditions = append(conditions, "im.movement_date <= $"+strconv.Itoa(argCounter))
			args = append(args, endDate.AddDate(0,0,1).Add(-time.Nanosecond)) // End of day
			argCounter++
		}
	}

	if len(conditions) > 0 {
		baseQuery += " WHERE " + string(join(conditions, " AND ")) // join function from inventory_handlers.go or define locally
	}
	baseQuery += " ORDER BY im.movement_date DESC, im.id DESC"

	rows, err := db.Query(baseQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch inventory movements: " + err.Error()})
		return
	}
	defer rows.Close()

	movements := []models.InventoryMovement{}
	for rows.Next() {
		var mv models.InventoryMovement
		var itemName, itemSKU, staffName sql.NullString
		if err := rows.Scan(
			&mv.ID, &mv.PricelistItemID, &mv.StaffID, &mv.MovementType, &mv.QuantityChanged,
			&mv.Reason, &mv.MovementDate, &mv.CreatedAt, &mv.UpdatedAt,
			&itemName, &itemSKU, &staffName,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan inventory movement: " + err.Error()})
			return
		}
		mv.PricelistItem = &models.PricelistItem{ID: mv.PricelistItemID}
		if itemName.Valid { mv.PricelistItem.Name = itemName.String }
		if itemSKU.Valid { mv.PricelistItem.SKU = &itemSKU.String }
		
		if mv.StaffID != nil {
			mv.StaffMember = &models.StaffMember{ID: *mv.StaffID}
			if staffName.Valid { mv.StaffMember.User = &models.User{FullName: &staffName.String} }
		}
		movements = append(movements, mv)
	}
	c.JSON(http.StatusOK, movements)
}

// Note: GetInventoryMovementByID, UpdateInventoryMovement, DeleteInventoryMovement might not be typical
// operations as movements are usually immutable records. Updates/deletions might be handled by creating
// counter-movements (adjustments). If direct CRUD is needed, implement similarly to other handlers.

// Helper function to join strings (if not already in a shared utils package)
// func join(s []string, sep string) string { ... } // Already defined in inventory_handlers.go and table_booking_handlers.go
// It's better to move this to a pkg/utils file to avoid redefinition.


