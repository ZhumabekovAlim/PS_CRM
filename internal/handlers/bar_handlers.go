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

const BarItemType = "BAR"

// CreateBarItem handles creation of a new bar item (a PricelistItem with type 'BAR')
func CreateBarItem(c *gin.Context) {
	var item models.PricelistItem
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	item.ItemType = BarItemType // Ensure item type is BAR

	db := database.GetDB()
	query := `INSERT INTO pricelist_items 
	          (category_id, name, description, price, sku, is_available, item_type, current_stock, low_stock_threshold, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) 
	          RETURNING id, created_at, updated_at`

	item.CreatedAt = time.Now()
	item.UpdatedAt = time.Now()
	if item.IsAvailable == false {
        // Default from DB is true, but if payload sets it, respect it.
    } else {
        item.IsAvailable = true
    }

	err := db.QueryRow(query, 
		item.CategoryID, item.Name, item.Description, item.Price, item.SKU, item.IsAvailable, 
		item.ItemType, item.CurrentStock, item.LowStockThreshold, item.CreatedAt, item.UpdatedAt,
	).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)

	if err != nil {
		// Check for foreign key violation for category_id
		// pqErr, ok := err.(*pq.Error)
		// if ok && pqErr.Code == "23503" { // Foreign key violation
		// 	c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category_id"})
		// 	return
		// }
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create bar item: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

// GetBarItems handles fetching all bar items (PricelistItems with type 'BAR')
func GetBarItems(c *gin.Context) {
	db := database.GetDB()
	
	queryStr := `SELECT pi.id, pi.category_id, pi.name, pi.description, pi.price, pi.sku, 
	                     pi.is_available, pi.item_type, pi.current_stock, pi.low_stock_threshold, 
	                     pi.created_at, pi.updated_at, pc.name as category_name
	              FROM pricelist_items pi
	              JOIN pricelist_categories pc ON pi.category_id = pc.id
	              WHERE pi.item_type = $1
	              ORDER BY pi.name`

	rows, err := db.Query(queryStr, BarItemType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bar items: " + err.Error()})
		return
	}
	defer rows.Close()

	items := []models.PricelistItem{}
	for rows.Next() {
		var item models.PricelistItem
		var categoryName string
		if err := rows.Scan(
			&item.ID, &item.CategoryID, &item.Name, &item.Description, &item.Price, &item.SKU, 
			&item.IsAvailable, &item.ItemType, &item.CurrentStock, &item.LowStockThreshold, 
			&item.CreatedAt, &item.UpdatedAt, &categoryName,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan bar item: " + err.Error()})
			return
		}
		item.Category = &models.PricelistCategory{ID: item.CategoryID, Name: categoryName}
		items = append(items, item)
	}
	c.JSON(http.StatusOK, items)
}

// GetBarItemByID handles fetching a single bar item by ID
func GetBarItemByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid bar item ID"})
		return
	}

	db := database.GetDB()
	var item models.PricelistItem
	var categoryName string
	query := `SELECT pi.id, pi.category_id, pi.name, pi.description, pi.price, pi.sku, 
	                 pi.is_available, pi.item_type, pi.current_stock, pi.low_stock_threshold, 
	                 pi.created_at, pi.updated_at, pc.name as category_name
	          FROM pricelist_items pi
	          JOIN pricelist_categories pc ON pi.category_id = pc.id
	          WHERE pi.id = $1 AND pi.item_type = $2`
	err = db.QueryRow(query, id, BarItemType).Scan(
		&item.ID, &item.CategoryID, &item.Name, &item.Description, &item.Price, &item.SKU, 
		&item.IsAvailable, &item.ItemType, &item.CurrentStock, &item.LowStockThreshold, 
		&item.CreatedAt, &item.UpdatedAt, &categoryName,
	)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bar item not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bar item: " + err.Error()})
		return
	}
	item.Category = &models.PricelistCategory{ID: item.CategoryID, Name: categoryName}
	c.JSON(http.StatusOK, item)
}

// UpdateBarItem handles updating an existing bar item
func UpdateBarItem(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid bar item ID"})
		return
	}

	var item models.PricelistItem
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	item.ItemType = BarItemType // Ensure item type remains BAR

	db := database.GetDB()
	// Check if the item to update is indeed a BAR item
	var currentItemType string
	if err := db.QueryRow("SELECT item_type FROM pricelist_items WHERE id = $1", id).Scan(&currentItemType); err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bar item not found to update"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify bar item type: " + err.Error()})
		return
	}
	if currentItemType != BarItemType {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot update item: it is not a bar item"})
		return
	}

	query := `UPDATE pricelist_items SET 
	          category_id = $1, name = $2, description = $3, price = $4, sku = $5, 
	          is_available = $6, item_type = $7, current_stock = $8, low_stock_threshold = $9, updated_at = $10
	          WHERE id = $11 
	          RETURNING id, category_id, name, description, price, sku, is_available, item_type, current_stock, low_stock_threshold, created_at, updated_at`

	item.UpdatedAt = time.Now()

	err = db.QueryRow(query, 
		item.CategoryID, item.Name, item.Description, item.Price, item.SKU, 
		item.IsAvailable, item.ItemType, item.CurrentStock, item.LowStockThreshold, item.UpdatedAt, id,
	).Scan(
		&item.ID, &item.CategoryID, &item.Name, &item.Description, &item.Price, &item.SKU, 
		&item.IsAvailable, &item.ItemType, &item.CurrentStock, &item.LowStockThreshold, 
		&item.CreatedAt, &item.UpdatedAt,
	)

	if err == sql.ErrNoRows { // Should be caught by pre-check, but good to have
		c.JSON(http.StatusNotFound, gin.H{"error": "Bar item not found to update (race condition?)"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update bar item: " + err.Error()})
		return
	}
	item.ID = id // Ensure ID from path is used
	c.JSON(http.StatusOK, item)
}

// DeleteBarItem handles deleting a bar item
func DeleteBarItem(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid bar item ID"})
		return
	}

	db := database.GetDB()

	// Check if the item to delete is indeed a BAR item
	var currentItemType string
	if err := db.QueryRow("SELECT item_type FROM pricelist_items WHERE id = $1", id).Scan(&currentItemType); err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bar item not found to delete"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify bar item type: " + err.Error()})
		return
	}
	if currentItemType != BarItemType {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete item: it is not a bar item"})
		return
	}

	// Consider checking if item is in active orders or has recent inventory movements before deleting
	result, err := db.Exec("DELETE FROM pricelist_items WHERE id = $1 AND item_type = $2", id, BarItemType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete bar item: " + err.Error()})
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bar item not found to delete (or was not a bar item)"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Bar item deleted successfully"})
}

