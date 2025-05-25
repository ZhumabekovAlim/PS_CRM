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

// Pricelist Categories Handlers

// CreatePricelistCategory handles creation of a new pricelist category
func CreatePricelistCategory(c *gin.Context) {
	var category models.PricelistCategory
	if err := c.ShouldBindJSON(&category); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	db := database.GetDB()
	query := `INSERT INTO pricelist_categories (name, description, created_at, updated_at)
	          VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at`

	category.CreatedAt = time.Now()
	category.UpdatedAt = time.Now()

	err := db.QueryRow(query, category.Name, category.Description, category.CreatedAt, category.UpdatedAt).
		Scan(&category.ID, &category.CreatedAt, &category.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create pricelist category: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, category)
}

// GetPricelistCategories handles fetching all pricelist categories
func GetPricelistCategories(c *gin.Context) {
	db := database.GetDB()
	rows, err := db.Query("SELECT id, name, description, created_at, updated_at FROM pricelist_categories ORDER BY name")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch pricelist categories: " + err.Error()})
		return
	}
	defer rows.Close()

	categories := []models.PricelistCategory{}
	for rows.Next() {
		var cat models.PricelistCategory
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Description, &cat.CreatedAt, &cat.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan pricelist category: " + err.Error()})
			return
		}
		categories = append(categories, cat)
	}
	c.JSON(http.StatusOK, categories)
}

// GetPricelistCategoryByID handles fetching a single pricelist category by ID
func GetPricelistCategoryByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	db := database.GetDB()
	var cat models.PricelistCategory
	query := "SELECT id, name, description, created_at, updated_at FROM pricelist_categories WHERE id = $1"
	err = db.QueryRow(query, id).Scan(&cat.ID, &cat.Name, &cat.Description, &cat.CreatedAt, &cat.UpdatedAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pricelist category not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch pricelist category: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, cat)
}

// UpdatePricelistCategory handles updating an existing pricelist category
func UpdatePricelistCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	var category models.PricelistCategory
	if err := c.ShouldBindJSON(&category); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	db := database.GetDB()
	query := `UPDATE pricelist_categories SET name = $1, description = $2, updated_at = $3
	          WHERE id = $4 RETURNING id, name, description, created_at, updated_at`

	category.UpdatedAt = time.Now()

	err = db.QueryRow(query, category.Name, category.Description, category.UpdatedAt, id).
		Scan(&category.ID, &category.Name, &category.Description, &category.CreatedAt, &category.UpdatedAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pricelist category not found to update"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pricelist category: " + err.Error()})
		return
	}
	category.ID = id // Ensure the ID from path is used
	c.JSON(http.StatusOK, category)
}

// DeletePricelistCategory handles deleting a pricelist category
func DeletePricelistCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	db := database.GetDB()
	// Check if items are associated with this category first
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM pricelist_items WHERE category_id = $1", id).Scan(&count)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check associated items: " + err.Error()})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Cannot delete category: it has associated pricelist items. Please reassign or delete items first."})
		return
	}

	result, err := db.Exec("DELETE FROM pricelist_categories WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete pricelist category: " + err.Error()})
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pricelist category not found to delete"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Pricelist category deleted successfully"})
}

// Pricelist Items Handlers

// CreatePricelistItem handles creation of a new pricelist item
func CreatePricelistItem(c *gin.Context) {
	var item models.PricelistItem
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	db := database.GetDB()
	query := `INSERT INTO pricelist_items 
	          (category_id, name, description, price, sku, is_available, item_type, current_stock, low_stock_threshold, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) 
	          RETURNING id, created_at, updated_at`

	item.CreatedAt = time.Now()
	item.UpdatedAt = time.Now()
	if item.IsAvailable == false { // Default from DB is true, but if payload sets it, respect it.
        // No action needed, will be set by the query
    } else {
        item.IsAvailable = true // Ensure it's true if not specified or specified as true
    }

	err := db.QueryRow(query, 
		item.CategoryID, item.Name, item.Description, item.Price, item.SKU, item.IsAvailable, 
		item.ItemType, item.CurrentStock, item.LowStockThreshold, item.CreatedAt, item.UpdatedAt,
	).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create pricelist item: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

// GetPricelistItems handles fetching all pricelist items, optionally filtered by category_id or item_type
func GetPricelistItems(c *gin.Context) {
	db := database.GetDB()
	
	baseQuery := `SELECT pi.id, pi.category_id, pi.name, pi.description, pi.price, pi.sku, 
	                     pi.is_available, pi.item_type, pi.current_stock, pi.low_stock_threshold, 
	                     pi.created_at, pi.updated_at, pc.name as category_name
	              FROM pricelist_items pi
	              JOIN pricelist_categories pc ON pi.category_id = pc.id`
	
	var conditions []string
	var args []interface{}
	argCounter := 1

	categoryID := c.Query("category_id")
	if categoryID != "" {
		conditions = append(conditions, "pi.category_id = $" + strconv.Itoa(argCounter))
		args = append(args, categoryID)
		argCounter++
	}

	itemType := c.Query("item_type")
	if itemType != "" {
		conditions = append(conditions, "pi.item_type = $" + strconv.Itoa(argCounter))
		args = append(args, itemType)
		argCounter++
	}

	if len(conditions) > 0 {
		baseQuery += " WHERE " + string(join(conditions, " AND "))
	}
	baseQuery += " ORDER BY pi.name"

	rows, err := db.Query(baseQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch pricelist items: " + err.Error()})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan pricelist item: " + err.Error()})
			return
		}
		item.Category = &models.PricelistCategory{ID: item.CategoryID, Name: categoryName} // Populate nested category info
		items = append(items, item)
	}
	c.JSON(http.StatusOK, items)
}

// Helper function to join strings (needed because strings.Join needs a slice of strings)
func join(s []string, sep string) string {
    if len(s) == 0 {
        return ""
    }
    if len(s) == 1 {
        return s[0]
    }
    n := len(sep) * (len(s) - 1)
    for i := 0; i < len(s); i++ {
        n += len(s[i])
    }

    var b []byte
    bp := 0
    b = make([]byte, n)
    copy(b[bp:], s[0])
    bp += len(s[0])
    for _, s := range s[1:] {
        copy(b[bp:], sep)
        bp += len(sep)
        copy(b[bp:], s)
        bp += len(s)
    }
    return string(b)
}


// GetPricelistItemByID handles fetching a single pricelist item by ID
func GetPricelistItemByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
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
	          WHERE pi.id = $1`
	err = db.QueryRow(query, id).Scan(
		&item.ID, &item.CategoryID, &item.Name, &item.Description, &item.Price, &item.SKU, 
		&item.IsAvailable, &item.ItemType, &item.CurrentStock, &item.LowStockThreshold, 
		&item.CreatedAt, &item.UpdatedAt, &categoryName,
	)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pricelist item not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch pricelist item: " + err.Error()})
		return
	}
	item.Category = &models.PricelistCategory{ID: item.CategoryID, Name: categoryName}
	c.JSON(http.StatusOK, item)
}

// UpdatePricelistItem handles updating an existing pricelist item
func UpdatePricelistItem(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	var item models.PricelistItem
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	db := database.GetDB()
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

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pricelist item not found to update"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pricelist item: " + err.Error()})
		return
	}
	item.ID = id // Ensure ID from path is used
	c.JSON(http.StatusOK, item)
}

// DeletePricelistItem handles deleting a pricelist item
func DeletePricelistItem(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	db := database.GetDB()
	// Consider checking if item is in active orders or has recent inventory movements before deleting
	// For now, direct delete:
	result, err := db.Exec("DELETE FROM pricelist_items WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete pricelist item: " + err.Error()})
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pricelist item not found to delete"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Pricelist item deleted successfully"})
}

// TODO: Implement Inventory Movement Handlers (CRUD)

