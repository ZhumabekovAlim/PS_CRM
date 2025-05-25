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

const HookahItemType = "HOOKAH"

// CreateHookahItem handles creation of a new hookah item (a PricelistItem with type 'HOOKAH')
func CreateHookahItem(c *gin.Context) {
	var item models.PricelistItem
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	item.ItemType = HookahItemType // Ensure item type is HOOKAH

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create hookah item: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

// GetHookahItems handles fetching all hookah items (PricelistItems with type 'HOOKAH')
func GetHookahItems(c *gin.Context) {
	db := database.GetDB()
	
	queryStr := `SELECT pi.id, pi.category_id, pi.name, pi.description, pi.price, pi.sku, 
	                     pi.is_available, pi.item_type, pi.current_stock, pi.low_stock_threshold, 
	                     pi.created_at, pi.updated_at, pc.name as category_name
	              FROM pricelist_items pi
	              JOIN pricelist_categories pc ON pi.category_id = pc.id
	              WHERE pi.item_type = $1
	              ORDER BY pi.name`

	rows, err := db.Query(queryStr, HookahItemType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch hookah items: " + err.Error()})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan hookah item: " + err.Error()})
			return
		}
		item.Category = &models.PricelistCategory{ID: item.CategoryID, Name: categoryName}
		items = append(items, item)
	}
	c.JSON(http.StatusOK, items)
}

// GetHookahItemByID handles fetching a single hookah item by ID
func GetHookahItemByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid hookah item ID"})
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
	err = db.QueryRow(query, id, HookahItemType).Scan(
		&item.ID, &item.CategoryID, &item.Name, &item.Description, &item.Price, &item.SKU, 
		&item.IsAvailable, &item.ItemType, &item.CurrentStock, &item.LowStockThreshold, 
		&item.CreatedAt, &item.UpdatedAt, &categoryName,
	)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Hookah item not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch hookah item: " + err.Error()})
		return
	}
	item.Category = &models.PricelistCategory{ID: item.CategoryID, Name: categoryName}
	c.JSON(http.StatusOK, item)
}

// UpdateHookahItem handles updating an existing hookah item
func UpdateHookahItem(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid hookah item ID"})
		return
	}

	var item models.PricelistItem
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	item.ItemType = HookahItemType // Ensure item type remains HOOKAH

	db := database.GetDB()
	// Check if the item to update is indeed a HOOKAH item
	var currentItemType string
	if err := db.QueryRow("SELECT item_type FROM pricelist_items WHERE id = $1", id).Scan(&currentItemType); err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Hookah item not found to update"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify hookah item type: " + err.Error()})
		return
	}
	if currentItemType != HookahItemType {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot update item: it is not a hookah item"})
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

	if err == sql.ErrNoRows { // Should be caught by pre-check
		c.JSON(http.StatusNotFound, gin.H{"error": "Hookah item not found to update (race condition?)"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update hookah item: " + err.Error()})
		return
	}
	item.ID = id // Ensure ID from path is used
	c.JSON(http.StatusOK, item)
}

// DeleteHookahItem handles deleting a hookah item
func DeleteHookahItem(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid hookah item ID"})
		return
	}

	db := database.GetDB()

	// Check if the item to delete is indeed a HOOKAH item
	var currentItemType string
	if err := db.QueryRow("SELECT item_type FROM pricelist_items WHERE id = $1", id).Scan(&currentItemType); err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Hookah item not found to delete"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify hookah item type: " + err.Error()})
		return
	}
	if currentItemType != HookahItemType {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete item: it is not a hookah item"})
		return
	}

	result, err := db.Exec("DELETE FROM pricelist_items WHERE id = $1 AND item_type = $2", id, HookahItemType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete hookah item: " + err.Error()})
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Hookah item not found to delete (or was not a hookah item)"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Hookah item deleted successfully"})
}

