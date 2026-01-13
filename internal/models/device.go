package models

import (
	"time"
)

// DevicePlatform represents the device platform
type DevicePlatform string

const (
	PlatformIOS     DevicePlatform = "ios"
	PlatformAndroid DevicePlatform = "android"
	PlatformWeb     DevicePlatform = "web"
)

// Device represents a user's registered device for push notifications
type Device struct {
	ID           int64          `json:"id" db:"id"`
	UserID       string         `json:"user_id" db:"user_id"`
	DeviceToken  string         `json:"device_token" db:"device_token"`
	Platform     DevicePlatform `json:"platform" db:"platform"`
	DeviceModel  string         `json:"device_model,omitempty" db:"device_model"`
	OSVersion    string         `json:"os_version,omitempty" db:"os_version"`
	AppVersion   string         `json:"app_version,omitempty" db:"app_version"`
	PushEnabled  bool           `json:"push_enabled" db:"push_enabled"`
	LastActiveAt time.Time      `json:"last_active_at" db:"last_active_at"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at" db:"updated_at"`
}

// DeviceRegistrationRequest for registering a new device
type DeviceRegistrationRequest struct {
	UserID      string         `json:"user_id" binding:"required"`
	DeviceToken string         `json:"device_token" binding:"required"`
	Platform    DevicePlatform `json:"platform" binding:"required"`
	DeviceModel string         `json:"device_model,omitempty"`
	OSVersion   string         `json:"os_version,omitempty"`
	AppVersion  string         `json:"app_version,omitempty"`
}
