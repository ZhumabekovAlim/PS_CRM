package repositories

import (
	"database/sql"
	"errors"
	"fmt"
	"ps_club_backend/internal/models"
	"strings"
	"time"
)

// InventoryMovementRepository defines the interface for inventory movement-related database operations.
type InventoryMovementRepository interface {
	CreateMovement(executor SQLExecutor, movement *models.InventoryMovement) (int64, error)
	GetMovements(itemID *int64, staffID *int64, movementType *string, page, pageSize int) ([]models.InventoryMovement, int, error)
}

type inventoryMovementRepository struct {
	db *sql.DB
}

// NewInventoryMovementRepository creates a new instance of InventoryMovementRepository.
func NewInventoryMovementRepository(db *sql.DB) InventoryMovementRepository {
	return &inventoryMovementRepository{db: db}
}

func (r *inventoryMovementRepository) CreateMovement(executor SQLExecutor, movement *models.InventoryMovement) (int64, error) {
	query := `INSERT INTO inventory_movements 
	          (pricelist_item_id, staff_id, movement_type, quantity_changed, reason, movement_date, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	          RETURNING id`
	currentTime := time.Now()
	if movement.MovementDate.IsZero() { // Default movement_date to current time if not provided
		movement.MovementDate = currentTime
	}

	var staffID sql.NullInt64
	if movement.StaffID != nil {
		staffID = sql.NullInt64{Int64: *movement.StaffID, Valid: true}
	}

	err := executor.QueryRow(query,
		movement.PricelistItemID, staffID, movement.MovementType, movement.QuantityChanged,
		movement.Reason, movement.MovementDate, currentTime, currentTime,
	).Scan(&movement.ID)

	if err != nil {
		// Handle foreign key violations (e.g., pricelist_item_id or staff_id does not exist)
		// or other specific errors if necessary
		return 0, fmt.Errorf("%w: creating inventory movement: %v", ErrDatabaseError, err)
	}
	return movement.ID, nil
}

func (r *inventoryMovementRepository) GetMovements(itemID *int64, staffID *int64, movementType *string, page, pageSize int) ([]models.InventoryMovement, int, error) {
	movements := []models.InventoryMovement{}
	totalCount := 0

	var queryBuilder strings.Builder
	queryBuilder.WriteString(`SELECT 
	    im.id, im.pricelist_item_id, im.staff_id, im.movement_type, im.quantity_changed, 
	    im.reason, im.movement_date, im.created_at, im.updated_at,
	    pi.name as item_name, pi.sku as item_sku, pi.item_type as item_item_type, pi.tracks_stock as item_tracks_stock,
	    u.full_name as staff_name,
	    COUNT(*) OVER() AS total_count
	  FROM inventory_movements im
	  JOIN pricelist_items pi ON im.pricelist_item_id = pi.id
	  LEFT JOIN staff_members sm ON im.staff_id = sm.id
	  LEFT JOIN users u ON sm.user_id = u.id`)

	var conditions []string
	var args []interface{}
	argCount := 1

	if itemID != nil {
		conditions = append(conditions, fmt.Sprintf("im.pricelist_item_id = $%d", argCount))
		args = append(args, *itemID)
		argCount++
	}
	if staffID != nil {
		conditions = append(conditions, fmt.Sprintf("im.staff_id = $%d", argCount))
		args = append(args, *staffID)
		argCount++
	}
	if movementType != nil && *movementType != "" {
		conditions = append(conditions, fmt.Sprintf("im.movement_type = $%d", argCount))
		args = append(args, *movementType)
		argCount++
	}

	if len(conditions) > 0 {
		queryBuilder.WriteString(" WHERE ")
		queryBuilder.WriteString(strings.Join(conditions, " AND "))
	}

	queryBuilder.WriteString(" ORDER BY im.movement_date DESC, im.created_at DESC")
	queryBuilder.WriteString(fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1))
	args = append(args, pageSize, (page-1)*pageSize)

	rows, err := r.db.Query(queryBuilder.String(), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: getting inventory movements: %v", ErrDatabaseError, err)
	}
	defer rows.Close()

	for rows.Next() {
		var movement models.InventoryMovement
		var pricelistItem models.PricelistItem
		var staffMember models.StaffMember
		var user models.User 

		var itemName, itemSKU, itemItemType, staffName sql.NullString
		var itemTracksStock sql.NullBool
		var scannedStaffID sql.NullInt64


		if err := rows.Scan(
			&movement.ID, &movement.PricelistItemID, &scannedStaffID, &movement.MovementType, &movement.QuantityChanged,
			&movement.Reason, &movement.MovementDate, &movement.CreatedAt, &movement.UpdatedAt,
			&itemName, &itemSKU, &itemItemType, &itemTracksStock,
			&staffName,
			&totalCount,
		); err != nil {
			return nil, 0, fmt.Errorf("%w: scanning inventory movement: %v", ErrDatabaseError, err)
		}

		pricelistItem.ID = movement.PricelistItemID
		if itemName.Valid { pricelistItem.Name = itemName.String }
		if itemSKU.Valid { sku := itemSKU.String; pricelistItem.SKU = &sku }
		if itemItemType.Valid { pricelistItem.ItemType = itemItemType.String }
		if itemTracksStock.Valid { pricelistItem.TracksStock = itemTracksStock.Bool }
		movement.PricelistItem = &pricelistItem

		if scannedStaffID.Valid {
			movement.StaffID = &scannedStaffID.Int64
			staffMember.ID = *movement.StaffID
			if staffName.Valid {
				name := staffName.String
				user.FullName = &name 
			}
			staffMember.User = &user 
			movement.StaffMember = &staffMember
		} else {
			movement.StaffID = nil
			movement.StaffMember = nil
		}
		
		movements = append(movements, movement)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("%w: iterating inventory movements: %v", ErrDatabaseError, err)
	}

	return movements, totalCount, nil
}
