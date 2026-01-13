package providers

import (
	"context"
	"fmt"

	"github.com/quickchat/notification-service/internal/config"
	"github.com/quickchat/notification-service/internal/models"
	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/payload"
	"github.com/sideshow/apns2/token"
	"github.com/sirupsen/logrus"
)

// APNSProvider implements push notifications via Apple Push Notification Service
type APNSProvider struct {
	client   *apns2.Client
	config   config.APNSConfig
	log      *logrus.Logger
	bundleID string
}

// NewAPNSProvider creates a new APNS provider
func NewAPNSProvider(cfg config.APNSConfig, log *logrus.Logger) (*APNSProvider, error) {
	// Load auth key for token-based authentication
	authKey, err := token.AuthKeyFromFile(cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load APNS auth key: %w", err)
	}

	// Create token
	apnsToken := &token.Token{
		AuthKey: authKey,
		KeyID:   cfg.KeyID,
		TeamID:  cfg.TeamID,
	}

	// Create client
	var client *apns2.Client
	if cfg.UseSandbox {
		client = apns2.NewTokenClient(apnsToken).Development()
	} else {
		client = apns2.NewTokenClient(apnsToken).Production()
	}

	return &APNSProvider{
		client:   client,
		config:   cfg,
		log:      log,
		bundleID: cfg.BundleID,
	}, nil
}

// Name returns the provider name
func (p *APNSProvider) Name() string {
	return "apns"
}

// Send sends a notification
func (p *APNSProvider) Send(ctx context.Context, notification *models.Notification) error {
	push := &models.PushNotification{Notification: *notification}
	return p.SendToToken(ctx, "", push)
}

// SendBatch sends multiple notifications
func (p *APNSProvider) SendBatch(ctx context.Context, notifications []*models.Notification) ([]SendResult, error) {
	results := make([]SendResult, len(notifications))
	for i, n := range notifications {
		err := p.Send(ctx, n)
		results[i] = SendResult{
			NotificationID: n.ID.Hex(),
			Success:        err == nil,
			Error:          err,
		}
	}
	return results, nil
}

// SendToToken sends a push notification to a single token
func (p *APNSProvider) SendToToken(ctx context.Context, deviceToken string, notification *models.PushNotification) error {
	apnsPayload := p.buildPayload(notification)

	apnsNotification := &apns2.Notification{
		DeviceToken: deviceToken,
		Topic:       p.bundleID,
		Payload:     apnsPayload,
		Priority:    apns2.PriorityHigh,
	}

	// Set collapse ID if provided
	if notification.CollapseKey != "" {
		apnsNotification.CollapseID = notification.CollapseKey
	}

	// Set thread ID for grouping
	if notification.ThreadID != "" {
		apnsNotification.Payload = apnsPayload.ThreadID(notification.ThreadID)
	}

	// Push
	response, err := p.client.PushWithContext(ctx, apnsNotification)
	if err != nil {
		p.log.Errorf("APNS send failed: %v", err)
		return fmt.Errorf("APNS send failed: %w", err)
	}

	if !response.Sent() {
		p.log.Errorf("APNS notification rejected: %s - %s", response.StatusCode, response.Reason)
		return fmt.Errorf("APNS rejected: %s", response.Reason)
	}

	p.log.Debugf("APNS notification sent: %s", response.ApnsID)
	return nil
}

// SendToTokens sends a push notification to multiple tokens
func (p *APNSProvider) SendToTokens(ctx context.Context, tokens []string, notification *models.PushNotification) ([]SendResult, error) {
	results := make([]SendResult, len(tokens))

	for i, token := range tokens {
		err := p.SendToToken(ctx, token, notification)
		results[i] = SendResult{
			NotificationID: notification.ID.Hex(),
			Success:        err == nil,
			Error:          err,
		}
	}

	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
	}

	p.log.Infof("APNS batch sent: %d success, %d failure", successCount, len(tokens)-successCount)
	return results, nil
}

// buildPayload builds an APNS payload from a notification
func (p *APNSProvider) buildPayload(notification *models.PushNotification) *payload.Payload {
	apnsPayload := payload.NewPayload().
		AlertTitle(notification.Title).
		AlertBody(notification.Body).
		MutableContent().
		Category(string(notification.Category))

	// Set badge
	if notification.Badge > 0 {
		apnsPayload = apnsPayload.Badge(notification.Badge)
	}

	// Set sound
	if notification.Sound != "" {
		apnsPayload = apnsPayload.Sound(notification.Sound)
	} else {
		apnsPayload = apnsPayload.Sound("default")
	}

	// Add custom data
	for k, v := range notification.Data {
		apnsPayload = apnsPayload.Custom(k, v)
	}

	// Add action URL
	if notification.ActionURL != "" {
		apnsPayload = apnsPayload.Custom("action_url", notification.ActionURL)
	}

	// Add image URL
	if notification.ImageURL != "" {
		apnsPayload = apnsPayload.Custom("image_url", notification.ImageURL)
	}

	return apnsPayload
}

// IsHealthy checks if APNS is available
func (p *APNSProvider) IsHealthy(ctx context.Context) bool {
	return p.client != nil
}
