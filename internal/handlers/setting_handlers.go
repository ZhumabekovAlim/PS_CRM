package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"ps_club_backend/internal/database"
	"ps_club_backend/internal/models"

	"github.com/gin-gonic/gin"
)

// GetApplicationSettings retrieves all application settings
func GetApplicationSettings(c *gin.Context) {
	db := database.GetDB()
	rows, err := db.Query("SELECT id, setting_key, setting_value, description, created_at, updated_at FROM application_settings ORDER BY setting_key")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch application settings: " + err.Error()})
		return
	}
	defer rows.Close()

	settings := []models.ApplicationSetting{}
	for rows.Next() {
		var s models.ApplicationSetting
		if err := rows.Scan(&s.ID, &s.SettingKey, &s.SettingValue, &s.Description, &s.CreatedAt, &s.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan application setting: " + err.Error()})
			return
		}
		settings = append(settings, s)
	}
	c.JSON(http.StatusOK, settings)
}

// GetApplicationSettingByKey retrieves a specific application setting by its key
func GetApplicationSettingByKey(c *gin.Context) {
	key := c.Param("key")
	db := database.GetDB()
	var s models.ApplicationSetting
	query := "SELECT id, setting_key, setting_value, description, created_at, updated_at FROM application_settings WHERE setting_key = $1"
	err := db.QueryRow(query, key).Scan(&s.ID, &s.SettingKey, &s.SettingValue, &s.Description, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Application setting not found for key: " + key})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch application setting: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, s)
}

// CreateOrUpdateApplicationSetting creates a new setting or updates an existing one by key
func CreateOrUpdateApplicationSetting(c *gin.Context) {
	var setting models.ApplicationSetting
	if err := c.ShouldBindJSON(&setting); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	if setting.SettingKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Setting key cannot be empty"})
		return
	}

	db := database.GetDB()
	now := time.Now()

	// Try to update first (UPSERT behavior)
	query := `
	    INSERT INTO application_settings (setting_key, setting_value, description, created_at, updated_at) 
	    VALUES ($1, $2, $3, $4, $5) 
	    ON CONFLICT (setting_key) 
	    DO UPDATE SET setting_value = EXCLUDED.setting_value, description = EXCLUDED.description, updated_at = EXCLUDED.updated_at
	    RETURNING id, setting_key, setting_value, description, created_at, updated_at`

	err := db.QueryRow(query, setting.SettingKey, setting.SettingValue, setting.Description, now, now).
		Scan(&setting.ID, &setting.SettingKey, &setting.SettingValue, &setting.Description, &setting.CreatedAt, &setting.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create or update application setting: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, setting) // Could be StatusCreated if we distinguish, but OK is fine for upsert.
}

// DeleteApplicationSettingByKey deletes an application setting by its key
func DeleteApplicationSettingByKey(c *gin.Context) {
	key := c.Param("key")
	db := database.GetDB()

	result, err := db.Exec("DELETE FROM application_settings WHERE setting_key = $1", key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete application setting: " + err.Error()})
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Application setting not found to delete for key: " + key})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Application setting '" + key + "' deleted successfully"})
}

