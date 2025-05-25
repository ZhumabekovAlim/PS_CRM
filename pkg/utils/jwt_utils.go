package utils

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// jwtSecretKey is used to sign and verify JWT tokens. 
// IMPORTANT: In a production environment, this key should be strong and come from a secure configuration (e.g., environment variable).
var jwtSecretKey = []byte("your-super-secret-and-long-jwt-key-ps-club-crm") // TODO: Move to config/env

const (
	AccessTokenTTL  = 15 * time.Minute    // Access token lives for 15 minutes
	RefreshTokenTTL = 7 * 24 * time.Hour // Refresh token lives for 7 days
)

// Claims defines the JWT claims structure
type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"` // User role for authorization
	jwt.RegisteredClaims
}

// GenerateAccessToken creates a new JWT access token for a given user ID, username, and role.
func GenerateAccessToken(userID int64, username string, role string) (string, error) {
	expirationTime := time.Now().Add(AccessTokenTTL)
	claims := &Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "ps-club-crm-backend",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}
	return tokenString, nil
}

// GenerateRefreshToken creates a new JWT refresh token for a given user ID.
// Refresh tokens typically have fewer claims and a longer expiry.
func GenerateRefreshToken(userID int64) (string, error) {
	expirationTime := time.Now().Add(RefreshTokenTTL)
	claims := &Claims{
		UserID: userID, // Only UserID needed for refresh token to identify user
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "ps-club-crm-backend-refresh",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecretKey) // Use the same secret for simplicity, or a different one for refresh tokens
	if err != nil {
		return "", fmt.Errorf("failed to sign refresh token: %w", err)
	}
	return tokenString, nil
}

// ValidateToken parses and validates a JWT token string.
// It returns the claims if the token is valid, otherwise an error.
func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

