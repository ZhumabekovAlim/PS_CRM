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

// StaffRepository defines the interface for staff and shift related database operations.
type StaffRepository interface {
	// StaffMember methods
	CreateStaffMember(executor SQLExecutor, staff *models.StaffMember) (*models.StaffMember, error)
	GetStaffMemberByID(id int64) (*models.StaffMember, error)
	GetStaffMemberByUserID(userID int64) (*models.StaffMember, error)
	GetStaffMembers(page, pageSize int, searchTerm *string) ([]models.StaffMember, int, error)
	UpdateStaffMember(executor SQLExecutor, staff *models.StaffMember) (*models.StaffMember, error)
	DeleteStaffMember(executor SQLExecutor, id int64) error

	// Shift methods
	CreateShift(executor SQLExecutor, shift *models.Shift) (*models.Shift, error)
	GetShiftByID(id int64) (*models.Shift, error)
	GetShifts(staffID *int64, startTimeFrom *time.Time, startTimeTo *time.Time, page, pageSize int) ([]models.Shift, int, error)
	UpdateShift(executor SQLExecutor, shift *models.Shift) (*models.Shift, error)
	DeleteShift(executor SQLExecutor, id int64) error
}

type staffRepository struct {
	db *sql.DB
}

// NewStaffRepository creates a new instance of StaffRepository.
func NewStaffRepository(db *sql.DB) StaffRepository {
	return &staffRepository{db: db}
}

// --- StaffMember Methods ---

func (r *staffRepository) CreateStaffMember(executor SQLExecutor, staff *models.StaffMember) (*models.StaffMember, error) {
	query := `INSERT INTO staff_members (user_id, phone_number, address, hire_date, position, salary, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	          RETURNING id, created_at, updated_at`
	
	currentTime := time.Now()
	staff.CreatedAt = currentTime
	staff.UpdatedAt = currentTime

	var hireDate sql.NullString
	if staff.HireDate != nil {
		hireDate = sql.NullString{String: *staff.HireDate, Valid: true}
	}

	err := executor.QueryRow(query,
		staff.UserID, staff.PhoneNumber, staff.Address, hireDate,
		staff.Position, staff.Salary, staff.CreatedAt, staff.UpdatedAt,
	).Scan(&staff.ID, &staff.CreatedAt, &staff.UpdatedAt)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			if pqErr.Code.Name() == "unique_violation" && pqErr.Constraint == "staff_members_user_id_key" {
				return nil, fmt.Errorf("%w: user_id %d is already associated with another staff member", ErrDuplicateKey, *staff.UserID)
			}
			if pqErr.Code.Name() == "foreign_key_violation" && pqErr.Constraint == "staff_members_user_id_fkey" {
				return nil, fmt.Errorf("%w: user with ID %d not found", ErrNotFound, *staff.UserID)
			}
		}
		return nil, fmt.Errorf("%w: creating staff member: %v", ErrDatabaseError, err)
	}
	return staff, nil
}

// scanStaffMemberRow scans a single row into a StaffMember, typically used by GetStaffMemberByID etc.
// It expects the query to join users and roles table.
func scanStaffMemberRow(row scanner) (*models.StaffMember, error) {
    var staff models.StaffMember
    var user models.User
    var role models.Role
    var hireDate sql.NullString
    var userEmail, userFullName, roleName sql.NullString
    var userRoleID sql.NullInt64

    err := row.Scan(
        &staff.ID, &staff.UserID, &staff.PhoneNumber, &staff.Address, &hireDate,
        &staff.Position, &staff.Salary, &staff.CreatedAt, &staff.UpdatedAt,
        &user.ID, &user.Username, &userEmail, &userFullName, &userRoleID, &user.IsActive,
        &user.CreatedAt, &user.UpdatedAt, &roleName,
    )
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrNotFound
        }
        return nil, fmt.Errorf("%w: scanning staff member with user details: %v", ErrDatabaseError, err)
    }

    if hireDate.Valid { staff.HireDate = &hireDate.String }
    if userEmail.Valid { user.Email = &userEmail.String }
    if userFullName.Valid { user.FullName = &userFullName.String }
    if userRoleID.Valid {
        user.RoleID = &userRoleID.Int64
        if roleName.Valid {
            role.ID = *user.RoleID // Assuming role ID is the same as user's role_id
            role.Name = roleName.String
            user.Role = &role
        }
    }
    staff.User = &user
    return &staff, nil
}

// scanner is an interface compatible with *sql.Row and *sql.Rows.
type scanner interface {
    Scan(dest ...interface{}) error
}


func (r *staffRepository) GetStaffMemberByID(id int64) (*models.StaffMember, error) {
	query := `SELECT 
	            sm.id, sm.user_id, sm.phone_number, sm.address, sm.hire_date, 
	            sm.position, sm.salary, sm.created_at, sm.updated_at,
	            u.id as user_id_fk, u.username, u.email, u.full_name, u.role_id, u.is_active,
	            u.created_at as user_created_at, u.updated_at as user_updated_at,
				COALESCE(r.name, '') as role_name
	          FROM staff_members sm
	          LEFT JOIN users u ON sm.user_id = u.id
			  LEFT JOIN roles r ON u.role_id = r.id
	          WHERE sm.id = $1`
	return scanStaffMemberRow(r.db.QueryRow(query, id))
}

func (r *staffRepository) GetStaffMemberByUserID(userID int64) (*models.StaffMember, error) {
	query := `SELECT 
	            sm.id, sm.user_id, sm.phone_number, sm.address, sm.hire_date, 
	            sm.position, sm.salary, sm.created_at, sm.updated_at,
	            u.id as user_id_fk, u.username, u.email, u.full_name, u.role_id, u.is_active,
	            u.created_at as user_created_at, u.updated_at as user_updated_at,
				COALESCE(r.name, '') as role_name
	          FROM staff_members sm
	          LEFT JOIN users u ON sm.user_id = u.id
			  LEFT JOIN roles r ON u.role_id = r.id
	          WHERE sm.user_id = $1`
	return scanStaffMemberRow(r.db.QueryRow(query, userID))
}

func (r *staffRepository) GetStaffMembers(page, pageSize int, searchTerm *string) ([]models.StaffMember, int, error) {
	staffMembers := []models.StaffMember{}
	totalCount := 0

	var queryBuilder strings.Builder
	queryBuilder.WriteString(`SELECT 
	    sm.id, sm.user_id, sm.phone_number, sm.address, sm.hire_date, 
	    sm.position, sm.salary, sm.created_at, sm.updated_at,
	    u.id as user_id_fk, u.username, u.email, u.full_name, u.role_id, u.is_active,
	    u.created_at as user_created_at, u.updated_at as user_updated_at,
		COALESCE(r.name, '') as role_name,
	    COUNT(*) OVER() as total_count
	  FROM staff_members sm
	  LEFT JOIN users u ON sm.user_id = u.id
	  LEFT JOIN roles r ON u.role_id = r.id`)

	var conditions []string
	var args []interface{}
	argCount := 1

	if searchTerm != nil && *searchTerm != "" {
		searchPattern := "%" + strings.ToLower(*searchTerm) + "%"
		conditions = append(conditions, fmt.Sprintf("(LOWER(u.full_name) ILIKE $%d OR LOWER(u.email) ILIKE $%d OR LOWER(sm.phone_number) ILIKE $%d OR LOWER(sm.position) ILIKE $%d)", argCount, argCount, argCount, argCount))
		args = append(args, searchPattern)
		argCount++
	}

	if len(conditions) > 0 {
		queryBuilder.WriteString(" WHERE " + strings.Join(conditions, " AND "))
	}
	queryBuilder.WriteString(" ORDER BY u.full_name ASC")

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
		return nil, 0, fmt.Errorf("%w: querying staff members: %v", ErrDatabaseError, err)
	}
	defer rows.Close()

	for rows.Next() {
		var staff models.StaffMember
		var user models.User
		var role models.Role
		var hireDate sql.NullString
		var userEmail, userFullName, roleName sql.NullString
		var userRoleID sql.NullInt64
		// Must scan totalCount from each row when using COUNT(*) OVER()
		var currentRowTotalCount int 

		err := rows.Scan(
			&staff.ID, &staff.UserID, &staff.PhoneNumber, &staff.Address, &hireDate,
			&staff.Position, &staff.Salary, &staff.CreatedAt, &staff.UpdatedAt,
			&user.ID, &user.Username, &userEmail, &userFullName, &userRoleID, &user.IsActive,
			&user.CreatedAt, &user.UpdatedAt, &roleName,
			&currentRowTotalCount, // Scan total_count from each row
		)
		if err != nil {
			return nil, 0, fmt.Errorf("%w: scanning staff member from list: %v", ErrDatabaseError, err)
		}
		totalCount = currentRowTotalCount // total_count is the same for all rows in this query

		if hireDate.Valid { staff.HireDate = &hireDate.String }
		if userEmail.Valid { user.Email = &userEmail.String }
		if userFullName.Valid { user.FullName = &userFullName.String }
		if userRoleID.Valid {
			user.RoleID = &userRoleID.Int64
			if roleName.Valid {
				role.ID = *user.RoleID
				role.Name = roleName.String
				user.Role = &role
			}
		}
		staff.User = &user
		staffMembers = append(staffMembers, staff)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("%w: iterating staff member rows: %v", ErrDatabaseError, err)
	}
	return staffMembers, totalCount, nil
}


func (r *staffRepository) UpdateStaffMember(executor SQLExecutor, staff *models.StaffMember) (*models.StaffMember, error) {
	query := `UPDATE staff_members SET 
	            phone_number = $1, address = $2, hire_date = $3, 
	            position = $4, salary = $5, updated_at = $6 
	          WHERE id = $7
	          RETURNING updated_at` 
	
	staff.UpdatedAt = time.Now()
	var hireDate sql.NullString
	if staff.HireDate != nil {
		hireDate = sql.NullString{String: *staff.HireDate, Valid: true}
	}

	err := executor.QueryRow(query,
		staff.PhoneNumber, staff.Address, hireDate, staff.Position,
		staff.Salary, staff.UpdatedAt, staff.ID,
	).Scan(&staff.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("%w: updating staff member ID %d: %v", ErrDatabaseError, staff.ID, err)
	}
	return staff, nil
}

func (r *staffRepository) DeleteStaffMember(executor SQLExecutor, id int64) error {
	query := `DELETE FROM staff_members WHERE id = $1`
	result, err := executor.Exec(query, id)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23503" { 
			return fmt.Errorf("%w: staff member ID %d cannot be deleted as they are referenced in other records (constraint: %s)", ErrDatabaseError, id, pqErr.Constraint)
		}
		return fmt.Errorf("%w: deleting staff member ID %d: %v", ErrDatabaseError, id, err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// --- Shift Methods ---

func (r *staffRepository) CreateShift(executor SQLExecutor, shift *models.Shift) (*models.Shift, error) {
	query := `INSERT INTO shifts (staff_id, start_time, end_time, notes, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6)
	          RETURNING id, created_at, updated_at`
	currentTime := time.Now()
	shift.CreatedAt = currentTime
	shift.UpdatedAt = currentTime

	err := executor.QueryRow(query,
		shift.StaffID, shift.StartTime, shift.EndTime, shift.Notes,
		shift.CreatedAt, shift.UpdatedAt,
	).Scan(&shift.ID, &shift.CreatedAt, &shift.UpdatedAt)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code.Name() == "foreign_key_violation" {
			return nil, fmt.Errorf("%w: creating shift (staff_id %d likely not found, constraint: %s): %v", ErrNotFound, shift.StaffID, pqErr.Constraint, err)
		}
		return nil, fmt.Errorf("%w: creating shift: %v", ErrDatabaseError, err)
	}
	return shift, nil
}

func (r *staffRepository) GetShiftByID(id int64) (*models.Shift, error) {
	shift := &models.Shift{}
	query := `SELECT s.id, s.staff_id, s.start_time, s.end_time, s.notes, s.created_at, s.updated_at,
			         sm.user_id, u.full_name as staff_full_name
	          FROM shifts s
			  JOIN staff_members sm ON s.staff_id = sm.id
			  JOIN users u ON sm.user_id = u.id
			  WHERE s.id = $1`
			  
	var staffMember models.StaffMember
	var user models.User
	var staffFullName sql.NullString

	err := r.db.QueryRow(query, id).Scan(
		&shift.ID, &shift.StaffID, &shift.StartTime, &shift.EndTime, &shift.Notes,
		&shift.CreatedAt, &shift.UpdatedAt,
		&staffMember.UserID, 
		&staffFullName,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("%w: getting shift by ID %d: %v", ErrDatabaseError, id, err)
	}
	staffMember.ID = shift.StaffID
	if staffFullName.Valid { user.FullName = &staffFullName.String }
	staffMember.User = &user
	shift.StaffMember = &staffMember
	
	return shift, nil
}

func (r *staffRepository) GetShifts(staffID *int64, startTimeFrom *time.Time, startTimeTo *time.Time, page, pageSize int) ([]models.Shift, int, error) {
	shifts := []models.Shift{}
	totalCount := 0

	var queryBuilder strings.Builder
	queryBuilder.WriteString(`SELECT 
	    s.id, s.staff_id, s.start_time, s.end_time, s.notes, s.created_at, s.updated_at,
	    sm.user_id as staff_user_id, u.full_name as staff_full_name,
	    COUNT(*) OVER() as total_count
	  FROM shifts s
	  JOIN staff_members sm ON s.staff_id = sm.id
	  JOIN users u ON sm.user_id = u.id`)

	var conditions []string
	var args []interface{}
	argCount := 1

	if staffID != nil {
		conditions = append(conditions, fmt.Sprintf("s.staff_id = $%d", argCount))
		args = append(args, *staffID)
		argCount++
	}
	if startTimeFrom != nil {
		conditions = append(conditions, fmt.Sprintf("s.start_time >= $%d", argCount))
		args = append(args, *startTimeFrom)
		argCount++
	}
	if startTimeTo != nil {
		conditions = append(conditions, fmt.Sprintf("s.end_time <= $%d", argCount))
		args = append(args, *startTimeTo)
		argCount++
	}

	if len(conditions) > 0 {
		queryBuilder.WriteString(" WHERE " + strings.Join(conditions, " AND "))
	}
	queryBuilder.WriteString(" ORDER BY s.start_time DESC")

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
		return nil, 0, fmt.Errorf("%w: querying shifts: %v", ErrDatabaseError, err)
	}
	defer rows.Close()

	for rows.Next() {
		var shift models.Shift
		var staffMember models.StaffMember
		var user models.User
		var staffFullName sql.NullString
		var currentTotalCount int

		if err := rows.Scan(
			&shift.ID, &shift.StaffID, &shift.StartTime, &shift.EndTime, &shift.Notes,
			&shift.CreatedAt, &shift.UpdatedAt,
			&staffMember.UserID, 
			&staffFullName,      
			&currentTotalCount,
		); err != nil {
			return nil, 0, fmt.Errorf("%w: scanning shift: %v", ErrDatabaseError, err)
		}
		totalCount = currentTotalCount
		
		staffMember.ID = shift.StaffID 
		if staffFullName.Valid {
			name := staffFullName.String
			user.FullName = &name
		}
		staffMember.User = &user
		shift.StaffMember = &staffMember
		
		shifts = append(shifts, shift)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("%w: iterating shift rows: %v", ErrDatabaseError, err)
	}
	return shifts, totalCount, nil
}

func (r *staffRepository) UpdateShift(executor SQLExecutor, shift *models.Shift) (*models.Shift, error) {
	query := `UPDATE shifts SET 
	            staff_id = $1, start_time = $2, end_time = $3, notes = $4, updated_at = $5 
	          WHERE id = $6
	          RETURNING updated_at`
	shift.UpdatedAt = time.Now()

	err := executor.QueryRow(query,
		shift.StaffID, shift.StartTime, shift.EndTime, shift.Notes,
		shift.UpdatedAt, shift.ID,
	).Scan(&shift.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code.Name() == "foreign_key_violation" {
			return nil, fmt.Errorf("%w: updating shift (staff_id %d likely not found, constraint: %s): %v", ErrNotFound, shift.StaffID, pqErr.Constraint, err)
		}
		return nil, fmt.Errorf("%w: updating shift ID %d: %v", ErrDatabaseError, shift.ID, err)
	}
	return shift, nil
}

func (r *staffRepository) DeleteShift(executor SQLExecutor, id int64) error {
	query := `DELETE FROM shifts WHERE id = $1`
	result, err := executor.Exec(query, id)
	if err != nil {
		return fmt.Errorf("%w: deleting shift ID %d: %v", ErrDatabaseError, id, err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
