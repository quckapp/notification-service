package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// NotificationPreferences represents user notification settings
type NotificationPreferences struct {
	ID        int64     `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`

	// Global settings
	PushEnabled   bool `json:"push_enabled" db:"push_enabled"`
	EmailEnabled  bool `json:"email_enabled" db:"email_enabled"`
	SMSEnabled    bool `json:"sms_enabled" db:"sms_enabled"`
	InAppEnabled  bool `json:"in_app_enabled" db:"in_app_enabled"`

	// Quiet hours
	QuietHoursEnabled bool   `json:"quiet_hours_enabled" db:"quiet_hours_enabled"`
	QuietHoursStart   string `json:"quiet_hours_start" db:"quiet_hours_start"` // HH:MM format
	QuietHoursEnd     string `json:"quiet_hours_end" db:"quiet_hours_end"`     // HH:MM format
	Timezone          string `json:"timezone" db:"timezone"`

	// Category-specific settings (stored as JSON)
	CategorySettings CategorySettingsJSON `json:"category_settings" db:"category_settings"`

	// Email preferences
	EmailDigestEnabled   bool   `json:"email_digest_enabled" db:"email_digest_enabled"`
	EmailDigestFrequency string `json:"email_digest_frequency" db:"email_digest_frequency"` // daily, weekly

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// CategorySettings for per-category notification preferences
type CategorySettings struct {
	Message   ChannelSettings `json:"message"`
	Call      ChannelSettings `json:"call"`
	Mention   ChannelSettings `json:"mention"`
	Reaction  ChannelSettings `json:"reaction"`
	Channel   ChannelSettings `json:"channel"`
	System    ChannelSettings `json:"system"`
	Marketing ChannelSettings `json:"marketing"`
}

// ChannelSettings for each notification channel
type ChannelSettings struct {
	Push   bool `json:"push"`
	Email  bool `json:"email"`
	SMS    bool `json:"sms"`
	InApp  bool `json:"in_app"`
}

// CategorySettingsJSON wraps CategorySettings for database storage
type CategorySettingsJSON struct {
	CategorySettings
}

// Value implements driver.Valuer for database storage
func (c CategorySettingsJSON) Value() (driver.Value, error) {
	return json.Marshal(c.CategorySettings)
}

// Scan implements sql.Scanner for database retrieval
func (c *CategorySettingsJSON) Scan(value interface{}) error {
	if value == nil {
		c.CategorySettings = DefaultCategorySettings()
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(bytes, &c.CategorySettings)
}

// DefaultCategorySettings returns default notification settings
func DefaultCategorySettings() CategorySettings {
	return CategorySettings{
		Message:   ChannelSettings{Push: true, Email: false, SMS: false, InApp: true},
		Call:      ChannelSettings{Push: true, Email: false, SMS: false, InApp: true},
		Mention:   ChannelSettings{Push: true, Email: true, SMS: false, InApp: true},
		Reaction:  ChannelSettings{Push: true, Email: false, SMS: false, InApp: true},
		Channel:   ChannelSettings{Push: true, Email: false, SMS: false, InApp: true},
		System:    ChannelSettings{Push: true, Email: true, SMS: false, InApp: true},
		Marketing: ChannelSettings{Push: false, Email: true, SMS: false, InApp: false},
	}
}

// DefaultPreferences creates default notification preferences for a user
func DefaultPreferences(userID string) *NotificationPreferences {
	return &NotificationPreferences{
		UserID:            userID,
		PushEnabled:       true,
		EmailEnabled:      true,
		SMSEnabled:        false,
		InAppEnabled:      true,
		QuietHoursEnabled: false,
		QuietHoursStart:   "22:00",
		QuietHoursEnd:     "08:00",
		Timezone:          "UTC",
		CategorySettings:  CategorySettingsJSON{DefaultCategorySettings()},
		EmailDigestEnabled:   false,
		EmailDigestFrequency: "daily",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

// PreferencesUpdateRequest for updating user preferences
type PreferencesUpdateRequest struct {
	PushEnabled          *bool             `json:"push_enabled,omitempty"`
	EmailEnabled         *bool             `json:"email_enabled,omitempty"`
	SMSEnabled           *bool             `json:"sms_enabled,omitempty"`
	InAppEnabled         *bool             `json:"in_app_enabled,omitempty"`
	QuietHoursEnabled    *bool             `json:"quiet_hours_enabled,omitempty"`
	QuietHoursStart      string            `json:"quiet_hours_start,omitempty"`
	QuietHoursEnd        string            `json:"quiet_hours_end,omitempty"`
	Timezone             string            `json:"timezone,omitempty"`
	CategorySettings     *CategorySettings `json:"category_settings,omitempty"`
	EmailDigestEnabled   *bool             `json:"email_digest_enabled,omitempty"`
	EmailDigestFrequency string            `json:"email_digest_frequency,omitempty"`
}
