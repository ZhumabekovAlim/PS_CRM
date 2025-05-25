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

// Staff Member Handlers

// CreateStaffMember handles the creation of a new staff member
// This might involve creating a user entry first or linking to an existing one.
func CreateStaffMember(c *gin.Context) {
	var staff models.StaffMember
	if err := c.ShouldBindJSON(&staff); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	// Business logic: If staff.UserID is not provided, and a user needs to be created (e.g., with a default role)
	// or if staff.User is provided with details for a new user, handle that here.
	// For simplicity, we assume staff.UserID might be null or point to an existing user.

	db := database.GetDB()
	query := `INSERT INTO staff_members (user_id, phone_number, address, hire_date, position, salary, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id, created_at, updated_at`

	staff.CreatedAt = time.Now()
	staff.UpdatedAt = time.Now()

	err := db.QueryRow(query,
		staff.UserID, staff.PhoneNumber, staff.Address, staff.HireDate,
		staff.Position, staff.Salary, staff.CreatedAt, staff.UpdatedAt,
	).Scan(&staff.ID, &staff.CreatedAt, &staff.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create staff member: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, staff)
}

// GetStaffMembers handles fetching all staff members, potentially with their user details
func GetStaffMembers(c *gin.Context) {
	db := database.GetDB()
	// Query to join staff_members with users table to get full_name, email etc.
	queryStr := `
		SELECT 
			sm.id, sm.user_id, sm.phone_number, sm.address, sm.hire_date, sm.position, sm.salary, 
			sm.created_at, sm.updated_at,
			u.full_name as user_full_name, u.email as user_email, u.username as user_username,
			r.name as user_role_name
		FROM staff_members sm
		LEFT JOIN users u ON sm.user_id = u.id
		LEFT JOIN roles r ON u.role_id = r.id
		ORDER BY u.full_name NULLS LAST, sm.id`

	rows, err := db.Query(queryStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch staff members: " + err.Error()})
		return
	}
	defer rows.Close()

	staffList := []models.StaffMember{}
	for rows.Next() {
		var sm models.StaffMember
		var userFullName, userEmail, userUsername, userRoleName sql.NullString
		if err := rows.Scan(
			&sm.ID, &sm.UserID, &sm.PhoneNumber, &sm.Address, &sm.HireDate, &sm.Position, &sm.Salary,
			&sm.CreatedAt, &sm.UpdatedAt,
			&userFullName, &userEmail, &userUsername, &userRoleName,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan staff member: " + err.Error()})
			return
		}
		if sm.UserID != nil {
			sm.User = &models.User{}
			if userFullName.Valid {
				sm.User.FullName = &userFullName.String
			}
			if userEmail.Valid {
				sm.User.Email = &userEmail.String
			}
			if userUsername.Valid {
				sm.User.Username = userUsername.String
			}
			if userRoleName.Valid {
				sm.User.Role = &models.Role{Name: userRoleName.String}
			}
		}
		staffList = append(staffList, sm)
	}
	c.JSON(http.StatusOK, staffList)
}

// GetStaffMemberByID handles fetching a single staff member by ID
func GetStaffMemberByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid staff ID"})
		return
	}

	db := database.GetDB()
	var sm models.StaffMember
	var userFullName, userEmail, userUsername, userRoleName sql.NullString
	queryStr := `
		SELECT 
			sm.id, sm.user_id, sm.phone_number, sm.address, sm.hire_date, sm.position, sm.salary, 
			sm.created_at, sm.updated_at,
			u.full_name as user_full_name, u.email as user_email, u.username as user_username,
			r.name as user_role_name
		FROM staff_members sm
		LEFT JOIN users u ON sm.user_id = u.id
		LEFT JOIN roles r ON u.role_id = r.id
		WHERE sm.id = $1`

	err = db.QueryRow(queryStr, id).Scan(
		&sm.ID, &sm.UserID, &sm.PhoneNumber, &sm.Address, &sm.HireDate, &sm.Position, &sm.Salary,
		&sm.CreatedAt, &sm.UpdatedAt,
		&userFullName, &userEmail, &userUsername, &userRoleName,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Staff member not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch staff member: " + err.Error()})
		return
	}

	if sm.UserID != nil {
		sm.User = &models.User{}
		if userFullName.Valid {
			sm.User.FullName = &userFullName.String
		}
		if userEmail.Valid {
			sm.User.Email = &userEmail.String
		}
		if userUsername.Valid {
			sm.User.Username = userUsername.String
		}
		if userRoleName.Valid {
			sm.User.Role = &models.Role{Name: userRoleName.String}
		}
	}
	c.JSON(http.StatusOK, sm)
}

// UpdateStaffMember handles updating an existing staff member
func UpdateStaffMember(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid staff ID"})
		return
	}

	var staff models.StaffMember
	if err := c.ShouldBindJSON(&staff); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	db := database.GetDB()
	query := `UPDATE staff_members SET 
	          user_id = $1, phone_number = $2, address = $3, hire_date = $4, 
	          position = $5, salary = $6, updated_at = $7
	          WHERE id = $8 
	          RETURNING id, user_id, phone_number, address, hire_date, position, salary, created_at, updated_at`

	staff.UpdatedAt = time.Now()

	err = db.QueryRow(query,
		staff.UserID, staff.PhoneNumber, staff.Address, staff.HireDate,
		staff.Position, staff.Salary, staff.UpdatedAt, id,
	).Scan(
		&staff.ID, &staff.UserID, &staff.PhoneNumber, &staff.Address, &staff.HireDate,
		&staff.Position, &staff.Salary, &staff.CreatedAt, &staff.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Staff member not found to update"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update staff member: " + err.Error()})
		return
	}
	staff.ID = id // Ensure ID from path is used
	c.JSON(http.StatusOK, staff)
}

// DeleteStaffMember handles deleting a staff member
// Consider implications: what happens to shifts, orders, bookings associated with this staff member?
// DB schema uses ON DELETE SET NULL or ON DELETE CASCADE for staff_id in related tables.
func DeleteStaffMember(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid staff ID"})
		return
	}

	db := database.GetDB()
	// Optional: Check if the staff member has an associated user account and decide if it should be deactivated/deleted.
	// For now, just delete the staff_member entry.
	result, err := db.Exec("DELETE FROM staff_members WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete staff member: " + err.Error()})
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Staff member not found to delete"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Staff member deleted successfully"})
}

// Shift Handlers

// CreateShift handles creation of a new shift
func CreateShift(c *gin.Context) {
	var shift models.Shift
	if err := c.ShouldBindJSON(&shift); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	if shift.EndTime.Before(shift.StartTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "End time cannot be before start time"})
		return
	}

	db := database.GetDB()
	query := `INSERT INTO shifts (staff_id, start_time, end_time, notes, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at, updated_at`

	shift.CreatedAt = time.Now()
	shift.UpdatedAt = time.Now()

	err := db.QueryRow(query, 
		shift.StaffID, shift.StartTime, shift.EndTime, shift.Notes, 
		shift.CreatedAt, shift.UpdatedAt,
	).Scan(&shift.ID, &shift.CreatedAt, &shift.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create shift: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, shift)
}

// GetShifts handles fetching all shifts, optionally filtered by staff_id or date range
func GetShifts(c *gin.Context) {
	db := database.GetDB()
	
	baseQuery := `
		SELECT s.id, s.staff_id, s.start_time, s.end_time, s.notes, s.created_at, s.updated_at,
		       sm.user_id, u.full_name as staff_full_name
		FROM shifts s
		JOIN staff_members sm ON s.staff_id = sm.id
		LEFT JOIN users u ON sm.user_id = u.id`
	
	var conditions []string
	var args []interface{}
	argCounter := 1

	staffIDStr := c.Query("staff_id")
	if staffIDStr != "" {
		staffID, err := strconv.ParseInt(staffIDStr, 10, 64)
		if err == nil {
			conditions = append(conditions, "s.staff_id = $" + strconv.Itoa(argCounter))
			args = append(args, staffID)
			argCounter++
		}
	}

	startDateStr := c.Query("start_date") // Expected format: YYYY-MM-DD
	endDateStr := c.Query("end_date")     // Expected format: YYYY-MM-DD

	if startDateStr != "" {
		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err == nil {
			conditions = append(conditions, "s.start_time >= $" + strconv.Itoa(argCounter))
			args = append(args, startDate)
			argCounter++
		}
	}
	if endDateStr != "" {
		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err == nil {
			// To include shifts on the end_date, we look for start_time < (endDate + 1 day)
			conditions = append(conditions, "s.start_time < $" + strconv.Itoa(argCounter))
			args = append(args, endDate.AddDate(0,0,1))
			argCounter++
		}
	}

	if len(conditions) > 0 {
		baseQuery += " WHERE " + string(join(conditions, " AND ")) // join function from inventory_handlers.go
	}
	baseQuery += " ORDER BY s.start_time DESC"

	rows, err := db.Query(baseQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch shifts: " + err.Error()})
		return
	}
	defer rows.Close()

	shifts := []models.Shift{}
	for rows.Next() {
		var sh models.Shift
		var staffUserID sql.NullInt64
		var staffFullName sql.NullString
		if err := rows.Scan(
			&sh.ID, &sh.StaffID, &sh.StartTime, &sh.EndTime, &sh.Notes, 
			&sh.CreatedAt, &sh.UpdatedAt,
			&staffUserID, &staffFullName,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan shift: " + err.Error()})
			return
		}
		sh.StaffMember = &models.StaffMember{ID: sh.StaffID}
		if staffUserID.Valid {
			sh.StaffMember.UserID = &staffUserID.Int64
		}
		if staffFullName.Valid {
			sh.StaffMember.User = &models.User{FullName: &staffFullName.String}
		}
		shifts = append(shifts, sh)
	}
	c.JSON(http.StatusOK, shifts)
}

// GetShiftByID handles fetching a single shift by ID
func GetShiftByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid shift ID"})
		return
	}

	db := database.GetDB()
	var sh models.Shift
	var staffUserID sql.NullInt64
	var staffFullName sql.NullString
	query := `
		SELECT s.id, s.staff_id, s.start_time, s.end_time, s.notes, s.created_at, s.updated_at,
		       sm.user_id, u.full_name as staff_full_name
		FROM shifts s
		JOIN staff_members sm ON s.staff_id = sm.id
		LEFT JOIN users u ON sm.user_id = u.id
		WHERE s.id = $1`
	err = db.QueryRow(query, id).Scan(
		&sh.ID, &sh.StaffID, &sh.StartTime, &sh.EndTime, &sh.Notes, 
		&sh.CreatedAt, &sh.UpdatedAt,
		&staffUserID, &staffFullName,
	)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Shift not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch shift: " + err.Error()})
		return
	}
	sh.StaffMember = &models.StaffMember{ID: sh.StaffID}
	if staffUserID.Valid {
		sh.StaffMember.UserID = &staffUserID.Int64
	}
	if staffFullName.Valid {
		sh.StaffMember.User = &models.User{FullName: &staffFullName.String}
	}
	c.JSON(http.StatusOK, sh)
}

// UpdateShift handles updating an existing shift
func UpdateShift(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid shift ID"})
		return
	}

	var shift models.Shift
	if err := c.ShouldBindJSON(&shift); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	if shift.EndTime.Before(shift.StartTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "End time cannot be before start time"})
		return
	}

	db := database.GetDB()
	query := `UPDATE shifts SET 
	          staff_id = $1, start_time = $2, end_time = $3, notes = $4, updated_at = $5
	          WHERE id = $6 
	          RETURNING id, staff_id, start_time, end_time, notes, created_at, updated_at`

	shift.UpdatedAt = time.Now()

	err = db.QueryRow(query, 
		shift.StaffID, shift.StartTime, shift.EndTime, shift.Notes, shift.UpdatedAt, id,
	).Scan(
		&shift.ID, &shift.StaffID, &shift.StartTime, &shift.EndTime, &shift.Notes, 
		&shift.CreatedAt, &shift.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Shift not found to update"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update shift: " + err.Error()})
		return
	}
	shift.ID = id // Ensure ID from path is used
	c.JSON(http.StatusOK, shift)
}

// DeleteShift handles deleting a shift
func DeleteShift(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid shift ID"})
		return
	}

	db := database.GetDB()
	result, err := db.Exec("DELETE FROM shifts WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete shift: " + err.Error()})
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Shift not found to delete"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Shift deleted successfully"})
}

