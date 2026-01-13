package providers

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/quickchat/notification-service/internal/config"
	"github.com/quickchat/notification-service/internal/models"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
)

// FCMProvider implements push notifications via Firebase Cloud Messaging
type FCMProvider struct {
	client *messaging.Client
	config config.FirebaseConfig
	log    *logrus.Logger
}

// NewFCMProvider creates a new FCM provider
func NewFCMProvider(cfg config.FirebaseConfig, log *logrus.Logger) (*FCMProvider, error) {
	ctx := context.Background()

	opt := option.WithCredentialsFile(cfg.CredentialsFile)
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Firebase app: %w", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get messaging client: %w", err)
	}

	return &FCMProvider{
		client: client,
		config: cfg,
		log:    log,
	}, nil
}

// Name returns the provider name
func (p *FCMProvider) Name() string {
	return "fcm"
}

// Send sends a notification
func (p *FCMProvider) Send(ctx context.Context, notification *models.Notification) error {
	push := &models.PushNotification{Notification: *notification}
	return p.SendToToken(ctx, "", push)
}

// SendBatch sends multiple notifications
func (p *FCMProvider) SendBatch(ctx context.Context, notifications []*models.Notification) ([]SendResult, error) {
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
func (p *FCMProvider) SendToToken(ctx context.Context, token string, notification *models.PushNotification) error {
	message := p.buildMessage(notification, token)

	_, err := p.client.Send(ctx, message)
	if err != nil {
		p.log.Errorf("FCM send failed: %v", err)
		return fmt.Errorf("FCM send failed: %w", err)
	}

	p.log.Debugf("FCM notification sent to token: %s", token[:20]+"...")
	return nil
}

// SendToTokens sends a push notification to multiple tokens
func (p *FCMProvider) SendToTokens(ctx context.Context, tokens []string, notification *models.PushNotification) ([]SendResult, error) {
	if len(tokens) == 0 {
		return nil, fmt.Errorf("no tokens provided")
	}

	messages := make([]*messaging.Message, len(tokens))
	for i, token := range tokens {
		messages[i] = p.buildMessage(notification, token)
	}

	// Use SendEach for multiple messages (SendAll is deprecated)
	response, err := p.client.SendEach(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("FCM batch send failed: %w", err)
	}

	results := make([]SendResult, len(tokens))
	for i, resp := range response.Responses {
		results[i] = SendResult{
			NotificationID: notification.ID.Hex(),
			Success:        resp.Success,
		}
		if resp.Error != nil {
			results[i].Error = resp.Error
		}
		if resp.MessageID != "" {
			results[i].ProviderID = resp.MessageID
		}
	}

	p.log.Infof("FCM batch sent: %d success, %d failure", response.SuccessCount, response.FailureCount)
	return results, nil
}

// buildMessage builds an FCM message from a notification
func (p *FCMProvider) buildMessage(notification *models.PushNotification, token string) *messaging.Message {
	// Build notification payload
	fcmNotification := &messaging.Notification{
		Title:    notification.Title,
		Body:     notification.Body,
		ImageURL: notification.ImageURL,
	}

	// Build Android config
	androidConfig := &messaging.AndroidConfig{
		Priority: "high",
		Notification: &messaging.AndroidNotification{
			ChannelID:    notification.ChannelID,
			Sound:        notification.Sound,
			ClickAction:  notification.ActionURL,
			DefaultSound: notification.Sound == "" || notification.Sound == "default",
		},
	}

	if notification.CollapseKey != "" {
		androidConfig.CollapseKey = notification.CollapseKey
	}

	// Build web push config
	webpushConfig := &messaging.WebpushConfig{
		Notification: &messaging.WebpushNotification{
			Title: notification.Title,
			Body:  notification.Body,
			Icon:  notification.ImageURL,
		},
	}

	// Build data payload
	data := make(map[string]string)
	for k, v := range notification.Data {
		data[k] = fmt.Sprintf("%v", v)
	}
	data["category"] = string(notification.Category)
	data["priority"] = string(notification.Priority)

	return &messaging.Message{
		Token:        token,
		Notification: fcmNotification,
		Data:         data,
		Android:      androidConfig,
		Webpush:      webpushConfig,
	}
}

// IsHealthy checks if FCM is available
func (p *FCMProvider) IsHealthy(ctx context.Context) bool {
	// FCM doesn't have a direct health check endpoint
	// We consider it healthy if client is initialized
	return p.client != nil
}
