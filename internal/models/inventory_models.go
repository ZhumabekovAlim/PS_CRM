package models

import "time"

// PricelistCategory represents a category for pricelist items
type PricelistCategory struct {
	ID          int64     `json:"id" db:"id"`
	Name        string    `json:"name" db:"name" binding:"required"`
	Description *string   `json:"description,omitempty" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// PricelistItem represents an item in the pricelist (generic for bar, hookah, snacks, services)
type PricelistItem struct {
	ID                int64     `json:"id" db:"id"`
	CategoryID        int64     `json:"category_id" db:"category_id" binding:"required"`
	Name              string    `json:"name" db:"name" binding:"required"`
	Description       *string   `json:"description,omitempty" db:"description"`
	Price             float64   `json:"price" db:"price" binding:"required,gt=0"`
	SKU               *string   `json:"sku,omitempty" db:"sku"`
	IsAvailable       bool      `json:"is_available" db:"is_available"`
	ItemType          string    `json:"item_type" db:"item_type" binding:"required"` // e.g., BAR, HOOKAH, SNACK, SERVICE
	TracksStock       bool      `json:"tracks_stock" db:"tracks_stock"`             // Whether this item's stock is tracked
	CurrentStock      *int      `json:"current_stock,omitempty" db:"current_stock"` // Nullable for items that don't track stock or if stock is not yet set
	LowStockThreshold *int      `json:"low_stock_threshold,omitempty" db:"low_stock_threshold"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
	Category          *PricelistCategory `json:"category,omitempty"` // For joining with Category
}

// InventoryMovement represents a change in stock for an item
type InventoryMovement struct {
	ID              int64     `json:"id" db:"id"`
	PricelistItemID int64     `json:"pricelist_item_id" db:"pricelist_item_id" binding:"required"`
	StaffID         *int64    `json:"staff_id,omitempty" db:"staff_id"`
	MovementType    string    `json:"movement_type" db:"movement_type" binding:"required"` // e.g., purchase, sale, adjustment_in, adjustment_out, spoilage
	QuantityChanged int       `json:"quantity_changed" db:"quantity_changed" binding:"required"`
	Reason          *string   `json:"reason,omitempty" db:"reason"`
	MovementDate    time.Time `json:"movement_date" db:"movement_date"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
	PricelistItem   *PricelistItem `json:"pricelist_item,omitempty"`
	StaffMember     *StaffMember   `json:"staff_member,omitempty"`
}


