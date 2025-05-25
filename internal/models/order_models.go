package models

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

// Note: The main Order and OrderItem structs are already in `internal/models/models.go` (or similar).
// If they were in a different file, they'd be consolidated or this file might be named more generally like `filters.go` or `query_params.go`.
// For this task, we are only moving OrderFilters.
// The existing Order and OrderItem models in `internal/models/models.go` will be used by the new OrderRepository.
