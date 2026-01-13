package providers

import (
	"context"
	"fmt"
	"sync"

	"github.com/quickchat/notification-service/internal/config"
	"github.com/quickchat/notification-service/internal/models"
	"github.com/sirupsen/logrus"
)

// Manager manages all notification providers
type Manager struct {
	config    *config.Config
	log       *logrus.Logger
	mu        sync.RWMutex

	fcmProvider    PushProvider
	apnsProvider   PushProvider
	emailProvider  EmailProvider
	smsProvider    SMSProvider
}

// NewManager creates a new provider manager
func NewManager(cfg *config.Config, log *logrus.Logger) *Manager {
	return &Manager{
		config: cfg,
		log:    log,
	}
}

// Initialize initializes all configured providers
func (m *Manager) Initialize() error {
	var errs []error

	// Initialize FCM (Firebase Cloud Messaging) for Android/Web
	if m.config.Firebase.CredentialsFile != "" {
		fcm, err := NewFCMProvider(m.config.Firebase, m.log)
		if err != nil {
			m.log.Warnf("Failed to initialize FCM provider: %v", err)
			errs = append(errs, err)
		} else {
			m.fcmProvider = fcm
			m.log.Info("FCM provider initialized")
		}
	}

	// Initialize APNS for iOS
	if m.config.APNS.KeyFile != "" {
		apns, err := NewAPNSProvider(m.config.APNS, m.log)
		if err != nil {
			m.log.Warnf("Failed to initialize APNS provider: %v", err)
			errs = append(errs, err)
		} else {
			m.apnsProvider = apns
			m.log.Info("APNS provider initialized")
		}
	}

	// Initialize Email provider
	if m.config.Email.APIKey != "" || m.config.Email.SMTPHost != "" {
		email, err := NewEmailProvider(m.config.Email, m.log)
		if err != nil {
			m.log.Warnf("Failed to initialize Email provider: %v", err)
			errs = append(errs, err)
		} else {
			m.emailProvider = email
			m.log.Info("Email provider initialized")
		}
	}

	// Initialize SMS provider
	if m.config.SMS.AccountSID != "" {
		sms, err := NewSMSProvider(m.config.SMS, m.log)
		if err != nil {
			m.log.Warnf("Failed to initialize SMS provider: %v", err)
			errs = append(errs, err)
		} else {
			m.smsProvider = sms
			m.log.Info("SMS provider initialized")
		}
	}

	// At least one provider should be available
	if m.fcmProvider == nil && m.apnsProvider == nil && m.emailProvider == nil && m.smsProvider == nil {
		return fmt.Errorf("no notification providers initialized")
	}

	return nil
}

// SendPushNotification sends a push notification
func (m *Manager) SendPushNotification(ctx context.Context, notification *models.PushNotification) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	switch notification.Platform {
	case "ios":
		if m.apnsProvider == nil {
			return fmt.Errorf("APNS provider not available")
		}
		return m.apnsProvider.SendToTokens(ctx, notification.DeviceTokens, notification)[0].Error

	case "android", "web":
		if m.fcmProvider == nil {
			return fmt.Errorf("FCM provider not available")
		}
		_, err := m.fcmProvider.SendToTokens(ctx, notification.DeviceTokens, notification)
		if len(err) > 0 && err[0].Error != nil {
			return err[0].Error
		}
		return nil

	default:
		return fmt.Errorf("unknown platform: %s", notification.Platform)
	}
}

// SendEmailNotification sends an email notification
func (m *Manager) SendEmailNotification(ctx context.Context, notification *models.EmailNotification) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.emailProvider == nil {
		return fmt.Errorf("email provider not available")
	}

	return m.emailProvider.SendEmail(ctx, notification)
}

// SendSMSNotification sends an SMS notification
func (m *Manager) SendSMSNotification(ctx context.Context, notification *models.SMSNotification) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.smsProvider == nil {
		return fmt.Errorf("SMS provider not available")
	}

	return m.smsProvider.SendSMS(ctx, notification)
}

// GetPushProvider returns the appropriate push provider for a platform
func (m *Manager) GetPushProvider(platform string) PushProvider {
	m.mu.RLock()
	defer m.mu.RUnlock()

	switch platform {
	case "ios":
		return m.apnsProvider
	case "android", "web":
		return m.fcmProvider
	default:
		return m.fcmProvider // Default to FCM
	}
}

// GetEmailProvider returns the email provider
func (m *Manager) GetEmailProvider() EmailProvider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.emailProvider
}

// GetSMSProvider returns the SMS provider
func (m *Manager) GetSMSProvider() SMSProvider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.smsProvider
}

// HealthCheck checks all provider health
func (m *Manager) HealthCheck(ctx context.Context) map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	health := make(map[string]bool)

	if m.fcmProvider != nil {
		health["fcm"] = m.fcmProvider.IsHealthy(ctx)
	}
	if m.apnsProvider != nil {
		health["apns"] = m.apnsProvider.IsHealthy(ctx)
	}
	if m.emailProvider != nil {
		health["email"] = m.emailProvider.IsHealthy(ctx)
	}
	if m.smsProvider != nil {
		health["sms"] = m.smsProvider.IsHealthy(ctx)
	}

	return health
}
