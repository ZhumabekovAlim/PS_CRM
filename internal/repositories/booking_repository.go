package repositories

import (
	"database/sql"
	"errors"
	"fmt"
	"ps_club_backend/internal/models"
	"strings"
	"time"

	// "github.com/lib/pq" // Not strictly needed if not checking specific pq errors here
)

// BookingRepository defines the interface for booking-related database operations.
type BookingRepository interface {
	CreateBooking(executor SQLExecutor, booking *models.Booking) (*models.Booking, error)
	GetBookingByID(id int64) (*models.Booking, error) // Should join with client, table, staff (user)
	GetBookings(filters models.BookingFilters) ([]models.Booking, int, error) // Bookings, total count. Joins.
	UpdateBooking(executor SQLExecutor, booking *models.Booking) (*models.Booking, error)
	DeleteBooking(executor SQLExecutor, id int64) error
	CheckTableAvailability(tableID int64, startTime time.Time, endTime time.Time, excludeBookingID *int64) (bool, error) // True if available
}

type bookingRepository struct {
	db *sql.DB
}

// NewBookingRepository creates a new instance of BookingRepository.
func NewBookingRepository(db *sql.DB) BookingRepository {
	return &bookingRepository{db: db}
}

// scanBookingRow is a helper to scan a single booking row and its joined details.
// It's used by GetBookingByID and GetBookings.
func scanBookingRow(row scanner, isList bool) (*models.Booking, int, error) {
	var booking models.Booking
	var client models.Client
	var gameTable models.GameTable
	var staffMember models.StaffMember
	var user models.User // For StaffMember.User

	// Nullable fields for Client
	var clientFullName, clientPhone, clientEmail, clientNotes sql.NullString
	var clientDOB sql.NullTime
	var clientLoyaltyPoints sql.NullInt32 

	// Nullable fields for GameTable (though most are NOT NULL in DB, COALESCE for safety in JOINs)
	var gameTableName, gameTableDesc, gameTableStatus sql.NullString
	var gameTableCapacity sql.NullInt32
	var gameTableHourlyRate sql.NullFloat64
	
	// Nullable fields for StaffMember
	var staffUserID sql.NullInt64 // This is User.ID for the staff
	var staffPhone, staffAddr, staffHireDate, staffPos sql.NullString
	var staffSalary sql.NullFloat64

	// Nullable fields for User (linked to StaffMember)
	var staffUserUsername, staffUserEmail, staffUserFullName sql.NullString
	var staffUserRoleID sql.NullInt64 // User's RoleID
	var staffUserIsActive sql.NullBool

	// totalCount for list queries
	var totalCount int

	// Base booking fields
	scanDest := []interface{}{
		&booking.ID, &booking.ClientID, &booking.TableID, &booking.StaffID,
		&booking.StartTime, &booking.EndTime, &booking.NumberOfGuests, &booking.Status, &booking.Notes, &booking.TotalPrice,
		&booking.CreatedAt, &booking.UpdatedAt,
	}

	// Fields for Client join
	scanDest = append(scanDest, &client.ID, &clientFullName, &clientPhone, &clientEmail, &clientDOB, &clientLoyaltyPoints, &clientNotes, &client.CreatedAt, &client.UpdatedAt)
	// Fields for GameTable join
	scanDest = append(scanDest, &gameTable.ID, &gameTableName, &gameTableDesc, &gameTableStatus, &gameTableCapacity, &gameTableHourlyRate, &gameTable.CreatedAt, &gameTable.UpdatedAt)
	// Fields for StaffMember join
	scanDest = append(scanDest, &staffMember.ID, &staffUserID, &staffPhone, &staffAddr, &staffHireDate, &staffPos, &staffSalary, &staffMember.CreatedAt, &staffMember.UpdatedAt)
	// Fields for User join (for StaffMember)
	scanDest = append(scanDest, &user.ID, &staffUserUsername, &staffUserEmail, &staffUserFullName, &staffUserIsActive, &staffUserRoleID, &user.CreatedAt, &user.UpdatedAt)

	if isList {
		scanDest = append(scanDest, &totalCount)
	}

	err := row.Scan(scanDest...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, 0, ErrNotFound
		}
		return nil, 0, fmt.Errorf("%w: scanning booking with details: %v", ErrDatabaseError, err)
	}

	if booking.ClientID != nil { 
		client.FullName = clientFullName.String
		if clientPhone.Valid { client.PhoneNumber = &clientPhone.String }
		if clientEmail.Valid { client.Email = &clientEmail.String }
		if clientDOB.Valid {
			dateStr := clientDOB.Time.Format("2006-01-02")
			client.DateOfBirth = &dateStr
		} else {
			client.DateOfBirth = nil
		}
		if clientLoyaltyPoints.Valid { lp := int(clientLoyaltyPoints.Int32); client.LoyaltyPoints = &lp }
		if clientNotes.Valid { client.Notes = &clientNotes.String }
		booking.Client = &client
	} else { // Ensure client is nil if ClientID is nil
		booking.Client = nil
	}


	gameTable.Name = gameTableName.String
	if gameTableDesc.Valid { gameTable.Description = &gameTableDesc.String }
	if gameTableStatus.Valid { gameTable.Status = gameTableStatus.String }
	if gameTableCapacity.Valid { cap := int(gameTableCapacity.Int32); gameTable.Capacity = &cap }
	if gameTableHourlyRate.Valid { gameTable.HourlyRate = &gameTableHourlyRate.Float64 }
	booking.GameTable = &gameTable
	
	if booking.StaffID != nil { 
		if staffUserID.Valid { staffMember.UserID = &staffUserID.Int64 } else { staffMember.UserID = nil}
		if staffPhone.Valid { staffMember.PhoneNumber = &staffPhone.String }
		if staffAddr.Valid { staffMember.Address = &staffAddr.String }
		if staffHireDate.Valid { staffMember.HireDate = &staffHireDate.String }
		if staffPos.Valid { staffMember.Position = &staffPos.String }
		if staffSalary.Valid { staffMember.Salary = &staffSalary.Float64 }
		
		if staffUserID.Valid { // Only populate user if staffUserID (which is u.id) is valid
			user.ID = staffUserID.Int64 
			if staffUserUsername.Valid { user.Username = staffUserUsername.String }
			if staffUserEmail.Valid { user.Email = &staffUserEmail.String }
			if staffUserFullName.Valid { user.FullName = &staffUserFullName.String }
			if staffUserIsActive.Valid { user.IsActive = staffUserIsActive.Bool } else { user.IsActive = false }
			if staffUserRoleID.Valid { user.RoleID = &staffUserRoleID.Int64 } else { user.RoleID = nil }
			staffMember.User = &user
		} else {
			staffMember.User = nil // Ensure user is nil if staffUserID is nil
		}
		booking.StaffMember = &staffMember
	} else { // Ensure staff member is nil if StaffID is nil
		booking.StaffMember = nil
	}

	return &booking, totalCount, nil
}


func (r *bookingRepository) CreateBooking(executor SQLExecutor, booking *models.Booking) (*models.Booking, error) {
	query := `INSERT INTO bookings 
	            (client_id, table_id, staff_id, start_time, end_time, number_of_guests, status, notes, total_price, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	          RETURNING id, created_at, updated_at`
	
	currentTime := time.Now()
	booking.CreatedAt = currentTime
	booking.UpdatedAt = currentTime

	err := executor.QueryRow(query,
		booking.ClientID, booking.TableID, booking.StaffID, booking.StartTime, booking.EndTime,
		booking.NumberOfGuests, booking.Status, booking.Notes, booking.TotalPrice,
		booking.CreatedAt, booking.UpdatedAt,
	).Scan(&booking.ID, &booking.CreatedAt, &booking.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("%w: creating booking: %v", ErrDatabaseError, err)
	}
	return booking, nil
}

const getBookingJoins = `
	FROM bookings b
	LEFT JOIN clients c ON b.client_id = c.id
	JOIN game_tables gt ON b.table_id = gt.id
	LEFT JOIN staff_members sm ON b.staff_id = sm.id
	LEFT JOIN users u ON sm.user_id = u.id
`
const selectBookingFields = `
	b.id, b.client_id, b.table_id, b.staff_id, b.start_time, b.end_time, 
	b.number_of_guests, b.status, b.notes, b.total_price, b.created_at, b.updated_at,
	COALESCE(c.id, 0), COALESCE(c.full_name, ''), COALESCE(c.phone_number, ''), COALESCE(c.email, ''), c.date_of_birth, COALESCE(c.loyalty_points, 0), COALESCE(c.notes, ''), COALESCE(c.created_at, '0001-01-01'::timestamp), COALESCE(c.updated_at, '0001-01-01'::timestamp),
	gt.id, gt.name, gt.description, gt.status, gt.capacity, gt.hourly_rate, gt.created_at, gt.updated_at,
	COALESCE(sm.id, 0), sm.user_id, COALESCE(sm.phone_number, ''), COALESCE(sm.address, ''), COALESCE(sm.hire_date, ''), COALESCE(sm.position, ''), COALESCE(sm.salary, 0), COALESCE(sm.created_at, '0001-01-01'::timestamp), COALESCE(sm.updated_at, '0001-01-01'::timestamp),
	COALESCE(u.id, 0), COALESCE(u.username, ''), COALESCE(u.email, ''), COALESCE(u.full_name, ''), COALESCE(u.is_active, false), u.role_id, COALESCE(u.created_at, '0001-01-01'::timestamp), COALESCE(u.updated_at, '0001-01-01'::timestamp)
`


func (r *bookingRepository) GetBookingByID(id int64) (*models.Booking, error) {
	query := "SELECT " + selectBookingFields + getBookingJoins + " WHERE b.id = $1"
	booking, _, err := scanBookingRow(r.db.QueryRow(query, id), false)
	return booking, err
}

func (r *bookingRepository) GetBookings(filters models.BookingFilters) ([]models.Booking, int, error) {
	bookings := []models.Booking{}
	var totalCount int // Initialize totalCount

	var queryBuilder strings.Builder
	queryBuilder.WriteString("SELECT " + selectBookingFields + ", COUNT(*) OVER() as total_count " + getBookingJoins)

	var conditions []string
	var args []interface{}
	argCount := 1

	if filters.ClientID != nil { conditions = append(conditions, fmt.Sprintf("b.client_id = $%d", argCount)); args = append(args, *filters.ClientID); argCount++ }
	if filters.TableID != nil { conditions = append(conditions, fmt.Sprintf("b.table_id = $%d", argCount)); args = append(args, *filters.TableID); argCount++ }
	if filters.StaffID != nil { conditions = append(conditions, fmt.Sprintf("b.staff_id = $%d", argCount)); args = append(args, *filters.StaffID); argCount++ }
	if filters.Status != nil && *filters.Status != "" { conditions = append(conditions, fmt.Sprintf("b.status = $%d", argCount)); args = append(args, *filters.Status); argCount++ }
	if filters.DateFrom != nil { conditions = append(conditions, fmt.Sprintf("b.start_time >= $%d", argCount)); args = append(args, *filters.DateFrom); argCount++ }
	if filters.DateTo != nil { conditions = append(conditions, fmt.Sprintf("b.end_time <= $%d", argCount)); args = append(args, *filters.DateTo); argCount++ }


	if len(conditions) > 0 {
		queryBuilder.WriteString(" WHERE " + strings.Join(conditions, " AND "))
	}
	queryBuilder.WriteString(" ORDER BY b.start_time DESC")

	if filters.PageSize > 0 {
		queryBuilder.WriteString(fmt.Sprintf(" LIMIT $%d", argCount)); args = append(args, filters.PageSize); argCount++
		if filters.Page > 0 {
			offset := (filters.Page - 1) * filters.PageSize
			queryBuilder.WriteString(fmt.Sprintf(" OFFSET $%d", argCount)); args = append(args, offset)
		}
	}

	rows, err := r.db.Query(queryBuilder.String(), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: querying bookings: %v", ErrDatabaseError, err)
	}
	defer rows.Close()

	for rows.Next() {
		booking, scannedTotalCount, scanErr := scanBookingRow(rows, true)
		if scanErr != nil {
			return nil, 0, scanErr // Error already wrapped in scanBookingRow
		}
		bookings = append(bookings, *booking)
		totalCount = scannedTotalCount // total_count is the same for all rows from OVER()
	}
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("%w: iterating booking rows: %v", ErrDatabaseError, err)
	}
	if len(bookings) == 0 { // If no results, totalCount from OVER() would be 0
		totalCount = 0
	}
	return bookings, totalCount, nil
}


func (r *bookingRepository) UpdateBooking(executor SQLExecutor, booking *models.Booking) (*models.Booking, error) {
	query := `UPDATE bookings SET 
	            client_id = $1, table_id = $2, staff_id = $3, start_time = $4, end_time = $5, 
	            number_of_guests = $6, status = $7, notes = $8, total_price = $9, updated_at = $10
	          WHERE id = $11
	          RETURNING updated_at`
	booking.UpdatedAt = time.Now()

	err := executor.QueryRow(query,
		booking.ClientID, booking.TableID, booking.StaffID, booking.StartTime, booking.EndTime,
		booking.NumberOfGuests, booking.Status, booking.Notes, booking.TotalPrice,
		booking.UpdatedAt, booking.ID,
	).Scan(&booking.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("%w: updating booking ID %d: %v", ErrDatabaseError, booking.ID, err)
	}
	return booking, nil
}

func (r *bookingRepository) DeleteBooking(executor SQLExecutor, id int64) error {
	query := `DELETE FROM bookings WHERE id = $1`
	result, err := executor.Exec(query, id)
	if err != nil {
		return fmt.Errorf("%w: deleting booking ID %d: %v", ErrDatabaseError, id, err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *bookingRepository) CheckTableAvailability(tableID int64, startTime time.Time, endTime time.Time, excludeBookingID *int64) (bool, error) {
	// Booking statuses that mean the table is occupied or unavailable for new bookings
	activeBookingStatuses := []string{models.BookingStatusConfirmed, models.BookingStatusCheckedIn /*, models.BookingStatusPending? - depends on rules */}
	
	var statusPlaceholders []string
	args := []interface{}{tableID, startTime, endTime}
	argIdx := 4 // Start after tableID, startTime, endTime

	for _, status := range activeBookingStatuses {
		statusPlaceholders = append(statusPlaceholders, fmt.Sprintf("$%d", argIdx))
		args = append(args, status)
		argIdx++
	}
	
	statusInClause := strings.Join(statusPlaceholders, ", ")


	query := fmt.Sprintf(`SELECT COUNT(*) FROM bookings 
	          WHERE table_id = $1 
	          AND status IN (%s)
	          AND start_time < $3 AND end_time > $2`, statusInClause) // Overlapping condition
	          
	if excludeBookingID != nil {
		query += fmt.Sprintf(" AND id != $%d", argIdx)
		args = append(args, *excludeBookingID)
	}

	var count int
	err := r.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("%w: checking table availability: %v", ErrDatabaseError, err)
	}
	return count == 0, nil 
}
