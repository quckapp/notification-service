package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/quickchat/notification-service/internal/db"
	"github.com/quickchat/notification-service/internal/models"
)

// DeviceRepository handles device persistence in MySQL
type DeviceRepository struct {
	db *db.MySQL
}

// NewDeviceRepository creates a new device repository
func NewDeviceRepository(db *db.MySQL) *DeviceRepository {
	return &DeviceRepository{db: db}
}

// Create registers a new device
func (r *DeviceRepository) Create(ctx context.Context, device *models.Device) error {
	query := `
		INSERT INTO devices (user_id, device_token, platform, device_model, os_version, app_version, push_enabled, last_active_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			device_model = VALUES(device_model),
			os_version = VALUES(os_version),
			app_version = VALUES(app_version),
			push_enabled = VALUES(push_enabled),
			last_active_at = VALUES(last_active_at),
			updated_at = VALUES(updated_at)
	`

	now := time.Now()
	device.CreatedAt = now
	device.UpdatedAt = now
	device.LastActiveAt = now

	result, err := r.db.ExecContext(ctx, query,
		device.UserID,
		device.DeviceToken,
		device.Platform,
		device.DeviceModel,
		device.OSVersion,
		device.AppVersion,
		device.PushEnabled,
		device.LastActiveAt,
		device.CreatedAt,
		device.UpdatedAt,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	device.ID = id

	return nil
}

// GetByToken retrieves a device by token
func (r *DeviceRepository) GetByToken(ctx context.Context, token string) (*models.Device, error) {
	query := `SELECT * FROM devices WHERE device_token = ?`

	var device models.Device
	err := r.db.GetContext(ctx, &device, query, token)
	if err != nil {
		return nil, err
	}

	return &device, nil
}

// GetByUserID retrieves all devices for a user
func (r *DeviceRepository) GetByUserID(ctx context.Context, userID string) ([]*models.Device, error) {
	query := `SELECT * FROM devices WHERE user_id = ? AND push_enabled = true ORDER BY last_active_at DESC`

	var devices []*models.Device
	err := r.db.SelectContext(ctx, &devices, query, userID)
	if err != nil {
		return nil, err
	}

	return devices, nil
}

// GetTokensByUserID retrieves device tokens for a user by platform
func (r *DeviceRepository) GetTokensByUserID(ctx context.Context, userID string, platform string) ([]string, error) {
	query := `SELECT device_token FROM devices WHERE user_id = ? AND push_enabled = true`
	args := []interface{}{userID}

	if platform != "" {
		query += ` AND platform = ?`
		args = append(args, platform)
	}

	var tokens []string
	err := r.db.SelectContext(ctx, &tokens, query, args...)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

// GetTokensByUserIDs retrieves device tokens for multiple users
func (r *DeviceRepository) GetTokensByUserIDs(ctx context.Context, userIDs []string) (map[string][]string, error) {
	if len(userIDs) == 0 {
		return make(map[string][]string), nil
	}

	query, args, err := sqlx.In(
		`SELECT user_id, device_token FROM devices WHERE user_id IN (?) AND push_enabled = true`,
		userIDs,
	)
	if err != nil {
		return nil, err
	}

	query = r.db.Rebind(query)

	type result struct {
		UserID      string `db:"user_id"`
		DeviceToken string `db:"device_token"`
	}

	var results []result
	if err := r.db.SelectContext(ctx, &results, query, args...); err != nil {
		return nil, err
	}

	tokens := make(map[string][]string)
	for _, r := range results {
		tokens[r.UserID] = append(tokens[r.UserID], r.DeviceToken)
	}

	return tokens, nil
}

// UpdateLastActive updates the last active timestamp
func (r *DeviceRepository) UpdateLastActive(ctx context.Context, token string) error {
	query := `UPDATE devices SET last_active_at = ?, updated_at = ? WHERE device_token = ?`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, now, now, token)
	return err
}

// UpdatePushEnabled enables/disables push for a device
func (r *DeviceRepository) UpdatePushEnabled(ctx context.Context, token string, enabled bool) error {
	query := `UPDATE devices SET push_enabled = ?, updated_at = ? WHERE device_token = ?`
	_, err := r.db.ExecContext(ctx, query, enabled, time.Now(), token)
	return err
}

// Delete removes a device
func (r *DeviceRepository) Delete(ctx context.Context, token string) error {
	query := `DELETE FROM devices WHERE device_token = ?`
	_, err := r.db.ExecContext(ctx, query, token)
	return err
}

// DeleteByUserID removes all devices for a user
func (r *DeviceRepository) DeleteByUserID(ctx context.Context, userID string) error {
	query := `DELETE FROM devices WHERE user_id = ?`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

// DeleteInactive removes devices inactive for specified duration
func (r *DeviceRepository) DeleteInactive(ctx context.Context, duration time.Duration) (int64, error) {
	cutoff := time.Now().Add(-duration)
	query := `DELETE FROM devices WHERE last_active_at < ?`

	result, err := r.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// sqlx for IN queries
import "github.com/jmoiron/sqlx"
