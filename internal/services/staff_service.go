package services

import (
	"database/sql"
	"errors"
	"fmt"
	"ps_club_backend/internal/models"
	"ps_club_backend/internal/repositories"
	"strings"
	"time"
)

// --- Custom Service Errors for Staff ---
var (
	ErrStaffNotFound       = errors.New("staff member not found")
	ErrUserForStaffNotFound= errors.New("user account for staff member not found")
	ErrStaffUserConflict   = errors.New("user ID is already associated with another staff member")
	ErrShiftNotFound       = errors.New("shift not found")
	ErrShiftValidation     = errors.New("shift validation error (e.g., end time before start time)")
	ErrShiftOverlap        = errors.New("shift overlaps with an existing shift for the staff member") 
	ErrStaffDataValidation = errors.New("staff data validation error")
	ErrHireDateFormat      = errors.New("invalid hire date format, please use YYYY-MM-DD")
	ErrShiftTimeFormat     = errors.New("invalid time format for shift, please use YYYY-MM-DDTHH:MM:SSZ or RFC3339 like format")
	ErrStaffInUse          = errors.New("staff member cannot be deleted as they are referenced in other records")
)

// --- StaffMember DTOs ---
type CreateStaffMemberRequest struct { 
	UserID      int64    `json:"user_id" binding:"required"`
	PhoneNumber *string  `json:"phone_number"`
	Address     *string  `json:"address"`
	HireDate    *string  `json:"hire_date"` 
	Position    *string  `json:"position" binding:"required"`
	Salary      *float64 `json:"salary"`
}

type UpdateStaffMemberRequest struct {
	PhoneNumber *string  `json:"phone_number"`
	Address     *string  `json:"address"`
	HireDate    *string  `json:"hire_date"`
	Position    *string  `json:"position"`
	Salary      *float64 `json:"salary"`
}

// --- Shift DTOs ---
type CreateShiftRequest struct {
	StaffID   int64   `json:"staff_id" binding:"required"`
	StartTime string  `json:"start_time" binding:"required"` 
	EndTime   string  `json:"end_time" binding:"required"`
	Notes     *string `json:"notes"`
}

type UpdateShiftRequest struct {
	StartTime *string `json:"start_time"`
	EndTime   *string `json:"end_time"`
	Notes     *string `json:"notes"`
}

// --- StaffService Interface ---
type StaffService interface {
	// StaffMember methods
	CreateStaffMember(req CreateStaffMemberRequest) (*models.StaffMember, error)
	GetStaffMemberByID(staffID int64) (*models.StaffMember, error)
	GetStaffMemberByUserID(userID int64) (*models.StaffMember, error)
	GetStaffMembers(page, pageSize int, searchTerm *string) ([]models.StaffMember, int, error)
	UpdateStaffMember(staffID int64, req UpdateStaffMemberRequest) (*models.StaffMember, error)
	DeleteStaffMember(staffID int64) error

	// Shift methods
	CreateShift(req CreateShiftRequest) (*models.Shift, error)
	GetShiftByID(shiftID int64) (*models.Shift, error)
	GetShifts(staffID *int64, startTimeFromStr *string, startTimeToStr *string, page, pageSize int) ([]models.Shift, int, error)
	UpdateShift(shiftID int64, req UpdateShiftRequest) (*models.Shift, error)
	DeleteShift(shiftID int64) error
}

// --- staffService Implementation ---
type staffService struct {
	staffRepo repositories.StaffRepository
	userRepo  repositories.AuthRepository 
	db        *sql.DB
}

// NewStaffService creates a new instance of StaffService.
func NewStaffService(sr repositories.StaffRepository, ur repositories.AuthRepository, db *sql.DB) StaffService {
	return &staffService{
		staffRepo: sr,
		userRepo:  ur,
		db:        db,
	}
}

func parseDate(dateStrPointer *string, format string, errorToReturn error) (*string, error) {
    if dateStrPointer == nil || strings.TrimSpace(*dateStrPointer) == "" {
        return nil, nil 
    }
    dateStr := strings.TrimSpace(*dateStrPointer)
    _, err := time.Parse(format, dateStr)
    if err != nil {
        return nil, errorToReturn
    }
    return &dateStr, nil 
}

func parseDateTime(dateTimeStr string, errorToReturn error) (time.Time, error) {
    parsedTime, err := time.Parse(time.RFC3339, dateTimeStr)
    if err != nil {
		// Try parsing without timezone if RFC3339 fails (common if client sends local time string)
		parsedTime, err = time.Parse("2006-01-02T15:04:05", dateTimeStr)
		if err != nil {
			return time.Time{}, errorToReturn
		}
    }
    return parsedTime, nil
}


// --- StaffMember Method Implementations ---

func (s *staffService) CreateStaffMember(req CreateStaffMemberRequest) (*models.StaffMember, error) {
	_, err := s.userRepo.FindUserByID(req.UserID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, fmt.Errorf("%w: user ID %d", ErrUserForStaffNotFound, req.UserID)
		}
		return nil, fmt.Errorf("failed to validate user for staff member: %w", err)
	}

	existingStaff, err := s.staffRepo.GetStaffMemberByUserID(req.UserID)
	if err != nil && !errors.Is(err, repositories.ErrNotFound) {
		return nil, fmt.Errorf("failed to check existing staff by user ID: %w", err)
	}
	if existingStaff != nil {
		return nil, fmt.Errorf("%w: user ID %d", ErrStaffUserConflict, req.UserID)
	}
	
	hireDateStrPtr, err := parseDate(req.HireDate, "2006-01-02", ErrHireDateFormat)
    if err != nil { return nil, err }

	if req.Position == nil || strings.TrimSpace(*req.Position) == "" {
		return nil, fmt.Errorf("%w: position cannot be empty", ErrStaffDataValidation)
	}
	if req.Salary != nil && *req.Salary < 0 {
		return nil, fmt.Errorf("%w: salary cannot be negative", ErrStaffDataValidation)
	}

	staff := &models.StaffMember{
		UserID:      &req.UserID,
		PhoneNumber: req.PhoneNumber,
		Address:     req.Address,
		HireDate:    hireDateStrPtr,
		Position:    req.Position,
		Salary:      req.Salary,
	}

	createdStaff, err := s.staffRepo.CreateStaffMember(s.db, staff)
	if err != nil {
		return nil, fmt.Errorf("failed to create staff member in repository: %w", err)
	}
	return s.staffRepo.GetStaffMemberByID(createdStaff.ID) // Fetch with User details
}

func (s *staffService) GetStaffMemberByID(staffID int64) (*models.StaffMember, error) {
	staff, err := s.staffRepo.GetStaffMemberByID(staffID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrStaffNotFound
		}
		return nil, fmt.Errorf("failed to get staff member by ID: %w", err)
	}
	return staff, nil
}

func (s *staffService) GetStaffMemberByUserID(userID int64) (*models.StaffMember, error) {
	staff, err := s.staffRepo.GetStaffMemberByUserID(userID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrStaffNotFound 
		}
		return nil, fmt.Errorf("failed to get staff member by user ID: %w", err)
	}
	return staff, nil
}

func (s *staffService) GetStaffMembers(page, pageSize int, searchTerm *string) ([]models.StaffMember, int, error) {
	if page <= 0 { page = 1 }
	if pageSize <= 0 { pageSize = 10 }
	
	staffMembers, totalCount, err := s.staffRepo.GetStaffMembers(page, pageSize, searchTerm)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get staff members: %w", err)
	}
	return staffMembers, totalCount, nil
}

func (s *staffService) UpdateStaffMember(staffID int64, req UpdateStaffMemberRequest) (*models.StaffMember, error) {
	staff, err := s.staffRepo.GetStaffMemberByID(staffID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrStaffNotFound
		}
		return nil, fmt.Errorf("failed to find staff member for update: %w", err)
	}

	if req.PhoneNumber != nil { staff.PhoneNumber = req.PhoneNumber }
	if req.Address != nil { staff.Address = req.Address }
	if req.HireDate != nil {
		hd, parseErr := parseDate(req.HireDate, "2006-01-02", ErrHireDateFormat)
		if parseErr != nil { return nil, parseErr }
		staff.HireDate = hd
	}
	if req.Position != nil { 
		if strings.TrimSpace(*req.Position) == "" {
			return nil, fmt.Errorf("%w: position cannot be empty if provided", ErrStaffDataValidation)
		}
		staff.Position = req.Position 
	}
	if req.Salary != nil { 
		if *req.Salary < 0 {
			return nil, fmt.Errorf("%w: salary cannot be negative", ErrStaffDataValidation)
		}
		staff.Salary = req.Salary 
	}
	
	updatedStaff, err := s.staffRepo.UpdateStaffMember(s.db, staff)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) { 
			return nil, ErrStaffNotFound
		}
		return nil, fmt.Errorf("failed to update staff member in repository: %w", err)
	}
	return s.staffRepo.GetStaffMemberByID(updatedStaff.ID)
}

func (s *staffService) DeleteStaffMember(staffID int64) error {
	_, err := s.staffRepo.GetStaffMemberByID(staffID) 
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ErrStaffNotFound
		}
		return fmt.Errorf("failed to find staff member for deletion: %w", err)
	}
	err = s.staffRepo.DeleteStaffMember(s.db, staffID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) { 
			return ErrStaffNotFound
		}
		if strings.Contains(err.Error(), "violates foreign key constraint") {
             return ErrStaffInUse
        }
		return fmt.Errorf("failed to delete staff member: %w", err)
	}
	return nil
}

// --- Shift Method Implementations ---

func (s *staffService) CreateShift(req CreateShiftRequest) (*models.Shift, error) {
	startTime, err := parseDateTime(req.StartTime, ErrShiftTimeFormat)
	if err != nil { return nil, fmt.Errorf("start_time: %w", err) }
	endTime, err := parseDateTime(req.EndTime, ErrShiftTimeFormat)
	if err != nil { return nil, fmt.Errorf("end_time: %w", err) }

	if !endTime.After(startTime) {
		return nil, fmt.Errorf("%w: end time must be after start time", ErrShiftValidation)
	}
	if endTime.Sub(startTime) > (24 * time.Hour) { // Example validation: shift not longer than 24 hours
        return nil, fmt.Errorf("%w: shift duration cannot exceed 24 hours", ErrShiftValidation)
    }


	_, err = s.staffRepo.GetStaffMemberByID(req.StaffID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, fmt.Errorf("%w: staff member with ID %d not found for shift", ErrStaffNotFound, req.StaffID)
		}
		return nil, fmt.Errorf("failed to validate staff member for shift: %w", err)
	}
	
	// TODO: Shift overlap validation
	// existingShifts, _, _ := s.staffRepo.GetShifts(&req.StaffID, &startTime, &endTime, 1, 1)
	// if len(existingShifts) > 0 { return nil, ErrShiftOverlap }

	shift := &models.Shift{
		StaffID:   req.StaffID,
		StartTime: startTime,
		EndTime:   endTime,
		Notes:     req.Notes,
	}

	createdShift, err := s.staffRepo.CreateShift(s.db, shift)
	if err != nil {
		return nil, fmt.Errorf("failed to create shift in repository: %w", err)
	}
	return s.staffRepo.GetShiftByID(createdShift.ID)
}

func (s *staffService) GetShiftByID(shiftID int64) (*models.Shift, error) {
	shift, err := s.staffRepo.GetShiftByID(shiftID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrShiftNotFound
		}
		return nil, fmt.Errorf("failed to get shift by ID: %w", err)
	}
	return shift, nil
}

func (s *staffService) GetShifts(staffID *int64, startTimeFromStr *string, startTimeToStr *string, page, pageSize int) ([]models.Shift, int, error) {
	if page <= 0 { page = 1 }
	if pageSize <= 0 { pageSize = 10 }

	var startTimeFrom, startTimeTo *time.Time
	var err error

	if startTimeFromStr != nil && strings.TrimSpace(*startTimeFromStr) != "" {
		t, parseErr := parseDateTime(*startTimeFromStr, ErrShiftTimeFormat)
		if parseErr != nil { return nil, 0, fmt.Errorf("start_time_from: %w", parseErr) }
		startTimeFrom = &t
	}
	if startTimeToStr != nil && strings.TrimSpace(*startTimeToStr) != "" {
		t, parseErr := parseDateTime(*startTimeToStr, ErrShiftTimeFormat)
		if parseErr != nil { return nil, 0, fmt.Errorf("start_time_to: %w", parseErr) }
		startTimeTo = &t
	}
	
	if startTimeFrom != nil && startTimeTo != nil && !startTimeTo.After(*startTimeFrom) {
        return nil, 0, fmt.Errorf("%w: end time filter must be after start time filter", ErrShiftValidation)
    }

	shifts, totalCount, err := s.staffRepo.GetShifts(staffID, startTimeFrom, startTimeTo, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get shifts: %w", err)
	}
	return shifts, totalCount, nil
}

func (s *staffService) UpdateShift(shiftID int64, req UpdateShiftRequest) (*models.Shift, error) {
	shift, err := s.staffRepo.GetShiftByID(shiftID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrShiftNotFound
		}
		return nil, fmt.Errorf("failed to find shift for update: %w", err)
	}

	newStartTime := shift.StartTime
    newEndTime := shift.EndTime

	if req.StartTime != nil {
		st, parseErr := parseDateTime(*req.StartTime, ErrShiftTimeFormat)
		if parseErr != nil { return nil, fmt.Errorf("start_time: %w", parseErr) }
		newStartTime = st
	}
	if req.EndTime != nil {
		et, parseErr := parseDateTime(*req.EndTime, ErrShiftTimeFormat)
		if parseErr != nil { return nil, fmt.Errorf("end_time: %w", parseErr) }
		newEndTime = et
	}

	if !newEndTime.After(newStartTime) {
		return nil, fmt.Errorf("%w: end time must be after start time", ErrShiftValidation)
	}
	if newEndTime.Sub(newStartTime) > (24 * time.Hour) {
        return nil, fmt.Errorf("%w: shift duration cannot exceed 24 hours", ErrShiftValidation)
    }
	
	shift.StartTime = newStartTime
    shift.EndTime = newEndTime

	if req.Notes != nil {
		shift.Notes = req.Notes
	}
	
	// TODO: Implement shift overlap validation for update if required
	
	updatedShift, err := s.staffRepo.UpdateShift(s.db, shift)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) { 
			return nil, ErrShiftNotFound
		}
		return nil, fmt.Errorf("failed to update shift in repository: %w", err)
	}
	return s.staffRepo.GetShiftByID(updatedShift.ID) 
}

func (s *staffService) DeleteShift(shiftID int64) error {
	_, err := s.staffRepo.GetShiftByID(shiftID) 
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ErrShiftNotFound
		}
		return fmt.Errorf("failed to find shift for deletion: %w", err)
	}
	err = s.staffRepo.DeleteShift(s.db, shiftID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) { 
			return ErrShiftNotFound
		}
		return fmt.Errorf("failed to delete shift: %w", err)
	}
	return nil
}
