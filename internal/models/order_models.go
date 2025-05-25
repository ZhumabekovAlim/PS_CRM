package models

import "time"

// Order represents a customer's order.
type Order struct {
	ID             int64      `json:"id" db:"id"`
	ClientID       *int64     `json:"client_id,omitempty" db:"client_id"`
	BookingID      *int64     `json:"booking_id,omitempty" db:"booking_id"`
	StaffID        *int64     `json:"staff_id,omitempty" db:"staff_id"` // UserID of the staff member who took/processed the order
	TableID        *int64     `json:"table_id,omitempty" db:"table_id"` // Optional, if order is associated with a table
	OrderTime      time.Time  `json:"order_time" db:"order_time"`
	Status         string     `json:"status" db:"status"` // e.g., pending, completed, cancelled, preparing, ready, served, paid
	TotalAmount    float64    `json:"total_amount" db:"total_amount"`
	DiscountAmount *float64   `json:"discount_amount,omitempty" db:"discount_amount"`
	FinalAmount    float64    `json:"final_amount" db:"final_amount"`
	PaymentMethod  *string    `json:"payment_method,omitempty" db:"payment_method"`
	Notes          *string    `json:"notes,omitempty" db:"notes"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`

	// Joined fields (populated by repository, not direct DB columns in 'orders' table)
	Client      *Client      `json:"client,omitempty"`
	GameTable   *GameTable   `json:"game_table,omitempty"`
	StaffMember *StaffMember `json:"staff_member,omitempty"` // Represents the staff profile
	OrderItems  []OrderItem  `json:"order_items,omitempty"`
}

// OrderItem represents an individual item within an order.
type OrderItem struct {
	ID              int64     `json:"id" db:"id"`
	OrderID         int64     `json:"order_id" db:"order_id"`
	PricelistItemID int64     `json:"pricelist_item_id" db:"pricelist_item_id"`
	Quantity        int       `json:"quantity" db:"quantity"`
	UnitPrice       float64   `json:"unit_price" db:"unit_price"` // Price at the time of order
	TotalPrice      float64   `json:"total_price" db:"total_price"`
	Notes           *string   `json:"notes,omitempty" db:"notes"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`

	// Joined fields
	PricelistItem *PricelistItem `json:"pricelist_item,omitempty"` // To get item name, SKU etc.
}

// OrderFilters defines the available filters for querying orders.
// This struct is used by both the service and repository layers.
type OrderFilters struct {
	ClientID *int64  `form:"client_id"`
	StaffID  *int64  `form:"staff_id"`
	TableID  *int64  `form:"table_id"`
	Status   *string `form:"status"`
	Date     *string `form:"date"` // Expected format YYYY-MM-DD
	Page     int     `form:"page"`
	PageSize int     `form:"page_size"`
}
