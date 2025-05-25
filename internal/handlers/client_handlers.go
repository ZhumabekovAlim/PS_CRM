package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"ps_club_backend/internal/database"
	"ps_club_backend/internal/models"

	"github.com/gin-gonic/gin"
)

// CreateClient handles the creation of a new client
func CreateClient(c *gin.Context) {
	var client models.Client
	if err := c.ShouldBindJSON(&client); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	db := database.GetDB()
	query := `INSERT INTO clients (full_name, phone_number, email, date_of_birth, loyalty_points, notes, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id, created_at, updated_at`

	client.CreatedAt = time.Now()
	client.UpdatedAt = time.Now()

	// Handle date_of_birth string to time.Time conversion if necessary, or store as string if DB schema supports it directly.
	// For simplicity, assuming date_of_birth is a string that fits the DB.

	err := db.QueryRow(query, 
		client.FullName, client.PhoneNumber, client.Email, client.DateOfBirth, 
		client.LoyaltyPoints, client.Notes, client.CreatedAt, client.UpdatedAt,
	).Scan(&client.ID, &client.CreatedAt, &client.UpdatedAt)

	if err != nil {
		// Check for unique constraint violation on phone_number or email if applicable
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create client: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, client)
}

// GetClients handles fetching all clients, with optional search/filtering
func GetClients(c *gin.Context) {
	db := database.GetDB()
	
	// Basic query, can be expanded with search parameters
	// e.g., c.Query("search_term") to filter by name or phone
	searchTerm := c.Query("search")

	queryStr := "SELECT id, full_name, phone_number, email, date_of_birth, loyalty_points, notes, created_at, updated_at FROM clients"
	var args []interface{}
	if searchTerm != "" {
		queryStr += " WHERE full_name ILIKE $1 OR phone_number ILIKE $1 OR email ILIKE $1"
		args = append(args, "%"+searchTerm+"%")
	}
	queryStr += " ORDER BY full_name"

	rows, err := db.Query(queryStr, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch clients: " + err.Error()})
		return
	}
	defer rows.Close()

	clients := []models.Client{}
	for rows.Next() {
		var cli models.Client
		if err := rows.Scan(
			&cli.ID, &cli.FullName, &cli.PhoneNumber, &cli.Email, 
			&cli.DateOfBirth, &cli.LoyaltyPoints, &cli.Notes, &cli.CreatedAt, &cli.UpdatedAt,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan client: " + err.Error()})
			return
		}
		clients = append(clients, cli)
	}
	c.JSON(http.StatusOK, clients)
}

// GetClientByID handles fetching a single client by ID
func GetClientByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
		return
	}

	db := database.GetDB()
	var cli models.Client
	query := "SELECT id, full_name, phone_number, email, date_of_birth, loyalty_points, notes, created_at, updated_at FROM clients WHERE id = $1"
	err = db.QueryRow(query, id).Scan(
		&cli.ID, &cli.FullName, &cli.PhoneNumber, &cli.Email, 
		&cli.DateOfBirth, &cli.LoyaltyPoints, &cli.Notes, &cli.CreatedAt, &cli.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch client: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, cli)
}

// UpdateClient handles updating an existing client
func UpdateClient(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
		return
	}

	var client models.Client
	if err := c.ShouldBindJSON(&client); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	db := database.GetDB()
	query := `UPDATE clients SET 
	          full_name = $1, phone_number = $2, email = $3, date_of_birth = $4, 
	          loyalty_points = $5, notes = $6, updated_at = $7
	          WHERE id = $8 
	          RETURNING id, full_name, phone_number, email, date_of_birth, loyalty_points, notes, created_at, updated_at`

	client.UpdatedAt = time.Now()

	err = db.QueryRow(query, 
		client.FullName, client.PhoneNumber, client.Email, client.DateOfBirth, 
		client.LoyaltyPoints, client.Notes, client.UpdatedAt, id,
	).Scan(
		&client.ID, &client.FullName, &client.PhoneNumber, &client.Email, 
		&client.DateOfBirth, &client.LoyaltyPoints, &client.Notes, &client.CreatedAt, &client.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found to update"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update client: " + err.Error()})
		return
	}
	client.ID = id // Ensure ID from path is used
	c.JSON(http.StatusOK, client)
}

// DeleteClient handles deleting a client
func DeleteClient(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
		return
	}

	db := database.GetDB()
	// Consider implications: what happens to bookings/orders associated with this client?
	// DB schema uses ON DELETE SET NULL for client_id in bookings and orders, so those records won't be deleted.
	result, err := db.Exec("DELETE FROM clients WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete client: " + err.Error()})
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client not found to delete"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Client deleted successfully"})
}

