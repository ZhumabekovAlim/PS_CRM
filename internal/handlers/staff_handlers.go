package handlers

import (
	"errors"
	"net/http"
	"strconv"
	// "time" // Not directly used by handlers, service handles time parsing

	"ps_club_backend/internal/services"
	"ps_club_backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

// StaffHandler holds the staff service.
type StaffHandler struct {
	staffService services.StaffService
}

// NewStaffHandler creates a new StaffHandler.
func NewStaffHandler(ss services.StaffService) *StaffHandler {
	return &StaffHandler{staffService: ss}
}

// --- StaffMember Handler Methods ---

// CreateStaffMember handles the creation of a new staff member.
func (h *StaffHandler) CreateStaffMember(c *gin.Context) {
	var req services.CreateStaffMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.LogError(err, "CreateStaffMember: Failed to bind JSON")
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid request payload: "+err.Error(), err.Error()))
		return
	}

	staffMember, err := h.staffService.CreateStaffMember(req)
	if err != nil {
		utils.LogError(err, "CreateStaffMember: Error from staffService.CreateStaffMember")
		if errors.Is(err, services.ErrUserForStaffNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeBadRequest, "User specified for staff member not found.", err.Error()))
		} else if errors.Is(err, services.ErrStaffUserConflict) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusConflict, utils.ErrCodeConflict, "User ID is already linked to another staff member.", err.Error()))
		} else if errors.Is(err, services.ErrHireDateFormat) || errors.Is(err, services.ErrStaffDataValidation) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Validation failed: "+err.Error(), err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to create staff member.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusCreated, staffMember)
}

// GetStaffMembers handles fetching all staff members with pagination and search.
func (h *StaffHandler) GetStaffMembers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	searchTerm := c.Query("search")

	if page <= 0 { page = 1 }
	if pageSize <= 0 { pageSize = 10 }
	
	var pSearchTerm *string
	if searchTerm != "" {
		pSearchTerm = &searchTerm
	}

	staffMembers, totalCount, err := h.staffService.GetStaffMembers(page, pageSize, pSearchTerm)
	if err != nil {
		utils.LogError(err, "GetStaffMembers: Error from staffService.GetStaffMembers")
		utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to fetch staff members.", "Internal error"))
		return
	}
	
	if staffMembers == nil {
	    staffMembers = []models.StaffMember{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  staffMembers,
		"total": totalCount,
		"page":  page,
		"page_size": pageSize,
	})
}

// GetStaffMemberByID handles fetching a single staff member by ID.
func (h *StaffHandler) GetStaffMemberByID(c *gin.Context) {
	idStr := c.Param("id")
	staffID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid staff member ID format.", err.Error()))
		return
	}

	staffMember, err := h.staffService.GetStaffMemberByID(staffID)
	if err != nil {
		utils.LogError(err, "GetStaffMemberByID: Error from staffService.GetStaffMemberByID for ID "+idStr)
		if errors.Is(err, services.ErrStaffNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusNotFound, utils.ErrCodeNotFound, "Staff member not found.", err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to fetch staff member.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusOK, staffMember)
}

// UpdateStaffMember handles updating a staff member.
func (h *StaffHandler) UpdateStaffMember(c *gin.Context) {
	idStr := c.Param("id")
	staffID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid staff member ID format.", err.Error()))
		return
	}

	var req services.UpdateStaffMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.LogError(err, "UpdateStaffMember: Failed to bind JSON for ID "+idStr)
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid request payload: "+err.Error(), err.Error()))
		return
	}

	staffMember, err := h.staffService.UpdateStaffMember(staffID, req)
	if err != nil {
		utils.LogError(err, "UpdateStaffMember: Error from staffService.UpdateStaffMember for ID "+idStr)
		if errors.Is(err, services.ErrStaffNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusNotFound, utils.ErrCodeNotFound, "Staff member not found to update.", err.Error()))
		} else if errors.Is(err, services.ErrHireDateFormat) || errors.Is(err, services.ErrStaffDataValidation) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Validation failed: "+err.Error(), err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to update staff member.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusOK, staffMember)
}

// DeleteStaffMember handles deleting a staff member.
func (h *StaffHandler) DeleteStaffMember(c *gin.Context) {
	idStr := c.Param("id")
	staffID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid staff member ID format.", err.Error()))
		return
	}

	err = h.staffService.DeleteStaffMember(staffID)
	if err != nil {
		utils.LogError(err, "DeleteStaffMember: Error from staffService.DeleteStaffMember for ID "+idStr)
		if errors.Is(err, services.ErrStaffNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusNotFound, utils.ErrCodeNotFound, "Staff member not found to delete.", err.Error()))
		} else if errors.Is(err, services.ErrStaffInUse) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusConflict, utils.ErrCodeConflict, "Staff member cannot be deleted as they are referenced in other records.", err.Error()))
		}else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to delete staff member.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Staff member deleted successfully"})
}

// --- Shift Handler Methods ---

// CreateShift handles the creation of a new shift.
func (h *StaffHandler) CreateShift(c *gin.Context) {
	var req services.CreateShiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.LogError(err, "CreateShift: Failed to bind JSON")
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid request payload: "+err.Error(), err.Error()))
		return
	}

	shift, err := h.staffService.CreateShift(req)
	if err != nil {
		utils.LogError(err, "CreateShift: Error from staffService.CreateShift")
		if errors.Is(err, services.ErrStaffNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeBadRequest, "Staff member for shift not found.", err.Error()))
		} else if errors.Is(err, services.ErrShiftTimeFormat) || errors.Is(err, services.ErrShiftValidation) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Validation failed: "+err.Error(), err.Error()))
		} else if errors.Is(err, services.ErrShiftOverlap) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusConflict, utils.ErrCodeConflict, "Shift overlaps with an existing shift.", err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to create shift.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusCreated, shift)
}

// GetShifts handles fetching all shifts with pagination and filters.
func (h *StaffHandler) GetShifts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	
	if page <= 0 { page = 1 }
	if pageSize <= 0 { pageSize = 10 }

	var staffID *int64
	if staffIDStr := c.Query("staff_id"); staffIDStr != "" {
		id, err := strconv.ParseInt(staffIDStr, 10, 64)
		if err == nil {
			staffID = &id
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid staff_id format.", err.Error()))
			return
		}
	}
	
	startTimeFromStr := c.Query("start_time_from")
	startTimeToStr := c.Query("start_time_to")
	var pStartTimeFrom, pStartTimeTo *string
	if startTimeFromStr != "" { pStartTimeFrom = &startTimeFromStr }
	if startTimeToStr != "" { pStartTimeTo = &startTimeToStr }


	shifts, totalCount, err := h.staffService.GetShifts(staffID, pStartTimeFrom, pStartTimeTo, page, pageSize)
	if err != nil {
		utils.LogError(err, "GetShifts: Error from staffService.GetShifts")
		if errors.Is(err, services.ErrShiftTimeFormat) || errors.Is(err, services.ErrShiftValidation) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Validation failed for time parameters: "+err.Error(), err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to fetch shifts.", "Internal error"))
		}
		return
	}
	
	if shifts == nil {
	    shifts = []models.Shift{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  shifts,
		"total": totalCount,
		"page":  page,
		"page_size": pageSize,
	})
}

// GetShiftByID handles fetching a single shift by ID.
func (h *StaffHandler) GetShiftByID(c *gin.Context) {
	idStr := c.Param("id")
	shiftID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid shift ID format.", err.Error()))
		return
	}

	shift, err := h.staffService.GetShiftByID(shiftID)
	if err != nil {
		utils.LogError(err, "GetShiftByID: Error from staffService.GetShiftByID for ID "+idStr)
		if errors.Is(err, services.ErrShiftNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusNotFound, utils.ErrCodeNotFound, "Shift not found.", err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to fetch shift.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusOK, shift)
}

// UpdateShift handles updating a shift.
func (h *StaffHandler) UpdateShift(c *gin.Context) {
	idStr := c.Param("id")
	shiftID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid shift ID format.", err.Error()))
		return
	}

	var req services.UpdateShiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.LogError(err, "UpdateShift: Failed to bind JSON for ID "+idStr)
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid request payload: "+err.Error(), err.Error()))
		return
	}

	shift, err := h.staffService.UpdateShift(shiftID, req)
	if err != nil {
		utils.LogError(err, "UpdateShift: Error from staffService.UpdateShift for ID "+idStr)
		if errors.Is(err, services.ErrShiftNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusNotFound, utils.ErrCodeNotFound, "Shift not found to update.", err.Error()))
		} else if errors.Is(err, services.ErrShiftTimeFormat) || errors.Is(err, services.ErrShiftValidation) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Validation failed: "+err.Error(), err.Error()))
		} else if errors.Is(err, services.ErrShiftOverlap) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusConflict, utils.ErrCodeConflict, "Updated shift overlaps with an existing shift.", err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to update shift.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusOK, shift)
}

// DeleteShift handles deleting a shift.
func (h *StaffHandler) DeleteShift(c *gin.Context) {
	idStr := c.Param("id")
	shiftID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.RespondWithError(c, utils.NewAPIError(http.StatusBadRequest, utils.ErrCodeValidationFailed, "Invalid shift ID format.", err.Error()))
		return
	}

	err = h.staffService.DeleteShift(shiftID)
	if err != nil {
		utils.LogError(err, "DeleteShift: Error from staffService.DeleteShift for ID "+idStr)
		if errors.Is(err, services.ErrShiftNotFound) {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusNotFound, utils.ErrCodeNotFound, "Shift not found to delete.", err.Error()))
		} else {
			utils.RespondWithError(c, utils.NewAPIError(http.StatusInternalServerError, utils.ErrCodeInternalServerError, "Failed to delete shift.", "Internal error"))
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Shift deleted successfully"})
}

// Ensure old standalone functions are removed or commented out.
// e.g., func CreateStaffMember(c *gin.Context) { ... }
// func GetStaffMembers(c *gin.Context) { ... }
// ...etc...
// func CreateShift(c *gin.Context) { ... }
// ...etc...
