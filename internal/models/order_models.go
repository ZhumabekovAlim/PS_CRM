package models

import "time"

// Order represents a customer order for products/services
type Order struct {
	ID             int64      `json:"id" db:"id"`
	ClientID       *int64     `json:"client_id,omitempty" db:"client_id"`
	BookingID      *int64     `json:"booking_id,omitempty" db:"booking_id"`
	StaffID        *int64     `json:"staff_id,omitempty" db:"staff_id"`
	TableID        *int64     `json:"table_id,omitempty" db:"table_id"` // Table where order was placed (if applicable, e.g., for food/drinks at a game table)
	OrderTime      time.Time  `json:"order_time" db:"order_time"`
	Status         string     `json:"status" db:"status"` // e.g., pending, preparing, completed, paid, cancelled
	TotalAmount    float64    `json:"total_amount" db:"total_amount"`
	DiscountAmount float64    `json:"discount_amount" db:"discount_amount"`
	FinalAmount    float64    `json:"final_amount" db:"final_amount"`
	PaymentMethod  *string    `json:"payment_method,omitempty" db:"payment_method"`
	Notes          *string    `json:"notes,omitempty" db:"notes"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
	OrderItems     []OrderItem `json:"order_items,omitempty"` // Nested order items
	Client         *Client     `json:"client,omitempty"`
	Booking        *Booking    `json:"booking,omitempty"`
	StaffMember    *StaffMember `json:"staff_member,omitempty"`
	GameTable      *GameTable  `json:"game_table,omitempty"`
}

// OrderItem represents an individual item within an order
type OrderItem struct {
	ID              int64     `json:"id" db:"id"`
	OrderID         int64     `json:"order_id" db:"order_id" binding:"required"`
	PricelistItemID int64     `json:"pricelist_item_id" db:"pricelist_item_id" binding:"required"`
	Quantity        int       `json:"quantity" db:"quantity" binding:"required,gt=0"`
	UnitPrice       float64   `json:"unit_price" db:"unit_price" binding:"required,gte=0"` // Price at the time of order
	TotalPrice      float64   `json:"total_price" db:"total_price" binding:"required,gte=0"` // quantity * unit_price
	Notes           *string   `json:"notes,omitempty" db:"notes"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
	PricelistItem   *PricelistItem `json:"pricelist_item,omitempty"` // For joining with PricelistItem details
}

