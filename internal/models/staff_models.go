package models

import "time"

// StaffMember represents an employee
type StaffMember struct {
	ID           int64     `json:"id" db:"id"`
	UserID       *int64    `json:"user_id,omitempty" db:"user_id"` // Link to users table for login
	PhoneNumber  *string   `json:"phone_number,omitempty" db:"phone_number"`
	Address      *string   `json:"address,omitempty" db:"address"`
	HireDate     *string   `json:"hire_date,omitempty" db:"hire_date"` // Store as string, parse to time.Time when needed
	Position     *string   `json:"position,omitempty" db:"position"`
	Salary       *float64  `json:"salary,omitempty" db:"salary"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
	User         *User     `json:"user,omitempty"` // For joining with User details (like full_name, email from users table)
}

// Shift represents a work shift for a staff member
type Shift struct {
	ID        int64     `json:"id" db:"id"`
	StaffID   int64     `json:"staff_id" db:"staff_id" binding:"required"`
	StartTime time.Time `json:"start_time" db:"start_time" binding:"required"`
	EndTime   time.Time `json:"end_time" db:"end_time" binding:"required"`
	Notes     *string   `json:"notes,omitempty" db:"notes"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	StaffMember *StaffMember `json:"staff_member,omitempty"` // For joining with StaffMember details
}

