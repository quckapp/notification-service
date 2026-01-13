package providers

import (
	"context"

	"github.com/quickchat/notification-service/internal/models"
)

// Provider interface for all notification providers
type Provider interface {
	// Name returns the provider name
	Name() string

	// Send sends a notification
	Send(ctx context.Context, notification *models.Notification) error

	// SendBatch sends multiple notifications
	SendBatch(ctx context.Context, notifications []*models.Notification) ([]SendResult, error)

	// IsHealthy checks if the provider is available
	IsHealthy(ctx context.Context) bool
}

// SendResult represents the result of a single send operation
type SendResult struct {
	NotificationID string
	Success        bool
	Error          error
	ProviderID     string // External ID from provider (e.g., message SID)
}

// PushProvider interface for push notification providers
type PushProvider interface {
	Provider
	SendToToken(ctx context.Context, token string, notification *models.PushNotification) error
	SendToTokens(ctx context.Context, tokens []string, notification *models.PushNotification) ([]SendResult, error)
}

// EmailProvider interface for email providers
type EmailProvider interface {
	Provider
	SendEmail(ctx context.Context, email *models.EmailNotification) error
}

// SMSProvider interface for SMS providers
type SMSProvider interface {
	Provider
	SendSMS(ctx context.Context, sms *models.SMSNotification) error
}
