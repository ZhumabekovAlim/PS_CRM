package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"ps_club_backend/internal/database"
	"ps_club_backend/internal/models"
	"ps_club_backend/pkg/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a given password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash compares a plain password with a hashed password
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// RegisterUser handles new user registration
func RegisterUser(c *gin.Context) {
	var payload models.RegistrationPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeBadRequest, "Invalid request payload", err.Error()))
		return
	}

	// Validate input
	if utils.IsEmpty(payload.Username) {
		utils.RespondValidationFailed(c, "Username cannot be empty.")
		return
	}
	if !utils.IsValidPasswordLength(payload.Password, 8) { // Example: Minimum 8 characters
		utils.RespondValidationFailed(c, "Password must be at least 8 characters long.")
		return
	}
	if payload.Email != nil && !utils.IsEmpty(*payload.Email) && !utils.IsValidEmail(*payload.Email) {
		utils.RespondValidationFailed(c, "Invalid email format.")
		return
	}

	hashedPassword, err := HashPassword(payload.Password)
	if err != nil {
		utils.LogError(err, "Failed to hash password during registration")
		utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to process registration", "Could not hash password."))
		return
	}

	db := database.GetDB()
	var existingUserID int64
	queryCheck := "SELECT id FROM users WHERE username = $1"
	argsCheck := []interface{}{payload.Username}
	if payload.Email != nil && !utils.IsEmpty(*payload.Email) {
		queryCheck += " OR email = $2"
		argsCheck = append(argsCheck, *payload.Email)
	}

	err = db.QueryRow(queryCheck, argsCheck...).Scan(&existingUserID)
	if err != nil && err != sql.ErrNoRows {
		utils.LogError(err, "Database error checking existing user")
		utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Database error", err.Error()))
		return
	}
	if err == nil { // User found
		utils.RespondWithError(c, utils.NewAPIError(http.StatusConflict, utils.ErrCodeConflict, "User already exists", "Username or email is already taken."))
		return
	}

	var roleID sql.NullInt64
	defaultRoleName := "Staff"
	if payload.RoleName != nil && !utils.IsEmpty(*payload.RoleName) {
		defaultRoleName = *payload.RoleName
	}

	var rID int64
	err = db.QueryRow("SELECT id FROM roles WHERE name = $1", defaultRoleName).Scan(&rID)
	if err != nil && err != sql.ErrNoRows {
		utils.LogError(err, "Failed to query role by name: "+defaultRoleName)
		utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to assign role", err.Error()))
		return
	}
	if err == sql.ErrNoRows {
		utils.LogInfo("Default role not found, creating: "+defaultRoleName)
		insertRoleQuery := "INSERT INTO roles (name, description) VALUES ($1, $2) RETURNING id"
		errInsert := db.QueryRow(insertRoleQuery, defaultRoleName, defaultRoleName+" role").Scan(&rID)
		if errInsert != nil {
			utils.LogError(errInsert, "Failed to create default role: "+defaultRoleName)
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to create role", errInsert.Error()))
			return
		}
	}
	roleID.Int64 = rID
	roleID.Valid = true

	newUser := models.User{
		Username:     payload.Username,
		PasswordHash: hashedPassword,
		Email:        payload.Email,
		FullName:     payload.FullName,
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if roleID.Valid {
		newUser.RoleID = &roleID.Int64
	}

	query := `INSERT INTO users (username, password_hash, email, full_name, role_id, is_active, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`
	err = db.QueryRow(query,
		newUser.Username, newUser.PasswordHash, newUser.Email, newUser.FullName, newUser.RoleID, newUser.IsActive, newUser.CreatedAt, newUser.UpdatedAt,
	).Scan(&newUser.ID)

	if err != nil {
		utils.LogError(err, "Failed to create user in database")
		utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to create user", err.Error()))
		return
	}

	utils.LogInfo("User registered successfully", map[string]interface{}{"userID": newUser.ID, "username": newUser.Username})
	newUser.PasswordHash = "" // Omit password hash in response
	c.JSON(http.StatusCreated, newUser)
}

// LoginUser handles user login and returns JWT tokens
func LoginUser(c *gin.Context) {
	var creds models.Credentials
	if err := c.ShouldBindJSON(&creds); err != nil {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeBadRequest, "Invalid request payload", err.Error()))
		return
	}

	if utils.IsEmpty(creds.Username) || utils.IsEmpty(creds.Password) {
		utils.RespondValidationFailed(c, "Username and password are required.")
		return
	}

	db := database.GetDB()
	var user models.User
	var roleName string

	query := `
		SELECT u.id, u.username, u.password_hash, u.email, u.full_name, u.role_id, u.is_active, 
		       COALESCE(r.name, (SELECT name FROM roles ORDER BY id LIMIT 1)) as role_name
		FROM users u
		LEFT JOIN roles r ON u.role_id = r.id
		WHERE u.username = $1`

	err := db.QueryRow(query, creds.Username).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.Email, &user.FullName, &user.RoleID, &user.IsActive,
		&roleName,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusUnauthorized, utils.ErrCodeUnauthorized, "Invalid credentials", "Username or password incorrect."))
		} else {
			utils.LogError(err, "Database error during login for user: "+creds.Username)
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Login failed", "Database error."))
		}
		return
	}

	if !user.IsActive {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusForbidden, utils.ErrCodeForbidden, "Account inactive", "User account is not active."))
		return
	}

	if !CheckPasswordHash(creds.Password, user.PasswordHash) {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusUnauthorized, utils.ErrCodeUnauthorized, "Invalid credentials", "Username or password incorrect."))
		return
	}

	accessToken, err := utils.GenerateAccessToken(user.ID, user.Username, roleName)
	if err != nil {
		utils.LogError(err, "Failed to generate access token for user: "+user.Username)
		utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Login failed", "Could not generate access token."))
		return
	}

	refreshToken, err := utils.GenerateRefreshToken(user.ID)
	if err != nil {
		utils.LogError(err, "Failed to generate refresh token for user: "+user.Username)
		utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Login failed", "Could not generate refresh token."))
		return
	}

	utils.LogInfo("User logged in successfully", map[string]interface{}{"userID": user.ID, "username": user.Username})
	user.PasswordHash = "" 
	c.JSON(http.StatusOK, gin.H{
		"message":       "Login successful",
		"user":          user,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"role":          roleName,
	})
}

// RefreshToken handles generating new access tokens using a refresh token
func RefreshToken(c *gin.Context) {
	var requestBody struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeBadRequest, "Invalid request", "refresh_token is required."))
		return
	}

	claims, err := utils.ValidateToken(requestBody.RefreshToken)
	if err != nil {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusUnauthorized, utils.ErrCodeUnauthorized, "Invalid token", "Refresh token is invalid or expired: "+err.Error()))
		return
	}

	db := database.GetDB()
	var user models.User
	var roleName string
	query := `
		SELECT u.id, u.username, u.is_active, COALESCE(r.name, (SELECT name FROM roles ORDER BY id LIMIT 1)) as role_name
		FROM users u
		LEFT JOIN roles r ON u.role_id = r.id
		WHERE u.id = $1`
	err = db.QueryRow(query, claims.UserID).Scan(&user.ID, &user.Username, &user.IsActive, &roleName)
	if err != nil {
		utils.LogError(err, "Failed to retrieve user details for token refresh, userID: "+string(claims.UserID))
		utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Token refresh failed", "Could not retrieve user details."))
		return
	}

	if !user.IsActive {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusForbidden, utils.ErrCodeForbidden, "Account inactive", "User account is not active."))
		return
	}

	newAccessToken, err := utils.GenerateAccessToken(claims.UserID, user.Username, roleName)
	if err != nil {
		utils.LogError(err, "Failed to generate new access token for userID: "+string(claims.UserID))
		utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Token refresh failed", "Could not generate new access token."))
		return
	}

	utils.LogInfo("Access token refreshed successfully", map[string]interface{}{"userID": claims.UserID})
	c.JSON(http.StatusOK, gin.H{
		"access_token": newAccessToken,
	})
}

// GetCurrentUser retrieves details for the currently authenticated user
func GetCurrentUser(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusUnauthorized, utils.ErrCodeUnauthorized, "Authentication required", "User ID not found in token claims."))
		return
	}
	userID, ok := userIDVal.(int64)
	if !ok {
		utils.LogError(nil, "User ID in token is not of expected type int64")
		utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Error processing user identity", "Invalid user ID type in token."))
		return
	}

	db := database.GetDB()
	var user models.User
	var roleName, roleDescription sql.NullString

	query := `
        SELECT u.id, u.username, u.email, u.full_name, u.role_id, u.is_active, u.created_at, u.updated_at,
               r.name as role_name, r.description as role_description
        FROM users u
        LEFT JOIN roles r ON u.role_id = r.id
        WHERE u.id = $1`

	err := db.QueryRow(query, userID).Scan(
		&user.ID, &user.Username, &user.Email, &user.FullName, &user.RoleID, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		&roleName, &roleDescription,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusNotFound, utils.ErrCodeNotFound, "User not found", "Authenticated user data not found."))
		} else {
			utils.LogError(err, "Database error fetching current user, userID: "+string(userID))
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to retrieve user data", err.Error()))
		}
		return
	}

	if user.RoleID != nil {
		user.Role = &models.Role{ID: *user.RoleID}
		if roleName.Valid { user.Role.Name = roleName.String }
		if roleDescription.Valid { user.Role.Description = &roleDescription.String }
	}

	user.PasswordHash = "" 
	utils.LogInfo("Current user data retrieved", map[string]interface{}{"userID": user.ID})
	c.JSON(http.StatusOK, user)
}

// LogoutUser (Conceptual - JWT is stateless)
func LogoutUser(c *gin.Context) {
	userIDVal, _ := c.Get("userID")
	utils.LogInfo("User logged out (client-side token removal expected)", map[string]interface{}{"userID": userIDVal})
	c.JSON(http.StatusOK, gin.H{"message": "Logout successful. Please clear your tokens on the client-side."})
}

