package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationTypePush    NotificationType = "push"
	NotificationTypeEmail   NotificationType = "email"
	NotificationTypeSMS     NotificationType = "sms"
	NotificationTypeInApp   NotificationType = "in_app"
)

// NotificationStatus represents the delivery status
type NotificationStatus string

const (
	StatusPending   NotificationStatus = "pending"
	StatusSent      NotificationStatus = "sent"
	StatusDelivered NotificationStatus = "delivered"
	StatusFailed    NotificationStatus = "failed"
	StatusRead      NotificationStatus = "read"
)

// NotificationCategory for grouping notifications
type NotificationCategory string

const (
	CategoryMessage      NotificationCategory = "message"
	CategoryCall         NotificationCategory = "call"
	CategoryMention      NotificationCategory = "mention"
	CategoryReaction     NotificationCategory = "reaction"
	CategoryChannel      NotificationCategory = "channel"
	CategorySystem       NotificationCategory = "system"
	CategoryMarketing    NotificationCategory = "marketing"
)

// Priority levels for notifications
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityNormal   Priority = "normal"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// Notification represents a notification to be sent
type Notification struct {
	ID          primitive.ObjectID       `json:"id" bson:"_id,omitempty"`
	UserID      string                   `json:"user_id" bson:"user_id"`
	Type        NotificationType         `json:"type" bson:"type"`
	Category    NotificationCategory     `json:"category" bson:"category"`
	Priority    Priority                 `json:"priority" bson:"priority"`
	Title       string                   `json:"title" bson:"title"`
	Body        string                   `json:"body" bson:"body"`
	Data        map[string]interface{}   `json:"data,omitempty" bson:"data,omitempty"`
	ImageURL    string                   `json:"image_url,omitempty" bson:"image_url,omitempty"`
	ActionURL   string                   `json:"action_url,omitempty" bson:"action_url,omitempty"`
	Status      NotificationStatus       `json:"status" bson:"status"`
	RetryCount  int                      `json:"retry_count" bson:"retry_count"`
	Error       string                   `json:"error,omitempty" bson:"error,omitempty"`
	SentAt      *time.Time               `json:"sent_at,omitempty" bson:"sent_at,omitempty"`
	DeliveredAt *time.Time               `json:"delivered_at,omitempty" bson:"delivered_at,omitempty"`
	ReadAt      *time.Time               `json:"read_at,omitempty" bson:"read_at,omitempty"`
	ExpiresAt   *time.Time               `json:"expires_at,omitempty" bson:"expires_at,omitempty"`
	CreatedAt   time.Time                `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time                `json:"updated_at" bson:"updated_at"`
}

// PushNotification contains push-specific fields
type PushNotification struct {
	Notification
	DeviceTokens []string               `json:"device_tokens" bson:"device_tokens"`
	Platform     string                 `json:"platform" bson:"platform"` // ios, android, web
	Badge        int                    `json:"badge,omitempty" bson:"badge,omitempty"`
	Sound        string                 `json:"sound,omitempty" bson:"sound,omitempty"`
	ChannelID    string                 `json:"channel_id,omitempty" bson:"channel_id,omitempty"` // Android
	ThreadID     string                 `json:"thread_id,omitempty" bson:"thread_id,omitempty"`   // iOS
	CollapseKey  string                 `json:"collapse_key,omitempty" bson:"collapse_key,omitempty"`
}

// EmailNotification contains email-specific fields
type EmailNotification struct {
	Notification
	ToEmail     string                 `json:"to_email" bson:"to_email"`
	ToName      string                 `json:"to_name,omitempty" bson:"to_name,omitempty"`
	Subject     string                 `json:"subject" bson:"subject"`
	HTMLBody    string                 `json:"html_body,omitempty" bson:"html_body,omitempty"`
	TextBody    string                 `json:"text_body,omitempty" bson:"text_body,omitempty"`
	TemplateID  string                 `json:"template_id,omitempty" bson:"template_id,omitempty"`
	TemplateData map[string]interface{} `json:"template_data,omitempty" bson:"template_data,omitempty"`
	Attachments []EmailAttachment      `json:"attachments,omitempty" bson:"attachments,omitempty"`
	ReplyTo     string                 `json:"reply_to,omitempty" bson:"reply_to,omitempty"`
}

// EmailAttachment represents an email attachment
type EmailAttachment struct {
	Filename    string `json:"filename" bson:"filename"`
	ContentType string `json:"content_type" bson:"content_type"`
	Content     []byte `json:"content" bson:"content"`
}

// SMSNotification contains SMS-specific fields
type SMSNotification struct {
	Notification
	PhoneNumber string `json:"phone_number" bson:"phone_number"`
	MessageSID  string `json:"message_sid,omitempty" bson:"message_sid,omitempty"`
}

// InAppNotification contains in-app notification fields
type InAppNotification struct {
	Notification
	Persistent  bool   `json:"persistent" bson:"persistent"`
	DismissedAt *time.Time `json:"dismissed_at,omitempty" bson:"dismissed_at,omitempty"`
}

// NotificationRequest is the API request for sending notifications
type NotificationRequest struct {
	UserID    string                 `json:"user_id" binding:"required"`
	Type      NotificationType       `json:"type" binding:"required"`
	Category  NotificationCategory   `json:"category" binding:"required"`
	Priority  Priority               `json:"priority"`
	Title     string                 `json:"title" binding:"required"`
	Body      string                 `json:"body" binding:"required"`
	Data      map[string]interface{} `json:"data,omitempty"`
	ImageURL  string                 `json:"image_url,omitempty"`
	ActionURL string                 `json:"action_url,omitempty"`
	ExpiresIn int                    `json:"expires_in,omitempty"` // seconds

	// Push-specific
	DeviceTokens []string `json:"device_tokens,omitempty"`
	Platform     string   `json:"platform,omitempty"`
	Badge        int      `json:"badge,omitempty"`
	Sound        string   `json:"sound,omitempty"`

	// Email-specific
	ToEmail      string                 `json:"to_email,omitempty"`
	ToName       string                 `json:"to_name,omitempty"`
	Subject      string                 `json:"subject,omitempty"`
	HTMLBody     string                 `json:"html_body,omitempty"`
	TemplateID   string                 `json:"template_id,omitempty"`
	TemplateData map[string]interface{} `json:"template_data,omitempty"`

	// SMS-specific
	PhoneNumber string `json:"phone_number,omitempty"`
}

// BulkNotificationRequest for sending to multiple users
type BulkNotificationRequest struct {
	UserIDs   []string               `json:"user_ids" binding:"required"`
	Type      NotificationType       `json:"type" binding:"required"`
	Category  NotificationCategory   `json:"category" binding:"required"`
	Priority  Priority               `json:"priority"`
	Title     string                 `json:"title" binding:"required"`
	Body      string                 `json:"body" binding:"required"`
	Data      map[string]interface{} `json:"data,omitempty"`
	ImageURL  string                 `json:"image_url,omitempty"`
	ActionURL string                 `json:"action_url,omitempty"`
}
