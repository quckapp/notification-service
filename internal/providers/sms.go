package providers

import (
	"context"
	"fmt"

	"github.com/quickchat/notification-service/internal/config"
	"github.com/quickchat/notification-service/internal/models"
	"github.com/sirupsen/logrus"
	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
)

// SMSProvider implements SMS notifications via Twilio
type SMSProvider struct {
	client *twilio.RestClient
	config config.SMSConfig
	log    *logrus.Logger
}

// NewSMSProvider creates a new SMS provider
func NewSMSProvider(cfg config.SMSConfig, log *logrus.Logger) (*SMSProvider, error) {
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: cfg.AccountSID,
		Password: cfg.AuthToken,
	})

	return &SMSProvider{
		client: client,
		config: cfg,
		log:    log,
	}, nil
}

// Name returns the provider name
func (p *SMSProvider) Name() string {
	return "sms"
}

// Send sends a notification
func (p *SMSProvider) Send(ctx context.Context, notification *models.Notification) error {
	sms := &models.SMSNotification{
		Notification: *notification,
	}
	return p.SendSMS(ctx, sms)
}

// SendBatch sends multiple notifications
func (p *SMSProvider) SendBatch(ctx context.Context, notifications []*models.Notification) ([]SendResult, error) {
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

// SendSMS sends an SMS notification
func (p *SMSProvider) SendSMS(ctx context.Context, sms *models.SMSNotification) error {
	if sms.PhoneNumber == "" {
		return fmt.Errorf("phone number is required")
	}

	// Build message body
	body := sms.Body
	if body == "" {
		body = fmt.Sprintf("%s: %s", sms.Title, sms.Body)
	}

	// Truncate if too long (SMS limit is 160 chars for single message)
	if len(body) > 160 {
		body = body[:157] + "..."
	}

	params := &twilioApi.CreateMessageParams{}
	params.SetTo(sms.PhoneNumber)
	params.SetFrom(p.config.FromNumber)
	params.SetBody(body)

	resp, err := p.client.Api.CreateMessage(params)
	if err != nil {
		p.log.Errorf("Twilio send failed: %v", err)
		return fmt.Errorf("Twilio send failed: %w", err)
	}

	if resp.Sid != nil {
		sms.MessageSID = *resp.Sid
		p.log.Debugf("SMS sent via Twilio, SID: %s", *resp.Sid)
	}

	return nil
}

// IsHealthy checks if SMS provider is available
func (p *SMSProvider) IsHealthy(ctx context.Context) bool {
	return p.client != nil && p.config.FromNumber != ""
}
