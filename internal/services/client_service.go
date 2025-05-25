package services

import (
	"database/sql"
	"errors"
	"fmt"
	"ps_club_backend/internal/models"
	"ps_club_backend/internal/repositories"
	"regexp"
	"strings"
	"time"
)

// --- Custom Service Errors for Client ---
var (
	ErrClientNotFound     = errors.New("client not found")
	ErrPhoneNumberExists  = errors.New("phone number already exists")
	ErrEmailExists        = errors.New("email already exists")
	ErrClientValidation   = errors.New("client data validation error")
	ErrDateFormat         = errors.New("invalid date format, please use YYYY-MM-DD")
	ErrClientInUse        = errors.New("client cannot be deleted as they are referenced in other records")
)

// --- Client DTOs ---
type CreateClientRequest struct {
	FullName      string  `json:"full_name" binding:"required"`
	PhoneNumber   *string `json:"phone_number"` 
	Email         *string `json:"email"`        
	DateOfBirth   *string `json:"date_of_birth"` // Format YYYY-MM-DD
	LoyaltyPoints *int    `json:"loyalty_points"`
	Notes         *string `json:"notes"`
}

type UpdateClientRequest struct {
	FullName      *string `json:"full_name"`
	PhoneNumber   *string `json:"phone_number"`
	Email         *string `json:"email"`
	DateOfBirth   *string `json:"date_of_birth"` // Format YYYY-MM-DD
	LoyaltyPoints *int    `json:"loyalty_points"`
	Notes         *string `json:"notes"`
}

// --- ClientService Interface ---
type ClientService interface {
	CreateClient(req CreateClientRequest) (*models.Client, error)
	GetClientByID(clientID int64) (*models.Client, error)
	GetClients(page, pageSize int, searchTerm *string) ([]models.Client, int, error)
	UpdateClient(clientID int64, req UpdateClientRequest) (*models.Client, error)
	DeleteClient(clientID int64) error
}

// --- clientService Implementation ---
type clientService struct {
	clientRepo repositories.ClientRepository
	db         *sql.DB 
}

// NewClientService creates a new instance of ClientService.
func NewClientService(repo repositories.ClientRepository, db *sql.DB) ClientService {
	return &clientService{
		clientRepo: repo,
		db:         db,
	}
}

var emailRegex = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)

func (s *clientService) validateClientData(fullName string, phoneNumber, email *string, isUpdate bool, clientID int64) error {
	if strings.TrimSpace(fullName) == "" && !isUpdate { // FullName is required for create
		return fmt.Errorf("%w: full name cannot be empty", ErrClientValidation)
	}
    if fullName != "" && strings.TrimSpace(fullName) == "" { // if provided in update, cannot be empty
        return fmt.Errorf("%w: full name cannot be empty if provided", ErrClientValidation)
    }


	if phoneNumber != nil {
		pn := strings.TrimSpace(*phoneNumber)
		if pn == "" && !isUpdate { // Required for create if provided as non-nil but empty
			return fmt.Errorf("%w: phone number cannot be empty if provided", ErrClientValidation)
		}
        if pn != "" {
            // Basic validation, can be more complex (e.g. E.164)
            // if !regexp.MustCompile(`^\+?[0-9\s\-()]{7,20}$`).MatchString(pn) {
            //     return fmt.Errorf("%w: phone number format is invalid", ErrClientValidation)
            // }
            // Check for uniqueness if phone number is being set or changed
            existingClient, err := s.clientRepo.GetClientByPhoneNumber(pn)
            if err != nil && !errors.Is(err, repositories.ErrNotFound) {
                return fmt.Errorf("failed to check phone number uniqueness: %w", err)
            }
            if existingClient != nil && existingClient.ID != clientID { // If client exists and it's not the current client being updated
                return ErrPhoneNumberExists
            }
        }
	}

	if email != nil && *email != "" {
		em := strings.ToLower(strings.TrimSpace(*email))
		if !emailRegex.MatchString(em) {
			return fmt.Errorf("%w: email format is invalid", ErrClientValidation)
		}
		// TODO: Add uniqueness check for email if required by business logic, similar to phone number
	}
	return nil
}

func (s *clientService) parseDateOfBirth(dobStr *string) (*time.Time, error) {
	if dobStr == nil || strings.TrimSpace(*dobStr) == "" {
		return nil, nil 
	}
	dob, err := time.Parse("2006-01-02", *dobStr)
	if err != nil {
		return nil, ErrDateFormat
	}
	if dob.After(time.Now()) {
		return nil, fmt.Errorf("%w: date of birth cannot be in the future", ErrClientValidation)
	}
	return &dob, nil
}

func (s *clientService) CreateClient(req CreateClientRequest) (*models.Client, error) {
	if err := s.validateClientData(req.FullName, req.PhoneNumber, req.Email, false, 0); err != nil {
		return nil, err
	}

	dob, err := s.parseDateOfBirth(req.DateOfBirth)
	if err != nil {
		return nil, err
	}

	loyaltyPoints := 0
	if req.LoyaltyPoints != nil {
		loyaltyPoints = *req.LoyaltyPoints
		if loyaltyPoints < 0 {
			return nil, fmt.Errorf("%w: loyalty points cannot be negative", ErrClientValidation)
		}
	}
	
	client := &models.Client{
		FullName:      req.FullName,
		PhoneNumber:   req.PhoneNumber,
		Email:         req.Email,
		DateOfBirth:   dob,
		LoyaltyPoints: &loyaltyPoints,
		Notes:         req.Notes,
	}

	id, err := s.clientRepo.CreateClient(s.db, client)
	if err != nil {
		if errors.Is(err, repositories.ErrDuplicateKey) {
			if req.PhoneNumber != nil && strings.Contains(err.Error(), "clients_phone_number_key") {
				return nil, ErrPhoneNumberExists
			}
			if req.Email != nil && strings.Contains(err.Error(), "clients_email_key") {
				return nil, ErrEmailExists
			}
			return nil, fmt.Errorf("failed to create client due to duplicate data: %w", err)
		}
		return nil, fmt.Errorf("failed to create client in repository: %w", err)
	}
	return s.clientRepo.GetClientByID(id)
}

func (s *clientService) GetClientByID(clientID int64) (*models.Client, error) {
	client, err := s.clientRepo.GetClientByID(clientID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrClientNotFound
		}
		return nil, fmt.Errorf("failed to get client by ID: %w", err)
	}
	return client, nil
}

func (s *clientService) GetClients(page, pageSize int, searchTerm *string) ([]models.Client, int, error) {
	if page <= 0 { page = 1 }
	if pageSize <= 0 { pageSize = 10 }

	clients, totalCount, err := s.clientRepo.GetClients(page, pageSize, searchTerm)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get clients: %w", err)
	}
	return clients, totalCount, nil
}

func (s *clientService) UpdateClient(clientID int64, req UpdateClientRequest) (*models.Client, error) {
	client, err := s.clientRepo.GetClientByID(clientID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrClientNotFound
		}
		return nil, fmt.Errorf("failed to find client for update: %w", err)
	}

	// Prepare fields for validation
	fullNameToValidate := client.FullName
	if req.FullName != nil {
		fullNameToValidate = *req.FullName
	}
	// Use new phone/email for validation if provided, otherwise existing
    phoneNumberToValidate := client.PhoneNumber
    if req.PhoneNumber != nil {
        phoneNumberToValidate = req.PhoneNumber
    }
    emailToValidate := client.Email
    if req.Email != nil {
        emailToValidate = req.Email
    }

	if err := s.validateClientData(fullNameToValidate, phoneNumberToValidate, emailToValidate, true, clientID); err != nil {
		return nil, err
	}
	
	if req.FullName != nil { client.FullName = *req.FullName }
	if req.PhoneNumber != nil { client.PhoneNumber = req.PhoneNumber }
	if req.Email != nil { client.Email = req.Email }
	if req.DateOfBirth != nil {
		dob, parseErr := s.parseDateOfBirth(req.DateOfBirth)
		if parseErr != nil { return nil, parseErr }
		client.DateOfBirth = dob
	}
	if req.LoyaltyPoints != nil {
		if *req.LoyaltyPoints < 0 {
			return nil, fmt.Errorf("%w: loyalty points cannot be negative", ErrClientValidation)
		}
		client.LoyaltyPoints = req.LoyaltyPoints
	}
	if req.Notes != nil { client.Notes = req.Notes }

	err = s.clientRepo.UpdateClient(s.db, client)
	if err != nil {
		if errors.Is(err, repositories.ErrDuplicateKey) {
			if req.PhoneNumber != nil && strings.Contains(err.Error(), "clients_phone_number_key") {
				return nil, ErrPhoneNumberExists
			}
			if req.Email != nil && strings.Contains(err.Error(), "clients_email_key") {
				return nil, ErrEmailExists
			}
			return nil, fmt.Errorf("failed to update client due to duplicate data: %w", err)
		}
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrClientNotFound // Should have been caught by GetClientByID
		}
		return nil, fmt.Errorf("failed to update client in repository: %w", err)
	}
	return s.clientRepo.GetClientByID(clientID)
}

func (s *clientService) DeleteClient(clientID int64) error {
	_, err := s.clientRepo.GetClientByID(clientID) 
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ErrClientNotFound
		}
		return fmt.Errorf("failed to find client for deletion: %w", err)
	}

	err = s.clientRepo.DeleteClient(s.db, clientID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ErrClientNotFound
		}
		if strings.Contains(err.Error(), "violates foreign key constraint") {
             return ErrClientInUse
        }
		return fmt.Errorf("failed to delete client: %w", err)
	}
	return nil
}
