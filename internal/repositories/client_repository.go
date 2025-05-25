package repositories

import (
	"database/sql"
	"errors"
	"fmt"
	"ps_club_backend/internal/models"
	"strings"
	"time"

	"github.com/lib/pq" // For pq.Error
)

// ClientRepository defines the interface for client-related database operations.
type ClientRepository interface {
	CreateClient(executor SQLExecutor, client *models.Client) (int64, error)
	GetClientByID(id int64) (*models.Client, error)
	GetClientByPhoneNumber(phoneNumber string) (*models.Client, error)
	GetClients(page, pageSize int, searchTerm *string) ([]models.Client, int, error) // Clients, total count, error
	UpdateClient(executor SQLExecutor, client *models.Client) error
	DeleteClient(executor SQLExecutor, id int64) error
}

type clientRepository struct {
	db *sql.DB
}

// NewClientRepository creates a new instance of ClientRepository.
func NewClientRepository(db *sql.DB) ClientRepository {
	return &clientRepository{db: db}
}

// CreateClient inserts a new client into the database.
func (r *clientRepository) CreateClient(executor SQLExecutor, client *models.Client) (int64, error) {
	query := `INSERT INTO clients (full_name, phone_number, email, date_of_birth, loyalty_points, notes, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	          RETURNING id`

	currentTime := time.Now()
	if client.CreatedAt.IsZero() {
		client.CreatedAt = currentTime
	}
	if client.UpdatedAt.IsZero() {
		client.UpdatedAt = currentTime
	}
	if client.LoyaltyPoints == nil { 
		defaultPoints := 0
		client.LoyaltyPoints = &defaultPoints
	}

	var dob sql.NullTime
	if client.DateOfBirth != nil && !client.DateOfBirth.IsZero() {
		dob = sql.NullTime{Time: *client.DateOfBirth, Valid: true}
	}


	err := executor.QueryRow(query,
		client.FullName, client.PhoneNumber, client.Email, dob,
		client.LoyaltyPoints, client.Notes, client.CreatedAt, client.UpdatedAt,
	).Scan(&client.ID)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code.Name() == "unique_violation" {
				return 0, fmt.Errorf("%w: %s (constraint: %s)", ErrDuplicateKey, pqErr.Message, pqErr.Constraint)
			}
		}
		return 0, fmt.Errorf("%w: creating client: %v", ErrDatabaseError, err)
	}
	return client.ID, nil
}

// GetClientByID retrieves a client by their ID.
func (r *clientRepository) GetClientByID(id int64) (*models.Client, error) {
	client := &models.Client{}
	query := `SELECT id, full_name, phone_number, email, date_of_birth, loyalty_points, notes, created_at, updated_at 
	          FROM clients WHERE id = $1`
	
	var dob sql.NullTime
	err := r.db.QueryRow(query, id).Scan(
		&client.ID, &client.FullName, &client.PhoneNumber, &client.Email, &dob,
		&client.LoyaltyPoints, &client.Notes, &client.CreatedAt, &client.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("%w: getting client by ID %d: %v", ErrDatabaseError, id, err)
	}
	if dob.Valid {
		client.DateOfBirth = &dob.Time
	}
	return client, nil
}

// GetClientByPhoneNumber retrieves a client by their phone number.
func (r *clientRepository) GetClientByPhoneNumber(phoneNumber string) (*models.Client, error) {
	client := &models.Client{}
	query := `SELECT id, full_name, phone_number, email, date_of_birth, loyalty_points, notes, created_at, updated_at 
	          FROM clients WHERE phone_number = $1`
	
	var dob sql.NullTime
	err := r.db.QueryRow(query, phoneNumber).Scan(
		&client.ID, &client.FullName, &client.PhoneNumber, &client.Email, &dob,
		&client.LoyaltyPoints, &client.Notes, &client.CreatedAt, &client.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("%w: getting client by phone number %s: %v", ErrDatabaseError, phoneNumber, err)
	}
	if dob.Valid {
		client.DateOfBirth = &dob.Time
	}
	return client, nil
}

// GetClients retrieves a list of clients with pagination and optional search.
func (r *clientRepository) GetClients(page, pageSize int, searchTerm *string) ([]models.Client, int, error) {
	clients := []models.Client{}
	totalCount := 0

	var queryBuilder strings.Builder
	queryBuilder.WriteString(`SELECT id, full_name, phone_number, email, date_of_birth, loyalty_points, notes, created_at, updated_at, COUNT(*) OVER() as total_count 
	                          FROM clients`)

	var conditions []string
	var args []interface{}
	argCount := 1

	if searchTerm != nil && *searchTerm != "" {
		searchPattern := "%" + strings.ToLower(*searchTerm) + "%"
		conditions = append(conditions, fmt.Sprintf("(LOWER(full_name) ILIKE $%d OR LOWER(phone_number) ILIKE $%d OR LOWER(email) ILIKE $%d)", argCount, argCount, argCount))
		args = append(args, searchPattern)
		argCount++
	}

	if len(conditions) > 0 {
		queryBuilder.WriteString(" WHERE " + strings.Join(conditions, " AND "))
	}

	queryBuilder.WriteString(" ORDER BY full_name ASC") 

	if pageSize > 0 {
		queryBuilder.WriteString(fmt.Sprintf(" LIMIT $%d", argCount))
		args = append(args, pageSize)
		argCount++
		if page > 0 {
			offset := (page - 1) * pageSize
			queryBuilder.WriteString(fmt.Sprintf(" OFFSET $%d", argCount))
			args = append(args, offset)
		}
	}
	
	rows, err := r.db.Query(queryBuilder.String(), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: querying clients: %v", ErrDatabaseError, err)
	}
	defer rows.Close()

	for rows.Next() {
		var client models.Client
		var dob sql.NullTime
		if err := rows.Scan(
			&client.ID, &client.FullName, &client.PhoneNumber, &client.Email, &dob,
			&client.LoyaltyPoints, &client.Notes, &client.CreatedAt, &client.UpdatedAt, &totalCount,
		); err != nil {
			return nil, 0, fmt.Errorf("%w: scanning client: %v", ErrDatabaseError, err)
		}
		if dob.Valid {
			client.DateOfBirth = &dob.Time
		}
		clients = append(clients, client)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("%w: iterating client rows: %v", ErrDatabaseError, err)
	}
	// If totalCount was not scanned from the first row (e.g., no results), it remains 0.
	// If there were results, totalCount is already set.
	if len(clients) == 0 { // If no clients matched, totalCount from OVER() would be 0 anyway.
		totalCount = 0
	}

	return clients, totalCount, nil
}

// UpdateClient updates an existing client in the database.
func (r *clientRepository) UpdateClient(executor SQLExecutor, client *models.Client) error {
	query := `UPDATE clients SET 
	            full_name = $1, phone_number = $2, email = $3, date_of_birth = $4, 
	            loyalty_points = $5, notes = $6, updated_at = $7 
	          WHERE id = $8`
	
	client.UpdatedAt = time.Now()
	var dob sql.NullTime
	if client.DateOfBirth != nil && !client.DateOfBirth.IsZero() {
		dob = sql.NullTime{Time: *client.DateOfBirth, Valid: true}
	}

	result, err := executor.Exec(query,
		client.FullName, client.PhoneNumber, client.Email, dob,
		client.LoyaltyPoints, client.Notes, client.UpdatedAt, client.ID,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code.Name() == "unique_violation" {
				return fmt.Errorf("%w: %s (constraint: %s)", ErrDuplicateKey, pqErr.Message, pqErr.Constraint)
			}
		}
		return fmt.Errorf("%w: updating client ID %d: %v", ErrDatabaseError, client.ID, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: getting rows affected for updating client ID %d: %v", ErrDatabaseError, client.ID, err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteClient removes a client from the database.
func (r *clientRepository) DeleteClient(executor SQLExecutor, id int64) error {
	query := `DELETE FROM clients WHERE id = $1`
	result, err := executor.Exec(query, id)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23503" { // foreign_key_violation
			return fmt.Errorf("%w: client ID %d cannot be deleted as it is referenced by other records (e.g., orders) (constraint: %s)", ErrDatabaseError, id, pqErr.Constraint)
		}
		return fmt.Errorf("%w: deleting client ID %d: %v", ErrDatabaseError, id, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: getting rows affected for deleting client ID %d: %v", ErrDatabaseError, id, err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
