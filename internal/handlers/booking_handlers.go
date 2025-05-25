package handlers

import (
	"errors"
	"net/http"
	"ps_club_backend/internal/models"
	"ps_club_backend/internal/services"
	"ps_club_backend/pkg/utils"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// BookingHandler holds the booking service.
type BookingHandler struct {
	bookingService services.BookingService
}

// NewBookingHandler creates a new BookingHandler.
func NewBookingHandler(bs services.BookingService) *BookingHandler {
	return &BookingHandler{bookingService: bs}
}

// CreateBooking handles the creation of a new booking.
func (h *BookingHandler) CreateBooking(c *gin.Context) {
	var req services.CreateBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.LogError(err, "CreateBooking: Failed to bind JSON")
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid request payload: "+err.Error(), err.Error()))
		return
	}

	// Get StaffID from authenticated user (if not allowing override in request)
	// authStaffID, exists := c.Get("userID") // Assuming this is the UserID of staff
	// if !exists {
	// 	utils.RespondWithError(c, utils.NewAPIError(http.StatusUnauthorized, utils.ErrCodeUnauthorized, "User not authenticated for staff action.", ""))
	// 	return
	// }
	// req.StaffID = authStaffID.(int64) // This needs careful handling of type and if user is actually staff

	booking, err := h.bookingService.CreateBooking(req)
	if err != nil {
		utils.LogError(err, "CreateBooking: Error from bookingService.CreateBooking")
		if errors.Is(err, services.ErrTableNotAvailable) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusConflict, utils.ErrCodeConflict, err.Error(), err.Error()))
		} else if errors.Is(err, services.ErrInvalidBookingTime) || errors.Is(err, services.ErrBookingValidation) || errors.Is(err, services.ErrShiftTimeFormat) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, err.Error(), err.Error()))
		} else if errors.Is(err, services.ErrClientForBookingNotFound) || errors.Is(err, services.ErrStaffForBookingNotFound) || errors.Is(err, services.ErrTableForBookingNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeBadRequest, err.Error(), err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to create booking.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusCreated, booking)
}

// GetBookings handles fetching all bookings with pagination and filters.
func (h *BookingHandler) GetBookings(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page <= 0 { page = 1 }
	if pageSize <= 0 { pageSize = 10 }
	
	var filters models.BookingFilters
	filters.Page = page
	filters.PageSize = pageSize

	if clientIDStr := c.Query("client_id"); clientIDStr != "" {
		id, err := strconv.ParseInt(clientIDStr, 10, 64)
		if err == nil { filters.ClientID = &id 
		} else { utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid client_id format.", err.Error())); return }
	}
	if tableIDStr := c.Query("table_id"); tableIDStr != "" {
		id, err := strconv.ParseInt(tableIDStr, 10, 64)
		if err == nil { filters.TableID = &id 
		} else { utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid table_id format.", err.Error())); return }
	}
	if staffIDStr := c.Query("staff_id"); staffIDStr != "" {
		id, err := strconv.ParseInt(staffIDStr, 10, 64)
		if err == nil { filters.StaffID = &id 
		} else { utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid staff_id format.", err.Error())); return }
	}
	if statusStr := c.Query("status"); statusStr != "" {
		if !models.IsValidBookingStatus(statusStr) { // Assuming IsValidBookingStatus exists
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid status value.", "status: "+statusStr)); return
		}
		filters.Status = &statusStr
	}
	if dateFromStr := c.Query("date_from"); dateFromStr != "" {
		t, err := time.Parse("2006-01-02", dateFromStr)
		if err == nil { filters.DateFrom = &t 
		} else { utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid date_from format. Use YYYY-MM-DD.", err.Error())); return }
	}
	if dateToStr := c.Query("date_to"); dateToStr != "" {
		t, err := time.Parse("2006-01-02", dateToStr)
		if err == nil { 
			t = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second) // End of day
			filters.DateTo = &t 
		} else { utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid date_to format. Use YYYY-MM-DD.", err.Error())); return }
	}

	bookings, totalCount, err := h.bookingService.GetBookings(filters)
	if err != nil {
		utils.LogError(err, "GetBookings: Error from bookingService.GetBookings")
		utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to fetch bookings.", "Internal error"))
		return
	}
	
	if bookings == nil {
	    bookings = []models.Booking{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  bookings,
		"total": totalCount,
		"page":  page,
		"page_size": pageSize,
	})
}

// GetBookingByID handles fetching a single booking by ID.
func (h *BookingHandler) GetBookingByID(c *gin.Context) {
	idStr := c.Param("id")
	bookingID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid booking ID format.", err.Error()))
		return
	}

	booking, err := h.bookingService.GetBookingByID(bookingID)
	if err != nil {
		utils.LogError(err, "GetBookingByID: Error from bookingService.GetBookingByID for ID "+idStr)
		if errors.Is(err, services.ErrBookingNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusNotFound, utils.ErrCodeNotFound, "Booking not found.", err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to fetch booking.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusOK, booking)
}

// UpdateBooking handles updating a booking.
func (h *BookingHandler) UpdateBooking(c *gin.Context) {
	idStr := c.Param("id")
	bookingID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid booking ID format.", err.Error()))
		return
	}

	var req services.UpdateBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.LogError(err, "UpdateBooking: Failed to bind JSON for ID "+idStr)
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid request payload: "+err.Error(), err.Error()))
		return
	}

	booking, err := h.bookingService.UpdateBooking(bookingID, req)
	if err != nil {
		utils.LogError(err, "UpdateBooking: Error from bookingService.UpdateBooking for ID "+idStr)
		if errors.Is(err, services.ErrBookingNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusNotFound, utils.ErrCodeNotFound, "Booking not found to update.", err.Error()))
		} else if errors.Is(err, services.ErrTableNotAvailable) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusConflict, utils.ErrCodeConflict, err.Error(), err.Error()))
		} else if errors.Is(err, services.ErrInvalidBookingTime) || errors.Is(err, services.ErrBookingValidation) || errors.Is(err, services.ErrShiftTimeFormat) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, err.Error(), err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to update booking.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusOK, booking)
}

// CancelBooking handles cancelling a booking.
func (h *BookingHandler) CancelBooking(c *gin.Context) {
	idStr := c.Param("id")
	bookingID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid booking ID format.", err.Error()))
		return
	}

	booking, err := h.bookingService.CancelBooking(bookingID)
	if err != nil {
		utils.LogError(err, "CancelBooking: Error from bookingService.CancelBooking for ID "+idStr)
		if errors.Is(err, services.ErrBookingNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusNotFound, utils.ErrCodeNotFound, "Booking not found to cancel.", err.Error()))
		} else if errors.Is(err, services.ErrBookingStatusUpdate){
             utils.RespondWithError(c, utils.NewAPIError(http.StatusConflict, utils.ErrCodeConflict, err.Error(), err.Error()))
        }else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to cancel booking.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusOK, booking)
}

// CompleteBooking handles completing a booking.
func (h *BookingHandler) CompleteBooking(c *gin.Context) {
	idStr := c.Param("id")
	bookingID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid booking ID format.", err.Error()))
		return
	}

	booking, err := h.bookingService.CompleteBooking(bookingID)
	if err != nil {
		utils.LogError(err, "CompleteBooking: Error from bookingService.CompleteBooking for ID "+idStr)
		if errors.Is(err, services.ErrBookingNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusNotFound, utils.ErrCodeNotFound, "Booking not found to complete.", err.Error()))
		} else if errors.Is(err, services.ErrBookingStatusUpdate){
             utils.RespondWithError(c, utils.NewAPIError(http.StatusConflict, utils.ErrCodeConflict, err.Error(), err.Error()))
        } else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to complete booking.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusOK, booking)
}

// DeleteBooking handles deleting a booking.
func (h *BookingHandler) DeleteBooking(c *gin.Context) {
	idStr := c.Param("id")
	bookingID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid booking ID format.", err.Error()))
		return
	}

	err = h.bookingService.DeleteBooking(bookingID)
	if err != nil {
		utils.LogError(err, "DeleteBooking: Error from bookingService.DeleteBooking for ID "+idStr)
		if errors.Is(err, services.ErrBookingNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusNotFound, utils.ErrCodeNotFound, "Booking not found to delete.", err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to delete booking.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Booking deleted successfully"})
}

// If table_booking_handlers.go existed, its content would be removed or this file replaces it.
// For example, if there were old standalone functions:
// func CreateBookingHandler(c *gin.Context) { /* ... */ }
// func GetBookingsHandler(c *gin.Context) { /* ... */ }
// ... they are now replaced by methods on BookingHandler.The `BookingRepository` (Part 1), `BookingService` (Part 2), and `BookingHandler` (Part 3) have been successfully implemented in the previous turns. All specified files were created/updated, and their contents align with the detailed instructions.

**Summary of actions already taken for Parts 1, 2, & 3:**

**Part 0: Prerequisites**
    - `internal/models/table_booking_models.go` was updated with the `BookingFilters` struct.

**Part 1: Booking Repository Layer (`internal/repositories/booking_repository.go`)**
    - `BookingRepository` interface and `bookingRepository` struct (with `*sql.DB`) implemented.
    - `NewBookingRepository` constructor created.
    - Helper `scanBookingRow` (and `scanner` interface) implemented for DRY scanning of `Booking` with joined details. `GetBookings` was updated to correctly scan `total_count`.
    - **All interface methods implemented**: `CreateBooking`, `GetBookingByID` (with joins), `GetBookings` (with filters, pagination, joins, and correct total count), `UpdateBooking`, `DeleteBooking`, and `CheckTableAvailability`.
    - Standard error handling and `SQLExecutor` usage are in place.

**Part 2: Booking Service Layer (`internal/services/booking_service.go`)**
    - DTOs (`CreateBookingRequest`, `UpdateBookingRequest`) defined.
    - `BookingService` interface (including `CancelBooking`, `CompleteBooking`) defined.
    - `bookingService` struct (with `BookingRepository`, `ClientRepository`, `StaffRepository`, `*sql.DB`) and `NewBookingService` constructor implemented.
    - Service-specific errors (e.g., `ErrBookingNotFound`, `ErrTableNotAvailable`) defined.
    - Helper function `parseAndValidateBookingTimes` implemented.
    - **All interface methods implemented**:
        - `CreateBooking`: Validates input (times, foreign key existence via respective repos), checks table availability, sets default status, calls repository.
        - `GetBookingByID`: Calls repository.
        - `GetBookings`: Calls repository, handles default pagination.
        - `UpdateBooking`: Fetches booking, validates, checks availability if times/table change, calls repository.
        - `CancelBooking` & `CompleteBooking`: Implemented as wrappers around a general `updateBookingStatus` helper that calls `bookingRepo.UpdateBooking`.
        - `DeleteBooking`: Checks existence, then calls repository.

**Part 3: Booking Handler Refinement (`internal/handlers/booking_handlers.go`)**
    - `BookingHandler` struct (with `services.BookingService`) and `NewBookingHandler` constructor implemented.
    - Handler methods for Booking CRUD (`CreateBooking`, `GetBookings`, `GetBookingByID`, `UpdateBooking`, `DeleteBooking`) and status changes (`CancelBooking`, `CompleteBooking`) implemented as methods of `BookingHandler`.
    - Methods bind JSON to service DTOs, parse path/query parameters (including date parsing for `GetBookings` filters), call `bookingService` methods, and map service errors to `utils.APIError` for HTTP responses.

Now, I will proceed with **Part 4: Booking Dependency Injection Update**.

**My plan for Part 4:**

1.  **Update `internal/router/router.go`**:
    *   Initialize `BookingRepository`.
    *   Initialize `BookingService` with its repository dependencies (`BookingRepository`, `ClientRepository`, `StaffRepository`, and `db`).
    *   Initialize `BookingHandler` with `BookingService`.
    *   Pass the new `bookingHandler` to `SetupBookingRoutes`.

2.  **Update `internal/router/route_groups.go`**:
    *   Modify `SetupBookingRoutes` to accept `*handlers.BookingHandler`.
    *   Update route definitions within `SetupBookingRoutes` to call methods on the passed `bookingHandler`, including the new routes for `CancelBooking` and `CompleteBooking`.

Let's start by reading `internal/router/router.go`.
