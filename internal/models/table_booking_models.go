package models

import "time"

// BookingStatus defines the type for booking statuses
type BookingStatus string

const (
	BookingStatusConfirmed  BookingStatus = "confirmed"
	BookingStatusCancelled  BookingStatus = "cancelled"
	BookingStatusCompleted  BookingStatus = "completed"
	BookingStatusNoShow     BookingStatus = "no-show"
	BookingStatusPending    BookingStatus = "pending" // Added as a common initial state
	// Add other statuses if they are used or anticipated
)

// IsValidBookingStatus checks if the provided status string is a valid BookingStatus.
func IsValidBookingStatus(status string) bool {
	s := BookingStatus(status) // Convert string to BookingStatus type for comparison
	switch s {
	case BookingStatusConfirmed,
		 BookingStatusCancelled,
		 BookingStatusCompleted,
		 BookingStatusNoShow,
		 BookingStatusPending:
		return true
	default:
		return false
	}
}

// GameTable represents a physical table or console in the club
type GameTable struct {
	ID          int64     `json:"id" db:"id"`
	Name        string    `json:"name" db:"name" binding:"required"`
	Description *string   `json:"description,omitempty" db:"description"`
	Status      string    `json:"status" db:"status"` // e.g., available, occupied, reserved, maintenance
	Capacity    *int      `json:"capacity,omitempty" db:"capacity"`
	HourlyRate  *float64  `json:"hourly_rate,omitempty" db:"hourly_rate"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Booking represents a reservation for a game table
type Booking struct {
	ID             int64      `json:"id" db:"id"`
	ClientID       *int64     `json:"client_id,omitempty" db:"client_id"`
	TableID        int64      `json:"table_id" db:"table_id" binding:"required"`
	StaffID        *int64     `json:"staff_id,omitempty" db:"staff_id"`
	StartTime      time.Time  `json:"start_time" db:"start_time" binding:"required"`
	EndTime        time.Time  `json:"end_time" db:"end_time" binding:"required"`
	NumberOfGuests *int       `json:"number_of_guests,omitempty" db:"number_of_guests"`
	Status         string     `json:"status" db:"status"` // e.g., confirmed, cancelled, completed, no-show
	Notes          *string    `json:"notes,omitempty" db:"notes"`
	TotalPrice     *float64   `json:"total_price,omitempty" db:"total_price"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
	Client         *Client    `json:"client,omitempty"`    // For joining with Client details
	GameTable      *GameTable `json:"game_table,omitempty"` // For joining with GameTable details
	StaffMember    *StaffMember `json:"staff_member,omitempty"` // For joining with StaffMember details
}

// BookingFilters defines the available filters for querying bookings.
type BookingFilters struct {
	ClientID  *int64     `form:"client_id"`
	TableID   *int64     `form:"table_id"`
	StaffID   *int64     `form:"staff_id"`
	DateFrom  *time.Time `form:"date_from"` // Expect YYYY-MM-DD, time part will be ignored or set to start/end of day
	DateTo    *time.Time `form:"date_to"`   // Expect YYYY-MM-DD
	Status    *string    `form:"status"`
	Page      int        `form:"page"`
	PageSize  int        `form:"page_size"`
}

