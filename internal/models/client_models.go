package models

import "time"

// Client represents a customer of the PS club
type Client struct {
	ID            int64     `json:"id" db:"id"`
	FullName      string    `json:"full_name" db:"full_name" binding:"required"`
	PhoneNumber   *string   `json:"phone_number,omitempty" db:"phone_number"`
	Email         *string   `json:"email,omitempty" db:"email"`
	DateOfBirth   *string   `json:"date_of_birth,omitempty" db:"date_of_birth"` // Store as string, parse to time.Time when needed
	LoyaltyPoints int       `json:"loyalty_points" db:"loyalty_points"`
	Notes         *string   `json:"notes,omitempty" db:"notes"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

