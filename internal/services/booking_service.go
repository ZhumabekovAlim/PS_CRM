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

// --- Custom Service Errors for Booking ---
var (
	ErrBookingNotFound          = errors.New("booking not found")
	ErrTableNotAvailable        = errors.New("table is not available for the requested time")
	ErrInvalidBookingTime       = errors.New("invalid booking time (e.g., end before start, duration limits, or in the past)")
	ErrClientForBookingNotFound = errors.New("client specified for booking not found")
	ErrStaffForBookingNotFound  = errors.New("staff member specified for booking not found")
	ErrTableForBookingNotFound  = errors.New("table specified for booking not found") 
	ErrBookingStatusUpdate      = errors.New("invalid status transition or error updating booking status")
	ErrBookingValidation        = errors.New("booking data validation error")
)


// --- Booking DTOs ---
type CreateBookingRequest struct {
	ClientID       *int64  `json:"client_id"`
	TableID        int64   `json:"table_id" binding:"required"`
	StaffID        int64   `json:"staff_id" binding:"required"` 
	StartTime      string  `json:"start_time" binding:"required"` 
	EndTime        string  `json:"end_time" binding:"required"`
	NumberOfGuests *int    `json:"number_of_guests"`
	Notes          *string `json:"notes"`
	Status         *string `json:"status"` 
}

type UpdateBookingRequest struct {
	TableID        *int64  `json:"table_id"`
	StartTime      *string `json:"start_time"`
	EndTime        *string `json:"end_time"`
	NumberOfGuests *int    `json:"number_of_guests"`
	Notes          *string `json:"notes"`
	Status         *string `json:"status"`
}

// --- BookingService Interface ---
type BookingService interface {
	CreateBooking(req CreateBookingRequest) (*models.Booking, error)
	GetBookingByID(bookingID int64) (*models.Booking, error)
	GetBookings(filters models.BookingFilters) ([]models.Booking, int, error)
	UpdateBooking(bookingID int64, req UpdateBookingRequest) (*models.Booking, error)
	CancelBooking(bookingID int64) (*models.Booking, error) 
	CompleteBooking(bookingID int64) (*models.Booking, error) 
	DeleteBooking(bookingID int64) error
}

// --- bookingService Implementation ---
type bookingService struct {
	bookingRepo repositories.BookingRepository
	clientRepo  repositories.ClientRepository 
	staffRepo   repositories.StaffRepository  
	// tableRepo repositories.GameTableRepository // TODO: Add when GameTableRepository exists
	db *sql.DB
}

// NewBookingService creates a new instance of BookingService.
func NewBookingService(
	br repositories.BookingRepository,
	cr repositories.ClientRepository,
	sr repositories.StaffRepository,
	// tr repositories.GameTableRepository, // TODO
	db *sql.DB,
) BookingService {
	return &bookingService{
		bookingRepo: br,
		clientRepo:  cr,
		staffRepo:   sr,
		// tableRepo: tr, // TODO
		db: db,
	}
}

// parseAndValidateBookingTimes parses string dates to time.Time and performs validation.
func (s *bookingService) parseAndValidateBookingTimes(startTimeStr, endTimeStr string, forUpdate bool, existingStartTime *time.Time) (time.Time, time.Time, error) {
	startTime, err := parseDateTime(startTimeStr, ErrShiftTimeFormat) 
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("start_time: %w: %v", ErrInvalidBookingTime, err)
	}
	endTime, err := parseDateTime(endTimeStr, ErrShiftTimeFormat)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("end_time: %w: %v", ErrInvalidBookingTime, err)
	}

	if !endTime.After(startTime) {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: end time must be after start time", ErrInvalidBookingTime)
	}
	if endTime.Sub(startTime) < 15*time.Minute { // Example: Minimum booking duration
		return time.Time{}, time.Time{}, fmt.Errorf("%w: minimum booking duration is 15 minutes", ErrInvalidBookingTime)
	}
	if endTime.Sub(startTime) > 12*time.Hour { // Example: Maximum booking duration
		return time.Time{}, time.Time{}, fmt.Errorf("%w: maximum booking duration is 12 hours", ErrInvalidBookingTime)
	}
	
	// For new bookings, start time cannot be in the past.
	// For updates, if start time is being changed, it cannot be moved to the past.
	// Allow a small buffer (e.g., 5 mins) for clock sync issues or quick edits.
	if !forUpdate && startTime.Before(time.Now().Add(-5*time.Minute)) {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: booking start time cannot be in the past for new bookings", ErrInvalidBookingTime)
	}
	if forUpdate && existingStartTime != nil && startTime != *existingStartTime && startTime.Before(time.Now().Add(-5*time.Minute)) {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: booking start time cannot be moved to the past", ErrInvalidBookingTime)
	}


	return startTime, endTime, nil
}


func (s *bookingService) CreateBooking(req CreateBookingRequest) (*models.Booking, error) {
	startTime, endTime, err := s.parseAndValidateBookingTimes(req.StartTime, req.EndTime, false, nil)
	if err != nil {
		return nil, err
	}

	if req.ClientID != nil {
		_, err = s.clientRepo.GetClientByID(*req.ClientID)
		if err != nil {
			if errors.Is(err, repositories.ErrNotFound) {
				return nil, fmt.Errorf("%w: ID %d", ErrClientForBookingNotFound, *req.ClientID)
			}
			return nil, fmt.Errorf("failed to validate client for booking: %w", err)
		}
	}

	_, err = s.staffRepo.GetStaffMemberByID(req.StaffID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, fmt.Errorf("%w: ID %d", ErrStaffForBookingNotFound, req.StaffID)
		}
		return nil, fmt.Errorf("failed to validate staff for booking: %w", err)
	}
	
	// TODO: Validate TableID using a GameTableRepository if it exists.
	// For now, CheckTableAvailability implicitly requires table to exist for the query to not fail in a specific way.

	available, err := s.bookingRepo.CheckTableAvailability(req.TableID, startTime, endTime, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to check table availability: %w", err)
	}
	if !available {
		return nil, ErrTableNotAvailable
	}

	status := models.BookingStatusConfirmed 
	if req.Status != nil && strings.TrimSpace(*req.Status) != "" {
		if !models.IsValidBookingStatus(*req.Status) {
			return nil, fmt.Errorf("%w: invalid status '%s'", ErrBookingValidation, *req.Status)
		}
		status = *req.Status
	}
	
	booking := &models.Booking{
		ClientID:       req.ClientID,
		TableID:        req.TableID,
		StaffID:        &req.StaffID,
		StartTime:      startTime,
		EndTime:        endTime,
		NumberOfGuests: req.NumberOfGuests,
		Status:         status,
		Notes:          req.Notes,
		// TotalPrice will be calculated by repository or trigger if not set
	}

	createdBooking, err := s.bookingRepo.CreateBooking(s.db, booking)
	if err != nil {
		return nil, fmt.Errorf("failed to create booking in repository: %w", err)
	}
	
	return s.bookingRepo.GetBookingByID(createdBooking.ID) // Fetch with all joins
}

func (s *bookingService) GetBookingByID(bookingID int64) (*models.Booking, error) {
	booking, err := s.bookingRepo.GetBookingByID(bookingID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrBookingNotFound
		}
		return nil, fmt.Errorf("failed to get booking by ID: %w", err)
	}
	return booking, nil
}

func (s *bookingService) GetBookings(filters models.BookingFilters) ([]models.Booking, int, error) {
	if filters.Page <= 0 { filters.Page = 1 }
	if filters.PageSize <= 0 { filters.PageSize = 10 }
	
	bookings, totalCount, err := s.bookingRepo.GetBookings(filters)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get bookings: %w", err)
	}
	return bookings, totalCount, nil
}

func (s *bookingService) UpdateBooking(bookingID int64, req UpdateBookingRequest) (*models.Booking, error) {
	booking, err := s.bookingRepo.GetBookingByID(bookingID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrBookingNotFound
		}
		return nil, fmt.Errorf("failed to find booking for update: %w", err)
	}

	// Prevent updates to bookings that are already completed or cancelled
	if booking.Status == models.BookingStatusCompleted || booking.Status == models.BookingStatusCancelled {
		return nil, fmt.Errorf("%w: cannot update a booking that is already '%s'", ErrBookingValidation, booking.Status)
	}


	if req.TableID != nil { booking.TableID = *req.TableID }
	
	newStartTime, newEndTime := booking.StartTime, booking.EndTime
	timeChanged := false
	if req.StartTime != nil || req.EndTime != nil {
		sTimeStr := booking.StartTime.Format(time.RFC3339)
		eTimeStr := booking.EndTime.Format(time.RFC3339)
		if req.StartTime != nil { sTimeStr = *req.StartTime }
		if req.EndTime != nil { eTimeStr = *req.EndTime }
		
		parsedStartTime, parsedEndTime, timeErr := s.parseAndValidateBookingTimes(sTimeStr, eTimeStr, true, &booking.StartTime)
		if timeErr != nil { return nil, timeErr }
		newStartTime = parsedStartTime
		newEndTime = parsedEndTime
		timeChanged = true
	}


	if timeChanged || (req.TableID != nil && *req.TableID != booking.TableID) {
		available, availabilityErr := s.bookingRepo.CheckTableAvailability(booking.TableID, newStartTime, newEndTime, &bookingID)
		if availabilityErr != nil {
			return nil, fmt.Errorf("failed to check table availability for update: %w", availabilityErr)
		}
		if !available {
			return nil, ErrTableNotAvailable
		}
	}
	booking.StartTime = newStartTime
	booking.EndTime = newEndTime
	
	if req.NumberOfGuests != nil { booking.NumberOfGuests = req.NumberOfGuests }
	if req.Notes != nil { booking.Notes = req.Notes }
	if req.Status != nil { 
		if !models.IsValidBookingStatus(*req.Status) {
			return nil, fmt.Errorf("%w: invalid status '%s'", ErrBookingValidation, *req.Status)
		}
		// Additional logic for status transitions can be added here
		// e.g., if current status is "confirmed", can it be changed to "pending"?
		booking.Status = *req.Status
	}
	// TODO: Recalculate TotalPrice if times or table changed

	updatedBooking, err := s.bookingRepo.UpdateBooking(s.db, booking)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return nil, ErrBookingNotFound 
		}
		return nil, fmt.Errorf("failed to update booking in repository: %w", err)
	}
	return s.bookingRepo.GetBookingByID(updatedBooking.ID)
}

func (s *bookingService) updateBookingStatus(bookingID int64, newStatus string) (*models.Booking, error) {
    booking, err := s.bookingRepo.GetBookingByID(bookingID)
    if err != nil {
        if errors.Is(err, repositories.ErrNotFound) {
            return nil, ErrBookingNotFound
        }
        return nil, fmt.Errorf("failed to find booking to update status: %w", err)
    }

    // Basic status transition validation (can be more complex)
    if booking.Status == models.BookingStatusCompleted && newStatus != models.BookingStatusCompleted {
        return nil, fmt.Errorf("%w: cannot change status of a completed booking", ErrBookingStatusUpdate)
    }
    if booking.Status == models.BookingStatusCancelled && newStatus != models.BookingStatusCancelled {
         return nil, fmt.Errorf("%w: cannot change status of a cancelled booking", ErrBookingStatusUpdate)
    }

    booking.Status = newStatus
    // The UpdateBooking method updates more than just status.
    // A more specific repository method `UpdateBookingStatus` would be better.
    // For now, using the general UpdateBooking.
    updatedBooking, err := s.bookingRepo.UpdateBooking(s.db, booking) 
    if err != nil {
        return nil, fmt.Errorf("%w: %v", ErrBookingStatusUpdate, err)
    }
    return s.bookingRepo.GetBookingByID(updatedBooking.ID)
}

func (s *bookingService) CancelBooking(bookingID int64) (*models.Booking, error) {
	return s.updateBookingStatus(bookingID, models.BookingStatusCancelled)
}

func (s *bookingService) CompleteBooking(bookingID int64) (*models.Booking, error) {
	return s.updateBookingStatus(bookingID, models.BookingStatusCompleted)
}

func (s *bookingService) DeleteBooking(bookingID int64) error {
	_, err := s.bookingRepo.GetBookingByID(bookingID) 
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			return ErrBookingNotFound
		}
		return fmt.Errorf("failed to find booking for deletion: %w", err)
	}
	err = s.bookingRepo.DeleteBooking(s.db, bookingID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) { 
			return ErrBookingNotFound
		}
		return fmt.Errorf("failed to delete booking: %w", err)
	}
	return nil
}
