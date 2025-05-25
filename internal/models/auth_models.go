package models

import "time"

// User represents a user in the system
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username" db:"username"`
	PasswordHash string    `json:"-" db:"password_hash"` // '-' means don't send in JSON response
	Email        *string   `json:"email,omitempty" db:"email"`
	FullName     *string   `json:"full_name,omitempty" db:"full_name"`
	RoleID       *int64    `json:"role_id,omitempty" db:"role_id"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
	Role         *Role     `json:"role,omitempty"` // For joining with Role
}

// Role represents a user role
type Role struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name" db:"name"`
	Description *string   `json:"description,omitempty" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	Permissions []Permission `json:"permissions,omitempty"` // For joining with Permissions
}

// Permission represents an action a role can perform
type Permission struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name" db:"name"`
	Description *string   `json:"description,omitempty" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// RolePermission is the join table for roles and permissions
type RolePermission struct {
	RoleID       int64     `json:"role_id" db:"role_id"`
	PermissionID int64     `json:"permission_id" db:"permission_id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// Credentials for login request
type Credentials struct {
    Username string `json:"username" binding:"required"`
    Password string `json:"password" binding:"required"`
}

// RegistrationPayload for user registration
type RegistrationPayload struct {
    Username string  `json:"username" binding:"required"`
    Password string  `json:"password" binding:"required"`
    Email    *string `json:"email,omitempty"`
    FullName *string `json:"full_name,omitempty"`
    RoleName *string `json:"role_name,omitempty"` // e.g., "Admin", "Staff"
}


