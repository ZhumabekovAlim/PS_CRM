package repositories

import (
	"database/sql"
	"errors"
	"fmt"
	"ps_club_backend/internal/models"
	"time"

	"github.com/lib/pq" // For pq.Error
)

// AuthRepository defines the interface for authentication-related database operations.
type AuthRepository interface {
	CreateUser(executor SQLExecutor, user *models.User, hashedPassword string) (int64, error)
	FindUserByUsername(username string) (*models.User, string, error) // Returns User, HashedPassword, Error
	FindUserByID(userID int64) (*models.User, error)
	// TODO: Add methods for refresh token management
}

// authRepository implements the AuthRepository interface.
type authRepository struct {
	db *sql.DB // The direct database connection pool
}

// NewAuthRepository creates a new instance of AuthRepository.
func NewAuthRepository(db *sql.DB) AuthRepository {
	return &authRepository{db: db}
}

// CreateUser inserts a new user into the database.
// It expects an SQLExecutor which can be a *sql.DB or *sql.Tx.
// The user model should have Username. Other fields like Email, FullName, RoleID are optional.
// IsActive is set to true by default. CreatedAt and UpdatedAt are set to the current time.
func (r *authRepository) CreateUser(executor SQLExecutor, user *models.User, hashedPassword string) (int64, error) {
	query := `INSERT INTO users (username, password_hash, email, full_name, role_id, is_active, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	          RETURNING id`
	
	currentTime := time.Now()
	isActive := true // Default to true for new users

	// Ensure RoleID is handled correctly if it's optional and might be nil on user model
	var roleID sql.NullInt64
	if user.RoleID != nil {
		roleID = sql.NullInt64{Int64: *user.RoleID, Valid: true}
	}

	var userID int64
	err := executor.QueryRow(
		query,
		user.Username,
		hashedPassword,
		user.Email,    // Can be nil
		user.FullName, // Can be nil
		roleID,        // Use sql.NullInt64 for nullable foreign keys
		isActive,
		currentTime,
		currentTime,
	).Scan(&userID)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code.Name() == "unique_violation" {
				// The specific column causing the violation can be checked via pqErr.Constraint.
				// This detail might be useful for the service layer to return a more specific error.
				return 0, fmt.Errorf("%w: %s (constraint: %s)", ErrDuplicateKey, pqErr.Message, pqErr.Constraint)
			}
		}
		return 0, fmt.Errorf("%w: creating user: %v", ErrDatabaseError, err)
	}
	return userID, nil
}

// FindUserByUsername retrieves a user by their username.
// It returns the user model, their hashed password, and an error if any.
func (r *authRepository) FindUserByUsername(username string) (*models.User, string, error) {
	user := &models.User{}
	var hashedPassword string
	// Query to fetch user details along with role name
	// Assumes 'roles' table exists and is joinable via users.role_id = roles.id
	query := `
		SELECT u.id, u.username, u.password_hash, u.email, u.full_name, u.role_id, u.is_active, u.created_at, u.updated_at,
		       COALESCE(ro.name, '') as role_name 
		FROM users u
		LEFT JOIN roles ro ON u.role_id = ro.id
		WHERE u.username = $1`

	var roleName sql.NullString 
	var roleID sql.NullInt64   // To correctly scan nullable role_id

	err := r.db.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &hashedPassword, &user.Email, &user.FullName,
		&roleID, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		&roleName,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", ErrNotFound // Use the common repository error
		}
		// Wrap other SQL errors
		return nil, "", fmt.Errorf("%w: finding user by username %s: %v", ErrDatabaseError, username, err)
	}

	if roleID.Valid {
		user.RoleID = &roleID.Int64
		if roleName.Valid {
			user.Role = &models.Role{ID: *user.RoleID, Name: roleName.String}
		}
	}


	return user, hashedPassword, nil
}

// FindUserByID retrieves a user by their ID.
// It returns the user model and an error if any.
func (r *authRepository) FindUserByID(userID int64) (*models.User, error) {
	user := &models.User{}
	// Query to fetch user details along with role name
	query := `
		SELECT u.id, u.username, u.password_hash, u.email, u.full_name, u.role_id, u.is_active, u.created_at, u.updated_at,
		       COALESCE(ro.name, '') as role_name
		FROM users u
		LEFT JOIN roles ro ON u.role_id = ro.id
		WHERE u.id = $1`
	
	var roleName sql.NullString
	var roleID sql.NullInt64 // To correctly scan nullable role_id
	var passwordHash string // Dummy variable to scan password_hash as it's in the select query

	err := r.db.QueryRow(query, userID).Scan(
		&user.ID, &user.Username, &passwordHash, &user.Email, &user.FullName,
		&roleID, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		&roleName,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound // Use the common repository error
		}
		// Wrap other SQL errors
		return nil, fmt.Errorf("%w: finding user by ID %d: %v", ErrDatabaseError, userID, err)
	}
	
	if roleID.Valid {
		user.RoleID = &roleID.Int64
		if roleName.Valid {
			user.Role = &models.Role{ID: *user.RoleID, Name: roleName.String}
		}
	}
	// user.PasswordHash is intentionally not populated here from passwordHash variable,
	// as this method is generally for retrieving user profile data, not for auth checks.
	// The PasswordHash is still selected to ensure the Scan works correctly with the query.

	return user, nil
}
