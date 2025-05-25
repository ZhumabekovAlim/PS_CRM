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

// Game Table Handlers

// CreateGameTable handles creation of a new game table
func CreateGameTable(c *gin.Context) {
	var table models.GameTable
	if err := c.ShouldBindJSON(&table); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	db := database.GetDB()
	query := `INSERT INTO game_tables (name, description, status, capacity, hourly_rate, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, created_at, updated_at`

	table.CreatedAt = time.Now()
	table.UpdatedAt = time.Now()
	if table.Status == "" {
		table.Status = "available" // Default status
	}

	err := db.QueryRow(query,
		table.Name, table.Description, table.Status, table.Capacity, table.HourlyRate,
		table.CreatedAt, table.UpdatedAt,
	).Scan(&table.ID, &table.CreatedAt, &table.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create game table: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, table)
}

// GetGameTables handles fetching all game tables
func GetGameTables(c *gin.Context) {
	db := database.GetDB()
	statusFilter := c.Query("status")

	queryStr := "SELECT id, name, description, status, capacity, hourly_rate, created_at, updated_at FROM game_tables"
	var args []interface{}
	if statusFilter != "" {
		queryStr += " WHERE status = $1"
		args = append(args, statusFilter)
	}
	queryStr += " ORDER BY name"

	rows, err := db.Query(queryStr, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch game tables: " + err.Error()})
		return
	}
	defer rows.Close()

	tables := []models.GameTable{}
	for rows.Next() {
		var tbl models.GameTable
		if err := rows.Scan(
			&tbl.ID, &tbl.Name, &tbl.Description, &tbl.Status, &tbl.Capacity, &tbl.HourlyRate,
			&tbl.CreatedAt, &tbl.UpdatedAt,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan game table: " + err.Error()})
			return
		}
		tables = append(tables, tbl)
	}
	c.JSON(http.StatusOK, tables)
}

// GetGameTableByID handles fetching a single game table by ID
func GetGameTableByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid table ID"})
		return
	}

	db := database.GetDB()
	var tbl models.GameTable
	query := "SELECT id, name, description, status, capacity, hourly_rate, created_at, updated_at FROM game_tables WHERE id = $1"
	err = db.QueryRow(query, id).Scan(
		&tbl.ID, &tbl.Name, &tbl.Description, &tbl.Status, &tbl.Capacity, &tbl.HourlyRate,
		&tbl.CreatedAt, &tbl.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game table not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch game table: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, tbl)
}

// UpdateGameTable handles updating an existing game table
func UpdateGameTable(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid table ID"})
		return
	}

	var table models.GameTable
	if err := c.ShouldBindJSON(&table); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	db := database.GetDB()
	query := `UPDATE game_tables SET 
	          name = $1, description = $2, status = $3, capacity = $4, hourly_rate = $5, updated_at = $6
	          WHERE id = $7 
	          RETURNING id, name, description, status, capacity, hourly_rate, created_at, updated_at`

	table.UpdatedAt = time.Now()

	err = db.QueryRow(query,
		table.Name, table.Description, table.Status, table.Capacity, table.HourlyRate,
		table.UpdatedAt, id,
	).Scan(
		&table.ID, &table.Name, &table.Description, &table.Status, &table.Capacity, &table.HourlyRate,
		&table.CreatedAt, &table.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game table not found to update"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update game table: " + err.Error()})
		return
	}
	table.ID = id // Ensure ID from path is used
	c.JSON(http.StatusOK, table)
}

// DeleteGameTable handles deleting a game table
func DeleteGameTable(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid table ID"})
		return
	}

	db := database.GetDB()
	// Check for active bookings associated with this table
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM bookings WHERE table_id = $1 AND status NOT IN ($2, $3)", id, "completed", "cancelled").Scan(&count)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check for active bookings: " + err.Error()})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Cannot delete table: it has active bookings. Please resolve bookings first."})
		return
	}

	result, err := db.Exec("DELETE FROM game_tables WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete game table: " + err.Error()})
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game table not found to delete"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Game table deleted successfully"})
}

// Booking Handlers

// CreateBooking handles creation of a new booking
func CreateBooking(c *gin.Context) {
	var booking models.Booking
	if err := c.ShouldBindJSON(&booking); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	if booking.EndTime.Before(booking.StartTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "End time cannot be before start time"})
		return
	}

	db := database.GetDB()

	// TODO: Add validation for overlapping bookings for the same table
	// TODO: Calculate TotalPrice based on table hourly_rate and duration if not provided

	query := `INSERT INTO bookings (client_id, table_id, staff_id, start_time, end_time, number_of_guests, status, notes, total_price, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) 
	          RETURNING id, created_at, updated_at`

	booking.CreatedAt = time.Now()
	booking.UpdatedAt = time.Now()
	if booking.Status == "" {
		booking.Status = "confirmed" // Default status
	}

	err := db.QueryRow(query,
		booking.ClientID, booking.TableID, booking.StaffID, booking.StartTime, booking.EndTime,
		booking.NumberOfGuests, booking.Status, booking.Notes, booking.TotalPrice,
		booking.CreatedAt, booking.UpdatedAt,
	).Scan(&booking.ID, &booking.CreatedAt, &booking.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create booking: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, booking)
}

// GetBookings handles fetching all bookings, with optional filters
func GetBookings(c *gin.Context) {
	db := database.GetDB()

	baseQuery := `
		SELECT 
			b.id, b.client_id, b.table_id, b.staff_id, b.start_time, b.end_time, 
			b.number_of_guests, b.status, b.notes, b.total_price, b.created_at, b.updated_at,
			cl.full_name as client_full_name, cl.phone_number as client_phone,
			gt.name as table_name,
			sm_u.full_name as staff_full_name
		FROM bookings b
		LEFT JOIN clients cl ON b.client_id = cl.id
		JOIN game_tables gt ON b.table_id = gt.id
		LEFT JOIN staff_members sm ON b.staff_id = sm.id
		LEFT JOIN users sm_u ON sm.user_id = sm_u.id
	`
	var conditions []string
	var args []interface{}
	argCounter := 1

	clientIDStr := c.Query("client_id")
	if clientIDStr != "" {
		conditions = append(conditions, "b.client_id = $"+strconv.Itoa(argCounter))
		args = append(args, clientIDStr)
		argCounter++
	}
	tableIDStr := c.Query("table_id")
	if tableIDStr != "" {
		conditions = append(conditions, "b.table_id = $"+strconv.Itoa(argCounter))
		args = append(args, tableIDStr)
		argCounter++
	}
	statusFilter := c.Query("status")
	if statusFilter != "" {
		conditions = append(conditions, "b.status = $"+strconv.Itoa(argCounter))
		args = append(args, statusFilter)
		argCounter++
	}
	dateFilter := c.Query("date") // Expects YYYY-MM-DD, filters bookings active on this date
	if dateFilter != "" {
		parsedDate, err := time.Parse("2006-01-02", dateFilter)
		if err == nil {
			conditions = append(conditions, "b.start_time <= $"+strconv.Itoa(argCounter)+" AND b.end_time >= $"+strconv.Itoa(argCounter+1))
			args = append(args, parsedDate.Add(23*time.Hour+59*time.Minute+59*time.Second)) // End of the day
			args = append(args, parsedDate) // Start of the day
			argCounter += 2
		}
	}

	if len(conditions) > 0 {
		baseQuery += " WHERE " + string(join(conditions, " AND ")) // join function from inventory_handlers.go
	}
	baseQuery += " ORDER BY b.start_time DESC"

	rows, err := db.Query(baseQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bookings: " + err.Error()})
		return
	}
	defer rows.Close()

	bookings := []models.Booking{}
	for rows.Next() {
		var bk models.Booking
		var clientFullName, clientPhone, tableName, staffFullName sql.NullString

		if err := rows.Scan(
			&bk.ID, &bk.ClientID, &bk.TableID, &bk.StaffID, &bk.StartTime, &bk.EndTime,
			&bk.NumberOfGuests, &bk.Status, &bk.Notes, &bk.TotalPrice,
			&bk.CreatedAt, &bk.UpdatedAt,
			&clientFullName, &clientPhone, &tableName, &staffFullName,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan booking: " + err.Error()})
			return
		}
		if bk.ClientID != nil {
			bk.Client = &models.Client{ID: *bk.ClientID}
			if clientFullName.Valid { bk.Client.FullName = clientFullName.String }
			if clientPhone.Valid { bk.Client.PhoneNumber = &clientPhone.String }
		}
		bk.GameTable = &models.GameTable{ID: bk.TableID}
		if tableName.Valid { bk.GameTable.Name = tableName.String }

		if bk.StaffID != nil {
			bk.StaffMember = &models.StaffMember{ID: *bk.StaffID}
			if staffFullName.Valid { bk.StaffMember.User = &models.User{FullName: &staffFullName.String} }
		}
		bookings = append(bookings, bk)
	}
	c.JSON(http.StatusOK, bookings)
}

// GetBookingByID handles fetching a single booking by ID
func GetBookingByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
		return
	}

	db := database.GetDB()
	var bk models.Booking
	var clientFullName, clientPhone, tableName, staffFullName sql.NullString

	query := `
		SELECT 
			b.id, b.client_id, b.table_id, b.staff_id, b.start_time, b.end_time, 
			b.number_of_guests, b.status, b.notes, b.total_price, b.created_at, b.updated_at,
			cl.full_name as client_full_name, cl.phone_number as client_phone,
			gt.name as table_name,
			sm_u.full_name as staff_full_name
		FROM bookings b
		LEFT JOIN clients cl ON b.client_id = cl.id
		JOIN game_tables gt ON b.table_id = gt.id
		LEFT JOIN staff_members sm ON b.staff_id = sm.id
		LEFT JOIN users sm_u ON sm.user_id = sm_u.id
		WHERE b.id = $1`
	err = db.QueryRow(query, id).Scan(
		&bk.ID, &bk.ClientID, &bk.TableID, &bk.StaffID, &bk.StartTime, &bk.EndTime,
		&bk.NumberOfGuests, &bk.Status, &bk.Notes, &bk.TotalPrice,
		&bk.CreatedAt, &bk.UpdatedAt,
		&clientFullName, &clientPhone, &tableName, &staffFullName,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch booking: " + err.Error()})
		return
	}

	if bk.ClientID != nil {
		bk.Client = &models.Client{ID: *bk.ClientID}
		if clientFullName.Valid { bk.Client.FullName = clientFullName.String }
		if clientPhone.Valid { bk.Client.PhoneNumber = &clientPhone.String }
	}
	bk.GameTable = &models.GameTable{ID: bk.TableID}
	if tableName.Valid { bk.GameTable.Name = tableName.String }

	if bk.StaffID != nil {
		bk.StaffMember = &models.StaffMember{ID: *bk.StaffID}
		if staffFullName.Valid { bk.StaffMember.User = &models.User{FullName: &staffFullName.String} }
	}
	c.JSON(http.StatusOK, bk)
}

// UpdateBooking handles updating an existing booking
func UpdateBooking(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
		return
	}

	var booking models.Booking
	if err := c.ShouldBindJSON(&booking); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	if booking.EndTime.Before(booking.StartTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "End time cannot be before start time"})
		return
	}

	db := database.GetDB()
	query := `UPDATE bookings SET 
	          client_id = $1, table_id = $2, staff_id = $3, start_time = $4, end_time = $5, 
	          number_of_guests = $6, status = $7, notes = $8, total_price = $9, updated_at = $10
	          WHERE id = $11 
	          RETURNING id, client_id, table_id, staff_id, start_time, end_time, number_of_guests, status, notes, total_price, created_at, updated_at`

	booking.UpdatedAt = time.Now()

	err = db.QueryRow(query,
		booking.ClientID, booking.TableID, booking.StaffID, booking.StartTime, booking.EndTime,
		booking.NumberOfGuests, booking.Status, booking.Notes, booking.TotalPrice,
		booking.UpdatedAt, id,
	).Scan(
		&booking.ID, &booking.ClientID, &booking.TableID, &booking.StaffID, &booking.StartTime, &booking.EndTime,
		&booking.NumberOfGuests, &booking.Status, &booking.Notes, &booking.TotalPrice,
		&booking.CreatedAt, &booking.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found to update"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update booking: " + err.Error()})
		return
	}
	booking.ID = id // Ensure ID from path is used
	c.JSON(http.StatusOK, booking)
}

// DeleteBooking handles deleting a booking
func DeleteBooking(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
		return
	}

	db := database.GetDB()
	// Usually bookings are not hard-deleted, but rather marked as 'cancelled'.
	// If hard delete is required:
	result, err := db.Exec("DELETE FROM bookings WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete booking: " + err.Error()})
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Booking not found to delete"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Booking deleted successfully"})
}

