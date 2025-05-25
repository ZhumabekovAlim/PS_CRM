package services

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"ps_club_backend/internal/models"
	"ps_club_backend/internal/repositories"
	// "ps_club_backend/pkg/utils" // Not using separate JWT utils for now

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// --- Custom Service Errors ---
var (
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUsernameExists     = errors.New("username already exists")
	ErrEmailExists        = errors.New("email already exists")
	ErrRoleNotFound       = errors.New("specified role not found")
	ErrTokenGeneration    = errors.New("failed to generate token")
)

// --- Data Transfer Objects (DTOs) ---

// LoginRequest DTO
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterUserRequest DTO
type RegisterUserRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	FullName string `json:"full_name" binding:"required"`
	RoleName string `json:"role_name"` // e.g., "Client", "Staff". Default if empty.
}

// AuthResponse DTO
type AuthResponse struct {
	User         *models.User `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token,omitempty"`
}

// --- AuthService Interface ---
type AuthService interface {
	RegisterUser(req RegisterUserRequest) (*models.User, error)
	LoginUser(req LoginRequest) (*AuthResponse, error)
	GetUserProfile(userID int64) (*models.User, error)
}

// --- authService Implementation ---
type authService struct {
	authRepo      repositories.AuthRepository
	db            *sql.DB // Used as SQLExecutor for single repo calls, or for managing transactions
	jwtSecret     string
	jwtExpiration time.Duration
}

// NewAuthService creates a new instance of AuthService.
func NewAuthService(authRepo repositories.AuthRepository, db *sql.DB, jwtSecret string, jwtExp time.Duration) AuthService {
	return &authService{
		authRepo:      authRepo,
		db:            db,
		jwtSecret:     jwtSecret,
		jwtExpiration: jwtExp,
	}
}

// generateJWT creates a new JWT token for a given user.
func (s *authService) generateJWT(user *models.User) (string, error) {
	roleName := "default" // Default role claim
	if user.Role != nil && user.Role.Name != "" {
		roleName = user.Role.Name
	} else if user.RoleID != nil {
		// This part is tricky without a RoleRepository.
		// For now, if Role.Name is not populated, we'll stick to "default"
		// or map known RoleIDs to names if necessary for the JWT claim.
		// A proper solution would involve ensuring user.Role is always populated by the repository
		// or fetching it here.
		// Let's assume for now the repo populates Role.Name if RoleID exists.
		// If user.Role is nil despite user.RoleID existing, that's a data consistency issue to be addressed.
	}

	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     roleName,
		"exp":      time.Now().Add(s.jwtExpiration).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrTokenGeneration, err)
	}
	return signedToken, nil
}

// RegisterUser handles the business logic for user registration.
func (s *authService) RegisterUser(req RegisterUserRequest) (*models.User, error) {
	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	hashedPassword := string(hashedPasswordBytes)

	var roleID *int64
	// Simplified RoleID determination logic
	// In a real app, this would involve a RoleRepository or a more robust lookup.
	if req.RoleName != "" {
		var tempRoleID int64
		// This mapping should ideally come from a configuration or database lookup
		roleMap := map[string]int64{
			"admin":  1, // Assuming 1 is Admin ID
			"staff":  2, // Assuming 2 is Staff ID
			"client": 3, // Assuming 3 is Client ID
		}
		normalizedRoleName := strings.ToLower(req.RoleName)
		if id, ok := roleMap[normalizedRoleName]; ok {
			tempRoleID = id
		} else {
			return nil, fmt.Errorf("%w: '%s'", ErrRoleNotFound, req.RoleName)
		}
		roleID = &tempRoleID
	} else {
		// Default to "client" role if no role_name is provided
		defaultClientRoleID := int64(3) // Assuming 3 is Client ID
		roleID = &defaultClientRoleID
	}

	user := models.User{
		Username: req.Username,
		Email:    &req.Email,
		FullName: &req.FullName,
		RoleID:   roleID,
	}

	createdUserID, err := s.authRepo.CreateUser(s.db, &user, hashedPassword)
	if err != nil {
		if errors.Is(err, repositories.ErrDuplicateKey) {
			// The repository error now includes constraint name, e.g.,
			// "duplicate key value violates unique constraint \"users_username_key\" (constraint: users_username_key)"
			if strings.Contains(err.Error(), "users_username_key") {
				return nil, ErrUsernameExists
			} else if strings.Contains(err.Error(), "users_email_key") { // Assuming 'users_email_key' is the constraint name for email
				return nil, ErrEmailExists
			}
			// Fallback if constraint name isn't as expected or not parsed
			return nil, fmt.Errorf("%w: %s", ErrUsernameExists, "username or email already taken")
		}
		return nil, fmt.Errorf("failed to register user: %w", err)
	}

	// Fetch the user again to get all details, including the ones set by DB (timestamps) and role name
	registeredUser, fetchErr := s.authRepo.FindUserByID(createdUserID)
	if fetchErr != nil {
		// This is unlikely but important to handle. The user was created but fetching failed.
		// Return a partial user object or a specific error.
		user.ID = createdUserID // At least return the ID
		user.PasswordHash = ""    // Ensure hash is not returned
		return &user, fmt.Errorf("user registered but failed to retrieve full details: %w", fetchErr)
	}
	registeredUser.PasswordHash = "" // Ensure hash is not returned
	return registeredUser, nil
}

// LoginUser handles user login and token generation.
func (s *authService) LoginUser(req LoginRequest) (*AuthResponse, error) {
	user, storedHashedPassword, err := s.authRepo.FindUserByUsername(req.Username)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("login attempt failed: %w", err)
	}

	if !user.IsActive {
		return nil, ErrInvalidCredentials // Or a more specific "user account is inactive" error
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedHashedPassword), []byte(req.Password))
	if err != nil {
		// err is bcrypt.ErrMismatchedHashAndPassword for wrong password
		return nil, ErrInvalidCredentials
	}

	accessToken, err := s.generateJWT(user)
	if err != nil {
		// Log the internal error for diagnosis
		// log.Printf("ERROR: Failed to generate JWT for user %s: %v", user.Username, err)
		return nil, fmt.Errorf("failed to generate access token: %w", err) // Return generic error to client
	}

	user.PasswordHash = "" // Clear password hash before returning user details
	return &AuthResponse{
		User:        user,
		AccessToken: accessToken,
		// RefreshToken: "...", // Implement refresh token generation if needed
	}, nil
}

// GetUserProfile retrieves a user's profile by their ID.
func (s *authService) GetUserProfile(userID int64) (*models.User, error) {
	user, err := s.authRepo.FindUserByID(userID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to retrieve user profile: %w", err)
	}
	user.PasswordHash = "" // Ensure password hash is not exposed
	return user, nil
}
