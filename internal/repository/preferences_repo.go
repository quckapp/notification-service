package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/quickchat/notification-service/internal/db"
	"github.com/quickchat/notification-service/internal/models"
)

// PreferencesRepository handles notification preferences persistence
type PreferencesRepository struct {
	db *db.MySQL
}

// NewPreferencesRepository creates a new preferences repository
func NewPreferencesRepository(db *db.MySQL) *PreferencesRepository {
	return &PreferencesRepository{db: db}
}

// GetByUserID retrieves preferences for a user
func (r *PreferencesRepository) GetByUserID(ctx context.Context, userID string) (*models.NotificationPreferences, error) {
	query := `SELECT * FROM notification_preferences WHERE user_id = ?`

	var prefs models.NotificationPreferences
	err := r.db.GetContext(ctx, &prefs, query, userID)
	if err == sql.ErrNoRows {
		// Return default preferences if not found
		return models.DefaultPreferences(userID), nil
	}
	if err != nil {
		return nil, err
	}

	return &prefs, nil
}

// Create creates new preferences for a user
func (r *PreferencesRepository) Create(ctx context.Context, prefs *models.NotificationPreferences) error {
	query := `
		INSERT INTO notification_preferences (
			user_id, push_enabled, email_enabled, sms_enabled, in_app_enabled,
			quiet_hours_enabled, quiet_hours_start, quiet_hours_end, timezone,
			category_settings, email_digest_enabled, email_digest_frequency,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	prefs.CreatedAt = now
	prefs.UpdatedAt = now

	result, err := r.db.ExecContext(ctx, query,
		prefs.UserID,
		prefs.PushEnabled,
		prefs.EmailEnabled,
		prefs.SMSEnabled,
		prefs.InAppEnabled,
		prefs.QuietHoursEnabled,
		prefs.QuietHoursStart,
		prefs.QuietHoursEnd,
		prefs.Timezone,
		prefs.CategorySettings,
		prefs.EmailDigestEnabled,
		prefs.EmailDigestFrequency,
		prefs.CreatedAt,
		prefs.UpdatedAt,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	prefs.ID = id

	return nil
}

// Update updates preferences for a user
func (r *PreferencesRepository) Update(ctx context.Context, prefs *models.NotificationPreferences) error {
	query := `
		UPDATE notification_preferences SET
			push_enabled = ?,
			email_enabled = ?,
			sms_enabled = ?,
			in_app_enabled = ?,
			quiet_hours_enabled = ?,
			quiet_hours_start = ?,
			quiet_hours_end = ?,
			timezone = ?,
			category_settings = ?,
			email_digest_enabled = ?,
			email_digest_frequency = ?,
			updated_at = ?
		WHERE user_id = ?
	`

	prefs.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		prefs.PushEnabled,
		prefs.EmailEnabled,
		prefs.SMSEnabled,
		prefs.InAppEnabled,
		prefs.QuietHoursEnabled,
		prefs.QuietHoursStart,
		prefs.QuietHoursEnd,
		prefs.Timezone,
		prefs.CategorySettings,
		prefs.EmailDigestEnabled,
		prefs.EmailDigestFrequency,
		prefs.UpdatedAt,
		prefs.UserID,
	)

	return err
}

// Upsert creates or updates preferences
func (r *PreferencesRepository) Upsert(ctx context.Context, prefs *models.NotificationPreferences) error {
	query := `
		INSERT INTO notification_preferences (
			user_id, push_enabled, email_enabled, sms_enabled, in_app_enabled,
			quiet_hours_enabled, quiet_hours_start, quiet_hours_end, timezone,
			category_settings, email_digest_enabled, email_digest_frequency,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			push_enabled = VALUES(push_enabled),
			email_enabled = VALUES(email_enabled),
			sms_enabled = VALUES(sms_enabled),
			in_app_enabled = VALUES(in_app_enabled),
			quiet_hours_enabled = VALUES(quiet_hours_enabled),
			quiet_hours_start = VALUES(quiet_hours_start),
			quiet_hours_end = VALUES(quiet_hours_end),
			timezone = VALUES(timezone),
			category_settings = VALUES(category_settings),
			email_digest_enabled = VALUES(email_digest_enabled),
			email_digest_frequency = VALUES(email_digest_frequency),
			updated_at = VALUES(updated_at)
	`

	now := time.Now()
	if prefs.CreatedAt.IsZero() {
		prefs.CreatedAt = now
	}
	prefs.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, query,
		prefs.UserID,
		prefs.PushEnabled,
		prefs.EmailEnabled,
		prefs.SMSEnabled,
		prefs.InAppEnabled,
		prefs.QuietHoursEnabled,
		prefs.QuietHoursStart,
		prefs.QuietHoursEnd,
		prefs.Timezone,
		prefs.CategorySettings,
		prefs.EmailDigestEnabled,
		prefs.EmailDigestFrequency,
		prefs.CreatedAt,
		prefs.UpdatedAt,
	)

	return err
}

// Delete deletes preferences for a user
func (r *PreferencesRepository) Delete(ctx context.Context, userID string) error {
	query := `DELETE FROM notification_preferences WHERE user_id = ?`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

// IsChannelEnabled checks if a notification channel is enabled for a category
func (r *PreferencesRepository) IsChannelEnabled(ctx context.Context, userID string, category models.NotificationCategory, channel models.NotificationType) (bool, error) {
	prefs, err := r.GetByUserID(ctx, userID)
	if err != nil {
		return false, err
	}

	// Check global settings first
	switch channel {
	case models.NotificationTypePush:
		if !prefs.PushEnabled {
			return false, nil
		}
	case models.NotificationTypeEmail:
		if !prefs.EmailEnabled {
			return false, nil
		}
	case models.NotificationTypeSMS:
		if !prefs.SMSEnabled {
			return false, nil
		}
	case models.NotificationTypeInApp:
		if !prefs.InAppEnabled {
			return false, nil
		}
	}

	// Check category-specific settings
	cs := prefs.CategorySettings.CategorySettings
	var settings models.ChannelSettings

	switch category {
	case models.CategoryMessage:
		settings = cs.Message
	case models.CategoryCall:
		settings = cs.Call
	case models.CategoryMention:
		settings = cs.Mention
	case models.CategoryReaction:
		settings = cs.Reaction
	case models.CategoryChannel:
		settings = cs.Channel
	case models.CategorySystem:
		settings = cs.System
	case models.CategoryMarketing:
		settings = cs.Marketing
	default:
		return true, nil
	}

	switch channel {
	case models.NotificationTypePush:
		return settings.Push, nil
	case models.NotificationTypeEmail:
		return settings.Email, nil
	case models.NotificationTypeSMS:
		return settings.SMS, nil
	case models.NotificationTypeInApp:
		return settings.InApp, nil
	}

	return true, nil
}
