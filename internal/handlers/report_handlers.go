package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ps_club_backend/internal/database"
	"ps_club_backend/internal/models"

	"github.com/gin-gonic/gin"
)

const DefaultReportDateLayout = "2006-01-02"

// parseReportRequestParams helps parse common query parameters for reports.
func parseReportRequestParams(c *gin.Context) models.ReportRequestParams {
	var params models.ReportRequestParams
	params.StartDate = c.Query("start_date")
	params.EndDate = c.Query("end_date")
	params.Period = c.Query("period") // daily, weekly, monthly, custom
	params.Granularity = c.Query("granularity") // hourly, daily

	if itemIDStr := c.Query("item_id"); itemIDStr != "" {
		if id, err := strconv.ParseInt(itemIDStr, 10, 64); err == nil {
			params.ItemID = &id
		}
	}
	if categoryIDStr := c.Query("category_id"); categoryIDStr != "" {
		if id, err := strconv.ParseInt(categoryIDStr, 10, 64); err == nil {
			params.CategoryID = &id
		}
	}
	if tableIDStr := c.Query("table_id"); tableIDStr != "" {
		if id, err := strconv.ParseInt(tableIDStr, 10, 64); err == nil {
			params.TableID = &id
		}
	}
	if staffIDStr := c.Query("staff_id"); staffIDStr != "" {
		if id, err := strconv.ParseInt(staffIDStr, 10, 64); err == nil {
			params.StaffID = &id
		}
	}
	return params
}

// GetDashboardSummary provides a summary of key metrics for the dashboard.
func GetDashboardSummary(c *gin.Context) {
	db := database.GetDB()
	var summary models.DashboardSummary
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.AddDate(0, 0, 1).Add(-time.Nanosecond)

	startOfWeek := startOfDay.AddDate(0, 0, -int(startOfDay.Weekday())+1) // Assuming Monday is the start of the week
	if startOfDay.Weekday() == time.Sunday { // Adjust if Sunday is considered start of week by system
		startOfWeek = startOfDay.AddDate(0,0,-6)
	}
	endOfWeek := startOfWeek.AddDate(0,0,7).Add(-time.Nanosecond)

	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Nanosecond)

	// Active Bookings Count
	err := db.QueryRow(`SELECT COUNT(*) FROM bookings WHERE status = 'active' AND start_time <= $1 AND end_time >= $1`, now).Scan(&summary.ActiveBookingsCount)
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get active bookings count: " + err.Error()})
		return
	}

	// Pending Orders Count
	err = db.QueryRow(`SELECT COUNT(*) FROM orders WHERE status = 'pending' OR status = 'preparing'`).Scan(&summary.PendingOrdersCount)
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get pending orders count: " + err.Error()})
		return
	}

	// Total Sales Today
	err = db.QueryRow(`SELECT COALESCE(SUM(final_amount), 0) FROM orders WHERE status = 'completed' AND order_time BETWEEN $1 AND $2`, startOfDay, endOfDay).Scan(&summary.TotalSalesToday)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get total sales today: " + err.Error()})
		return
	}

	// Total Sales This Week
	err = db.QueryRow(`SELECT COALESCE(SUM(final_amount), 0) FROM orders WHERE status = 'completed' AND order_time BETWEEN $1 AND $2`, startOfWeek, endOfWeek).Scan(&summary.TotalSalesThisWeek)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get total sales this week: " + err.Error()})
		return
	}

	// Total Sales This Month
	err = db.QueryRow(`SELECT COALESCE(SUM(final_amount), 0) FROM orders WHERE status = 'completed' AND order_time BETWEEN $1 AND $2`, startOfMonth, endOfMonth).Scan(&summary.TotalSalesThisMonth)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get total sales this month: " + err.Error()})
		return
	}

	// Low Stock Items Count
	err = db.QueryRow(`SELECT COUNT(*) FROM pricelist_items WHERE current_stock IS NOT NULL AND low_stock_threshold IS NOT NULL AND current_stock <= low_stock_threshold AND is_available = TRUE`).Scan(&summary.LowStockItemsCount)
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get low stock items count: " + err.Error()})
		return
	}

	// Upcoming Bookings Count (e.g., next 24 hours)
	upcomingEndTime := now.Add(24 * time.Hour)
	err = db.QueryRow(`SELECT COUNT(*) FROM bookings WHERE status = 'confirmed' AND start_time BETWEEN $1 AND $2`, now, upcomingEndTime).Scan(&summary.UpcomingBookingsCount)
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get upcoming bookings count: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// GetSalesReports generates sales reports based on query parameters.
func GetSalesReports(c *gin.Context) {
	params := parseReportRequestParams(c)
	db := database.GetDB()

	var queryBuilder strings.Builder
	args := []interface{}{}
	argIdx := 1

	queryBuilder.WriteString(`
		SELECT 
			TO_CHAR(o.order_time, $` + strconv.Itoa(argIdx) + `) as report_date,
			oi.pricelist_item_id,
			pi.name as item_name,
			pi.category_id,
			pc.name as category_name,
			SUM(oi.quantity) as total_quantity,
			SUM(oi.total_price) as total_sales,
			SUM(o.discount_amount / (SELECT COUNT(*) FROM order_items WHERE order_id = o.id)) as estimated_item_discount, -- Approximate discount per item
			SUM(oi.total_price - (o.discount_amount / (SELECT COUNT(*) FROM order_items WHERE order_id = o.id))) as net_sales
		FROM orders o
		JOIN order_items oi ON o.id = oi.order_id
		JOIN pricelist_items pi ON oi.pricelist_item_id = pi.id
		LEFT JOIN pricelist_categories pc ON pi.category_id = pc.id
		WHERE o.status = 'completed'
	`)

	dateFormat := "YYYY-MM-DD" // Default daily
	groupByClause := "report_date, oi.pricelist_item_id, pi.name, pi.category_id, pc.name"

	switch params.Period {
	case "weekly":
		dateFormat = "IYYY-IW" // ISO Year and Week number
	case "monthly":
		dateFormat = "YYYY-MM"
	}
	args = append(args, dateFormat)
	argIdx++

	if params.StartDate != "" {
		queryBuilder.WriteString(" AND o.order_time >= $" + strconv.Itoa(argIdx))
		args = append(args, params.StartDate)
		argIdx++
	}
	if params.EndDate != "" {
		// Adjust end date to include the whole day
		endDateParsed, err := time.Parse(DefaultReportDateLayout, params.EndDate)
		if err == nil {
			endDateAdjusted := endDateParsed.AddDate(0,0,1).Format(DefaultReportDateLayout)
			queryBuilder.WriteString(" AND o.order_time < $" + strconv.Itoa(argIdx))
			args = append(args, endDateAdjusted)
			argIdx++
		} else {
			queryBuilder.WriteString(" AND o.order_time <= $" + strconv.Itoa(argIdx))
			args = append(args, params.EndDate)
			argIdx++
		}
	}
	if params.ItemID != nil {
		queryBuilder.WriteString(" AND oi.pricelist_item_id = $" + strconv.Itoa(argIdx))
		args = append(args, *params.ItemID)
		argIdx++
	}
	if params.CategoryID != nil {
		queryBuilder.WriteString(" AND pi.category_id = $" + strconv.Itoa(argIdx))
		args = append(args, *params.CategoryID)
		argIdx++
	}

	queryBuilder.WriteString(" GROUP BY " + groupByClause)
	queryBuilder.WriteString(" ORDER BY report_date DESC, net_sales DESC")

	rows, err := db.Query(queryBuilder.String(), args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query sales report: " + err.Error(), "query": queryBuilder.String()})
		return
	}
	defer rows.Close()

	reportItems := []models.SalesReportItem{}
	for rows.Next() {
		var item models.SalesReportItem
		var estimatedDiscount sql.NullFloat64
		if err := rows.Scan(
			&item.Date,
			&item.ItemID,
			&item.ItemName,
			&item.CategoryID,
			&item.CategoryName,
			&item.TotalQuantity,
			&item.TotalSales,
			&estimatedDiscount,
			&item.NetSales,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan sales report item: " + err.Error()})
			return
		}
		if estimatedDiscount.Valid {
			item.TotalDiscount = estimatedDiscount.Float64
		}
		reportItems = append(reportItems, item)
	}

	c.JSON(http.StatusOK, reportItems)
}

// GetBookingReports generates booking reports.
func GetBookingReports(c *gin.Context) {
	params := parseReportRequestParams(c)
	db := database.GetDB()

	var queryBuilder strings.Builder
	args := []interface{}{}
	argIdx := 1

	selectClause := `
		SELECT 
			TO_CHAR(b.start_time, 'YYYY-MM-DD') as report_date,
			b.table_id,
			gt.name as table_name,
	`
	groupByClause := "report_date, b.table_id, gt.name"

	if params.Granularity == "hourly" {
		selectClause += " EXTRACT(HOUR FROM b.start_time) as hour_of_day,\n"
		groupByClause += ", hour_of_day"
	} else {
		selectClause += " NULL as hour_of_day,\n"
	}

	selectClause += `
			COUNT(b.id) as bookings_count,
			SUM(EXTRACT(EPOCH FROM (b.end_time - b.start_time))) / 3600.0 as total_hours_booked
		FROM bookings b
		JOIN game_tables gt ON b.table_id = gt.id
		WHERE (b.status = 'completed' OR b.status = 'active')
	`
	queryBuilder.WriteString(selectClause)

	if params.StartDate != "" {
		queryBuilder.WriteString(" AND b.start_time >= $" + strconv.Itoa(argIdx))
		args = append(args, params.StartDate)
		argIdx++
	}
	if params.EndDate != "" {
		endDateParsed, err := time.Parse(DefaultReportDateLayout, params.EndDate)
		if err == nil {
			endDateAdjusted := endDateParsed.AddDate(0,0,1).Format(DefaultReportDateLayout)
			queryBuilder.WriteString(" AND b.start_time < $" + strconv.Itoa(argIdx))
			args = append(args, endDateAdjusted)
			argIdx++
		} else {
			queryBuilder.WriteString(" AND b.start_time <= $" + strconv.Itoa(argIdx))
			args = append(args, params.EndDate)
			argIdx++
		}
	}
	if params.TableID != nil {
		queryBuilder.WriteString(" AND b.table_id = $" + strconv.Itoa(argIdx))
		args = append(args, *params.TableID)
		argIdx++
	}

	queryBuilder.WriteString(" GROUP BY " + groupByClause)
	queryBuilder.WriteString(" ORDER BY report_date DESC, table_name, hour_of_day ASC")

	rows, err := db.Query(queryBuilder.String(), args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query booking report: " + err.Error(), "query": queryBuilder.String()})
		return
	}
	defer rows.Close()

	reportItems := []models.BookingReportItem{}
	for rows.Next() {
		var item models.BookingReportItem
		var hourOfDay sql.NullInt64
		if err := rows.Scan(
			&item.Date,
			&item.TableID,
			&item.TableName,
			&hourOfDay,
			&item.BookingsCount,
			&item.TotalHours,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan booking report item: " + err.Error()})
			return
		}
		if hourOfDay.Valid {
			hour := int(hourOfDay.Int64)
			item.Hour = &hour
		}
		reportItems = append(reportItems, item)
	}

	c.JSON(http.StatusOK, reportItems)
}

// GetInventoryReports generates inventory reports (e.g., low stock, current stock levels).
func GetInventoryReports(c *gin.Context) {
	// For simplicity, this will list items with stock levels, highlighting low stock.
	// More complex reports could include movement history, spoilage, etc.
	db := database.GetDB()
	query := `
		SELECT 
			pi.id, pi.name, pi.sku, pi.category_id, pc.name as category_name, 
			pi.current_stock, pi.low_stock_threshold,
			(SELECT MAX(im.movement_date) FROM inventory_movements im WHERE im.pricelist_item_id = pi.id) as last_movement_date
		FROM pricelist_items pi
		LEFT JOIN pricelist_categories pc ON pi.category_id = pc.id
		WHERE pi.current_stock IS NOT NULL -- Only items that are tracked
		ORDER BY pi.name ASC
	`

	rows, err := db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query inventory report: " + err.Error()})
		return
	}
	defer rows.Close()

	reportItems := []models.InventoryReportItem{}
	for rows.Next() {
		var item models.InventoryReportItem
		var currentStock sql.NullInt64
		var lowStockThreshold sql.NullInt64
		var lastMovementDate sql.NullTime

		if err := rows.Scan(
			&item.ItemID,
			&item.ItemName,
			&item.SKU,
			&item.CategoryID,
			&item.CategoryName,
			&currentStock,
			&lowStockThreshold,
			&lastMovementDate,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan inventory report item: " + err.Error()})
			return
		}
		if currentStock.Valid {
			item.CurrentStock = int(currentStock.Int64)
		}
		if lowStockThreshold.Valid {
			threshold := int(lowStockThreshold.Int64)
			item.LowStockThreshold = &threshold
			if currentStock.Valid && currentStock.Int64 <= lowStockThreshold.Int64 {
				item.Status = "Low Stock"
			} else if currentStock.Valid && currentStock.Int64 == 0 {
			    item.Status = "Out of Stock"
			} else {
				item.Status = "In Stock"
			}
		} else {
		    if currentStock.Valid && currentStock.Int64 == 0 {
		        item.Status = "Out of Stock"
		    } else {
		        item.Status = "In Stock" // No threshold defined
		    }
		}
		if lastMovementDate.Valid {
			item.LastMovementDate = &lastMovementDate.Time
		}
		reportItems = append(reportItems, item)
	}

	c.JSON(http.StatusOK, reportItems)
}

