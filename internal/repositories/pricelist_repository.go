package repositories

import (
	"database/sql"
	"errors"
	"fmt"
	"ps_club_backend/internal/models"
	"strings"
	"time"

	"github.com/lib/pq"
)

// PricelistRepository defines the interface for pricelist-related database operations.
type PricelistRepository interface {
	// PricelistCategory methods
	CreateCategory(executor SQLExecutor, category *models.PricelistCategory) (int64, error)
	GetCategoryByID(id int64) (*models.PricelistCategory, error)
	GetCategories(page, pageSize int) ([]models.PricelistCategory, int, error) // Returns categories, total count, error
	UpdateCategory(executor SQLExecutor, category *models.PricelistCategory) error
	DeleteCategory(executor SQLExecutor, id int64) error

	// PricelistItem methods
	CreateItem(executor SQLExecutor, item *models.PricelistItem) (int64, error)
	GetItemByID(id int64) (*models.PricelistItem, error) // Should join with category
	GetItems(categoryID *int64, itemType *string, page, pageSize int) ([]models.PricelistItem, int, error) // Returns items, total count, error. Joins with category.
	UpdateItem(executor SQLExecutor, item *models.PricelistItem) error
	DeleteItem(executor SQLExecutor, id int64) error
	UpdateStock(executor SQLExecutor, itemID int64, quantityChange int) (int, error) // Returns new stock level
	GetItemPriceAndStock(itemID int64) (price float64, currentStock sql.NullInt64, itemName string, tracksStock bool, err error) // Used by OrderService
}

type pricelistRepository struct {
	db *sql.DB
}

// NewPricelistRepository creates a new instance of PricelistRepository.
func NewPricelistRepository(db *sql.DB) PricelistRepository {
	return &pricelistRepository{db: db}
}

// --- PricelistCategory Methods ---

func (r *pricelistRepository) CreateCategory(executor SQLExecutor, category *models.PricelistCategory) (int64, error) {
	query := `INSERT INTO pricelist_categories (name, description, created_at, updated_at)
	          VALUES ($1, $2, $3, $4)
	          RETURNING id`
	currentTime := time.Now()
	err := executor.QueryRow(query, category.Name, category.Description, currentTime, currentTime).Scan(&category.ID)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code.Name() == "unique_violation" {
			return 0, fmt.Errorf("%w: pricelist category name '%s' already exists (constraint: %s)", ErrDuplicateKey, category.Name, pqErr.Constraint)
		}
		return 0, fmt.Errorf("%w: creating pricelist category: %v", ErrDatabaseError, err)
	}
	return category.ID, nil
}

func (r *pricelistRepository) GetCategoryByID(id int64) (*models.PricelistCategory, error) {
	category := &models.PricelistCategory{}
	query := `SELECT id, name, description, created_at, updated_at FROM pricelist_categories WHERE id = $1`
	err := r.db.QueryRow(query, id).Scan(&category.ID, &category.Name, &category.Description, &category.CreatedAt, &category.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("%w: getting pricelist category by ID %d: %v", ErrDatabaseError, id, err)
	}
	return category, nil
}

func (r *pricelistRepository) GetCategories(page, pageSize int) ([]models.PricelistCategory, int, error) {
	categories := []models.PricelistCategory{}
	totalCount := 0
	query := `SELECT id, name, description, created_at, updated_at, COUNT(*) OVER() AS total_count
	          FROM pricelist_categories
	          ORDER BY name
	          LIMIT $1 OFFSET $2`
	offset := (page - 1) * pageSize
	rows, err := r.db.Query(query, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: getting pricelist categories: %v", ErrDatabaseError, err)
	}
	defer rows.Close()

	for rows.Next() {
		var category models.PricelistCategory
		if err := rows.Scan(&category.ID, &category.Name, &category.Description, &category.CreatedAt, &category.UpdatedAt, &totalCount); err != nil {
			return nil, 0, fmt.Errorf("%w: scanning pricelist category: %v", ErrDatabaseError, err)
		}
		categories = append(categories, category)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("%w: iterating pricelist categories: %v", ErrDatabaseError, err)
	}
	return categories, totalCount, nil
}

func (r *pricelistRepository) UpdateCategory(executor SQLExecutor, category *models.PricelistCategory) error {
	query := `UPDATE pricelist_categories SET name = $1, description = $2, updated_at = $3 WHERE id = $4`
	result, err := executor.Exec(query, category.Name, category.Description, time.Now(), category.ID)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code.Name() == "unique_violation" {
			return fmt.Errorf("%w: pricelist category name '%s' already exists (constraint: %s)", ErrDuplicateKey, category.Name, pqErr.Constraint)
		}
		return fmt.Errorf("%w: updating pricelist category ID %d: %v", ErrDatabaseError, category.ID, err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *pricelistRepository) DeleteCategory(executor SQLExecutor, id int64) error {
	// Check if category is in use by any pricelist_items
	var count int
	checkQuery := "SELECT COUNT(*) FROM pricelist_items WHERE category_id = $1"
	err := executor.QueryRow(checkQuery, id).Scan(&count)
	if err != nil {
		return fmt.Errorf("%w: checking if category %d is in use: %v", ErrDatabaseError, id, err)
	}
	if count > 0 {
		return fmt.Errorf("%w: category ID %d cannot be deleted as it is currently in use by %d pricelist item(s)", ErrDatabaseError, id, count)
	}
	
	query := `DELETE FROM pricelist_categories WHERE id = $1`
	result, err := executor.Exec(query, id)
	if err != nil {
		return fmt.Errorf("%w: deleting pricelist category ID %d: %v", ErrDatabaseError, id, err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// --- PricelistItem Methods ---

func (r *pricelistRepository) CreateItem(executor SQLExecutor, item *models.PricelistItem) (int64, error) {
	query := `INSERT INTO pricelist_items 
	          (category_id, name, description, price, sku, is_available, item_type, tracks_stock, current_stock, low_stock_threshold, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	          RETURNING id`
	currentTime := time.Now()

	var currentStock sql.NullInt64
	if item.TracksStock && item.CurrentStock != nil {
		currentStock = sql.NullInt64{Int64: int64(*item.CurrentStock), Valid: true}
	} else if !item.TracksStock {
        currentStock = sql.NullInt64{Valid: false} // Ensure NULL if not tracking
    } else { // Tracks stock but CurrentStock is nil (e.g. initial creation without stock value)
		currentStock = sql.NullInt64{Valid: false} // Or default to 0: sql.NullInt64{Int64: 0, Valid: true}
	}


	var lowStockThreshold sql.NullInt64
	if item.TracksStock && item.LowStockThreshold != nil {
		lowStockThreshold = sql.NullInt64{Int64: int64(*item.LowStockThreshold), Valid: true}
	} else if !item.TracksStock {
        lowStockThreshold = sql.NullInt64{Valid: false} // Ensure NULL if not tracking
    }

	err := executor.QueryRow(query,
		item.CategoryID, item.Name, item.Description, item.Price, item.SKU, item.IsAvailable,
		item.ItemType, item.TracksStock, currentStock, lowStockThreshold, currentTime, currentTime,
	).Scan(&item.ID)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code.Name() == "unique_violation" {
				return 0, fmt.Errorf("%w: creating pricelist item (constraint: %s): %v", ErrDuplicateKey, pqErr.Constraint, err)
			}
			if pqErr.Code.Name() == "foreign_key_violation" && pqErr.Constraint == "pricelist_items_category_id_fkey" {
				return 0, fmt.Errorf("%w: invalid category_id %d (constraint: %s): %v", ErrDatabaseError, item.CategoryID, pqErr.Constraint, err)
			}
		}
		return 0, fmt.Errorf("%w: creating pricelist item: %v", ErrDatabaseError, err)
	}
	return item.ID, nil
}

func (r *pricelistRepository) GetItemByID(id int64) (*models.PricelistItem, error) {
	item := &models.PricelistItem{}
	category := &models.PricelistCategory{}

	query := `SELECT 
	            pi.id, pi.category_id, pi.name, pi.description, pi.price, pi.sku, 
	            pi.is_available, pi.item_type, pi.tracks_stock, pi.current_stock, pi.low_stock_threshold, 
	            pi.created_at, pi.updated_at,
	            pc.id as cat_id, pc.name as cat_name, pc.description as cat_desc, 
	            pc.created_at as cat_created_at, pc.updated_at as cat_updated_at
	          FROM pricelist_items pi
	          JOIN pricelist_categories pc ON pi.category_id = pc.id
	          WHERE pi.id = $1`

	var currentStock sql.NullInt64
	var lowStockThreshold sql.NullInt64

	err := r.db.QueryRow(query, id).Scan(
		&item.ID, &item.CategoryID, &item.Name, &item.Description, &item.Price, &item.SKU,
		&item.IsAvailable, &item.ItemType, &item.TracksStock, &currentStock, &lowStockThreshold,
		&item.CreatedAt, &item.UpdatedAt,
		&category.ID, &category.Name, &category.Description, &category.CreatedAt, &category.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("%w: getting pricelist item by ID %d: %v", ErrDatabaseError, id, err)
	}

	if currentStock.Valid {
		val := int(currentStock.Int64)
		item.CurrentStock = &val
	}
	if lowStockThreshold.Valid {
		val := int(lowStockThreshold.Int64)
		item.LowStockThreshold = &val
	}
	item.Category = category
	return item, nil
}

func (r *pricelistRepository) GetItems(categoryID *int64, itemType *string, page, pageSize int) ([]models.PricelistItem, int, error) {
	items := []models.PricelistItem{}
	totalCount := 0

	var queryBuilder strings.Builder
	queryBuilder.WriteString(`SELECT 
	    pi.id, pi.category_id, pi.name, pi.description, pi.price, pi.sku, 
	    pi.is_available, pi.item_type, pi.tracks_stock, pi.current_stock, pi.low_stock_threshold, 
	    pi.created_at, pi.updated_at,
	    pc.id as cat_id, pc.name as cat_name, pc.description as cat_desc, 
	    pc.created_at as cat_created_at, pc.updated_at as cat_updated_at,
	    COUNT(*) OVER() AS total_count
	  FROM pricelist_items pi
	  JOIN pricelist_categories pc ON pi.category_id = pc.id`)

	var conditions []string
	var args []interface{}
	argCount := 1

	if categoryID != nil {
		conditions = append(conditions, fmt.Sprintf("pi.category_id = $%d", argCount))
		args = append(args, *categoryID)
		argCount++
	}
	if itemType != nil && *itemType != "" {
		conditions = append(conditions, fmt.Sprintf("pi.item_type = $%d", argCount))
		args = append(args, *itemType)
		argCount++
	}

	if len(conditions) > 0 {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(strings.Join(conditions, " AND "))
	}

	queryBuilder.WriteString(" ORDER BY pi.name") // Consider making order configurable
	queryBuilder.WriteString(fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1))
	args = append(args, pageSize, (page-1)*pageSize)

	rows, err := r.db.Query(queryBuilder.String(), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: getting pricelist items: %v", ErrDatabaseError, err)
	}
	defer rows.Close()

	for rows.Next() {
		var item models.PricelistItem
		var category models.PricelistCategory
		var currentStock sql.NullInt64
		var lowStockThreshold sql.NullInt64

		if err := rows.Scan(
			&item.ID, &item.CategoryID, &item.Name, &item.Description, &item.Price, &item.SKU,
			&item.IsAvailable, &item.ItemType, &item.TracksStock, &currentStock, &lowStockThreshold,
			&item.CreatedAt, &item.UpdatedAt,
			&category.ID, &category.Name, &category.Description, &category.CreatedAt, &category.UpdatedAt,
			&totalCount,
		); err != nil {
			return nil, 0, fmt.Errorf("%w: scanning pricelist item: %v", ErrDatabaseError, err)
		}
		if currentStock.Valid {
			val := int(currentStock.Int64)
			item.CurrentStock = &val
		}
		if lowStockThreshold.Valid {
			val := int(lowStockThreshold.Int64)
			item.LowStockThreshold = &val
		}
		item.Category = &category
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("%w: iterating pricelist items: %v", ErrDatabaseError, err)
	}
	// totalCount is set from the first successfully scanned row. If no rows, it remains 0.
	return items, totalCount, nil
}

func (r *pricelistRepository) UpdateItem(executor SQLExecutor, item *models.PricelistItem) error {
	query := `UPDATE pricelist_items SET 
	            category_id = $1, name = $2, description = $3, price = $4, sku = $5, 
	            is_available = $6, item_type = $7, tracks_stock = $8, current_stock = $9, 
	            low_stock_threshold = $10, updated_at = $11 
	          WHERE id = $12`

	var currentStock sql.NullInt64
	if item.TracksStock && item.CurrentStock != nil {
		currentStock = sql.NullInt64{Int64: int64(*item.CurrentStock), Valid: true}
	} else if !item.TracksStock {
         currentStock = sql.NullInt64{Valid: false}
    } else { // Tracks stock but CurrentStock is nil
		currentStock = sql.NullInt64{Valid: false} // Or default to 0
	}

	var lowStockThreshold sql.NullInt64
	if item.TracksStock && item.LowStockThreshold != nil {
		lowStockThreshold = sql.NullInt64{Int64: int64(*item.LowStockThreshold), Valid: true}
	} else if !item.TracksStock {
        lowStockThreshold = sql.NullInt64{Valid: false}
    }

	result, err := executor.Exec(query,
		item.CategoryID, item.Name, item.Description, item.Price, item.SKU,
		item.IsAvailable, item.ItemType, item.TracksStock, currentStock, lowStockThreshold,
		time.Now(), item.ID,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code.Name() == "unique_violation" {
				return fmt.Errorf("%w: updating pricelist item (constraint: %s): %v", ErrDuplicateKey, pqErr.Constraint, err)
			}
			if pqErr.Code.Name() == "foreign_key_violation" && pqErr.Constraint == "pricelist_items_category_id_fkey" {
				return fmt.Errorf("%w: invalid category_id %d (constraint: %s): %v", ErrDatabaseError, item.CategoryID, pqErr.Constraint, err)
			}
		}
		return fmt.Errorf("%w: updating pricelist item ID %d: %v", ErrDatabaseError, item.ID, err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *pricelistRepository) DeleteItem(executor SQLExecutor, id int64) error {
	query := `DELETE FROM pricelist_items WHERE id = $1`
	result, err := executor.Exec(query, id)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23503" { // foreign_key_violation
			return fmt.Errorf("%w: item ID %d cannot be deleted as it is referenced by other records (e.g., orders, inventory movements) (constraint: %s)", ErrDatabaseError, id, pqErr.Constraint)
		}
		return fmt.Errorf("%w: deleting pricelist item ID %d: %v", ErrDatabaseError, id, err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *pricelistRepository) UpdateStock(executor SQLExecutor, itemID int64, quantityChange int) (int, error) {
	var newStock sql.NullInt64 // Use NullInt64 to handle cases where current_stock might be NULL
	query := `UPDATE pricelist_items 
	          SET current_stock = COALESCE(current_stock, 0) + $1, updated_at = $2 
	          WHERE id = $3 AND tracks_stock = TRUE
	          RETURNING current_stock`
	err := executor.QueryRow(query, quantityChange, time.Now(), itemID).Scan(&newStock)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			var tracksStockActual sql.NullBool
			checkErr := r.db.QueryRow("SELECT tracks_stock FROM pricelist_items WHERE id = $1", itemID).Scan(&tracksStockActual)
			if errors.Is(checkErr, sql.ErrNoRows) {
				return 0, ErrNotFound // Item does not exist
			}
			if checkErr == nil && tracksStockActual.Valid && !tracksStockActual.Bool {
				return 0, fmt.Errorf("%w: stock not updated for item ID %d because it does not track stock", ErrDatabaseError, itemID)
			}
			// Other reasons for ErrNoRows from UPDATE (e.g., item exists but tracks_stock is false and was not caught by above)
			return 0, fmt.Errorf("%w: failed to update stock for item ID %d (item may not track stock or not exist): %v", ErrDatabaseError, itemID, err)
		}
		return 0, fmt.Errorf("%w: updating stock for item ID %d: %v", ErrDatabaseError, itemID, err)
	}
	if !newStock.Valid { // Should not happen if RETURNING current_stock and update was successful
	    return 0, fmt.Errorf("%w: stock update for item ID %d resulted in NULL stock, which is unexpected", ErrDatabaseError, itemID)
    }
	return int(newStock.Int64), nil
}

func (r *pricelistRepository) GetItemPriceAndStock(itemID int64) (float64, sql.NullInt64, string, bool, error) {
	var price float64
	var currentStock sql.NullInt64
	var name string
	var tracksStock bool
	query := `SELECT name, price, tracks_stock, current_stock FROM pricelist_items WHERE id = $1`
	err := r.db.QueryRow(query, itemID).Scan(&name, &price, &tracksStock, &currentStock)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, sql.NullInt64{}, "", false, ErrNotFound
		}
		return 0, sql.NullInt64{}, "", false, fmt.Errorf("%w: getting price and stock for item ID %d: %v", ErrDatabaseError, itemID, err)
	}
	return price, currentStock, name, tracksStock, nil
}
