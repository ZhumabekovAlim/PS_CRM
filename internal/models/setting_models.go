package models

import "time"

// ApplicationSetting represents a key-value pair for application configuration
type ApplicationSetting struct {
	ID            int64     `json:"id" db:"id"`
	SettingKey    string    `json:"setting_key" db:"setting_key" binding:"required"`
	SettingValue  *string   `json:"setting_value,omitempty" db:"setting_value"`
	Description   *string   `json:"description,omitempty" db:"description"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

