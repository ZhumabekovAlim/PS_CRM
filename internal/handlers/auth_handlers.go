package handlers

import (
	"errors"
	"net/http"
	"ps_club_backend/internal/services"
	"ps_club_backend/pkg/utils" // For APIError and error codes

	"github.com/gin-gonic/gin"
)

// AuthHandler holds the authentication service.
type AuthHandler struct {
	authService services.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(as services.AuthService) *AuthHandler {
	return &AuthHandler{authService: as}
}

// RegisterUser handles user registration.
func (h *AuthHandler) RegisterUser(c *gin.Context) {
	var req services.RegisterUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.LogError(err, "RegisterUser: Failed to bind JSON")
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid request payload: "+err.Error(), err.Error()))
		return
	}

	user, err := h.authService.RegisterUser(req)
	if err != nil {
		utils.LogError(err, "RegisterUser: Error from authService.RegisterUser")
		if errors.Is(err, services.ErrUsernameExists) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusConflict, utils.ErrCodeConflict, "Username already exists.", err.Error()))
		} else if errors.Is(err, services.ErrEmailExists) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusConflict, utils.ErrCodeConflict, "Email already exists.", err.Error()))
		} else if errors.Is(err, services.ErrRoleNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeBadRequest, "Specified role not found.", err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to register user.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusCreated, user)
}

// LoginUser handles user login.
func (h *AuthHandler) LoginUser(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.LogError(err, "LoginUser: Failed to bind JSON")
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid request payload: "+err.Error(), err.Error()))
		return
	}

	authResp, err := h.authService.LoginUser(req)
	if err != nil {
		utils.LogError(err, "LoginUser: Error from authService.LoginUser")
		if errors.Is(err, services.ErrInvalidCredentials) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusUnauthorized, utils.ErrCodeUnauthorized, "Invalid username or password.", err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to login.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusOK, authResp)
}

// GetCurrentUser retrieves the profile of the currently authenticated user.
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userIDRaw, exists := c.Get("userID")
	if !exists {
		utils.LogError(errors.New("userID not found in context"), "GetCurrentUser: userID not in context")
		utils.RespondWithError(c, utils.NewAPIError(http.StatusUnauthorized, utils.ErrCodeUnauthorized, "User not authenticated.", "Missing user ID in context"))
		return
	}

	userID, ok := userIDRaw.(int64)
	if !ok {
		utils.LogError(errors.New("userID is not of type int64"), "GetCurrentUser: userID type assertion failed")
		utils.RespondWithError(c, utils.NewAPIError(http.StatusUnauthorized, utils.ErrCodeUnauthorized, "User ID format incorrect.", "Invalid user ID format in context"))
		return
	}

	user, err := h.authService.GetUserProfile(userID)
	if err != nil {
		utils.LogError(err, "GetCurrentUser: Error from authService.GetUserProfile for userID "+utils.Int64ToStr(userID))
		if errors.Is(err, services.ErrUserNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusNotFound, utils.ErrCodeNotFound, "User profile not found.", err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to retrieve user profile.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusOK, user)
}

// LogoutUser handles user logout.
// For stateless JWT, this is primarily a client-side action.
func (h *AuthHandler) LogoutUser(c *gin.Context) {
	// Optional: could extract userID from token to log the logout action.
	// For stateless JWT, server doesn't do much other than acknowledge.
	// If using a refresh token blocklist, that logic would be invoked here via the authService.
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully. Please discard your token."})
}

// RefreshToken handles refreshing an access token.
// This is a placeholder as RefreshAccessToken is not yet in AuthService.
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	// TODO: Implement when AuthService supports RefreshAccessToken.
	// Example structure:
	// var req struct { RefreshToken string `json:"refresh_token" binding:"required"` }
	// if err := c.ShouldBindJSON(&req); err != nil { ... }
	// authResp, err := h.authService.RefreshAccessToken(req.RefreshToken)
	// if err != nil { ... handle errors like invalid/expired refresh token ... }
	// c.JSON(http.StatusOK, authResp)

	utils.RespondWithError(c, utils.NewAPIError(http.StatusNotImplemented, utils.ErrCodeNotImplemented, "Refresh token functionality is not yet implemented.", "Not implemented"))
}

// Standalone handler functions that are not yet part of AuthHandler (if any)
// For example, if RegisterUser, LoginUser etc. were not methods of AuthHandler initially.
// This section should be empty after refactoring.
// handlers.RegisterUser, handlers.LoginUser, etc., are now methods of AuthHandler.
// The direct calls like `handlers.RegisterUser` in `route_groups.go` will be updated
// to `authHandler.RegisterUser`.

// Ensure all old standalone auth functions are removed or commented out if they existed in this file.
// func RegisterUser(c *gin.Context) { /* ... old code ... */ }
// func LoginUser(c *gin.Context) { /* ... old code ... */ }
// func GetCurrentUser(c *gin.Context) { /* ... old code ... */ }
// func LogoutUser(c *gin.Context) { /* ... old code ... */ }
// func RefreshToken(c *gin.Context) { /* ... old code ... */ }
