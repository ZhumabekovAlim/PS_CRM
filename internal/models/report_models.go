package models

import "time"

// SalesReportItem represents a single item in a sales report.
// This could be aggregated by day, week, item, category, etc.
type SalesReportItem struct {
	Date        string  `json:"date,omitempty"` // e.g., YYYY-MM-DD or YYYY-WW or YYYY-MM
	ItemID      *int64  `json:"item_id,omitempty"`
	ItemName    *string `json:"item_name,omitempty"`
	CategoryID  *int64  `json:"category_id,omitempty"`
	CategoryName *string `json:"category_name,omitempty"`
	TotalQuantity int     `json:"total_quantity"`
	TotalSales    float64 `json:"total_sales"`
	TotalDiscount float64 `json:"total_discount,omitempty"`
	NetSales      float64 `json:"net_sales"`
}

// BookingReportItem represents data for booking reports.
// e.g., occupancy per table, popular times.
type BookingReportItem struct {
	TableID       *int64  `json:"table_id,omitempty"`
	TableName     *string `json:"table_name,omitempty"`
	Date          string  `json:"date,omitempty"`      // YYYY-MM-DD
	Hour          *int    `json:"hour,omitempty"`       // 0-23
	BookingsCount int     `json:"bookings_count"`
	TotalHours    float64 `json:"total_hours_booked"` // Total duration booked for this table/hour
	OccupancyRate *float64 `json:"occupancy_rate,omitempty"` // If applicable for a time slot
}

// InventoryReportItem represents data for inventory reports.
// e.g., low stock items, stock levels.
type InventoryReportItem struct {
	ItemID            int64   `json:"item_id"`
	ItemName          string  `json:"item_name"`
	SKU               *string `json:"sku,omitempty"`
	CategoryID        *int64  `json:"category_id,omitempty"`
	CategoryName      *string `json:"category_name,omitempty"`
	CurrentStock      int     `json:"current_stock"`
	LowStockThreshold *int    `json:"low_stock_threshold,omitempty"`
	LastMovementDate  *time.Time `json:"last_movement_date,omitempty"`
	Status            string  `json:"status,omitempty"` // e.g., "Low Stock", "In Stock", "Out of Stock"
}

// DashboardSummary holds key metrics for the dashboard.
type DashboardSummary struct {
	ActiveBookingsCount   int     `json:"active_bookings_count"`
	PendingOrdersCount    int     `json:"pending_orders_count"`
	TotalSalesToday       float64 `json:"total_sales_today"`
	TotalSalesThisWeek    float64 `json:"total_sales_this_week"`
	TotalSalesThisMonth   float64 `json:"total_sales_this_month"`
	LowStockItemsCount    int     `json:"low_stock_items_count"`
	UpcomingBookingsCount int     `json:"upcoming_bookings_count"` // e.g., for next 24 hours
}

// ReportRequestParams holds common parameters for requesting reports.
type ReportRequestParams struct {
	StartDate   string `form:"start_date"` // YYYY-MM-DD
	EndDate     string `form:"end_date"`   // YYYY-MM-DD
	Period      string `form:"period"`     // e.g., "daily", "weekly", "monthly", "custom"
	ItemID      *int64 `form:"item_id"`
	CategoryID  *int64 `form:"category_id"`
	TableID     *int64 `form:"table_id"`
	StaffID     *int64 `form:"staff_id"`
	Granularity string `form:"granularity"` // e.g., "hourly", "daily" for booking reports
}

