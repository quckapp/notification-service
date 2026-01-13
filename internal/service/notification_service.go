package service

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/quickchat/notification-service/internal/db"
	"github.com/quickchat/notification-service/internal/models"
	"github.com/quickchat/notification-service/internal/providers"
	"github.com/quickchat/notification-service/internal/repository"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// NotificationService handles notification business logic
type NotificationService struct {
	notificationRepo *repository.NotificationRepository
	deviceRepo       *repository.DeviceRepository
	prefsRepo        *repository.PreferencesRepository
	providers        *providers.Manager
	redis            *db.Redis
	log              *logrus.Logger
}

// NewNotificationService creates a new notification service
func NewNotificationService(
	mysql *db.MySQL,
	mongo *db.MongoDB,
	redis *db.Redis,
	providers *providers.Manager,
	log *logrus.Logger,
) *NotificationService {
	return &NotificationService{
		notificationRepo: repository.NewNotificationRepository(mongo),
		deviceRepo:       repository.NewDeviceRepository(mysql),
		prefsRepo:        repository.NewPreferencesRepository(mysql),
		providers:        providers,
		redis:            redis,
		log:              log,
	}
}

// Send sends a notification
func (s *NotificationService) Send(ctx context.Context, req *models.NotificationRequest) (*models.Notification, error) {
	// Check user preferences
	enabled, err := s.prefsRepo.IsChannelEnabled(ctx, req.UserID, req.Category, req.Type)
	if err != nil {
		s.log.Warnf("Failed to check preferences: %v", err)
	} else if !enabled {
		s.log.Infof("Notification channel %s disabled for user %s", req.Type, req.UserID)
		return nil, fmt.Errorf("notification channel %s is disabled for this user", req.Type)
	}

	// Check quiet hours
	if s.isQuietHours(ctx, req.UserID) && req.Priority != models.PriorityCritical {
		s.log.Infof("Quiet hours active for user %s, skipping non-critical notification", req.UserID)
		return nil, fmt.Errorf("quiet hours active, notification queued")
	}

	// Check rate limiting
	if s.isRateLimited(ctx, req.UserID, req.Type) {
		s.log.Warnf("Rate limit exceeded for user %s", req.UserID)
		return nil, fmt.Errorf("rate limit exceeded")
	}

	// Create notification record
	notification := s.createNotification(req)

	// Save to database
	if err := s.notificationRepo.Create(ctx, notification); err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	// Send notification based on type
	var sendErr error
	switch req.Type {
	case models.NotificationTypePush:
		sendErr = s.sendPush(ctx, req, notification)
	case models.NotificationTypeEmail:
		sendErr = s.sendEmail(ctx, req, notification)
	case models.NotificationTypeSMS:
		sendErr = s.sendSMS(ctx, req, notification)
	case models.NotificationTypeInApp:
		sendErr = s.sendInApp(ctx, notification)
	default:
		sendErr = fmt.Errorf("unknown notification type: %s", req.Type)
	}

	// Update status based on result
	if sendErr != nil {
		notification.Status = models.StatusFailed
		notification.Error = sendErr.Error()
		s.notificationRepo.Update(ctx, notification)
		return notification, sendErr
	}

	notification.Status = models.StatusSent
	now := time.Now()
	notification.SentAt = &now
	s.notificationRepo.Update(ctx, notification)

	// Update rate limit counter
	s.incrementRateLimit(ctx, req.UserID, req.Type)

	return notification, nil
}

// SendBulk sends notifications to multiple users
func (s *NotificationService) SendBulk(ctx context.Context, req *models.BulkNotificationRequest) ([]string, []error) {
	var notificationIDs []string
	var errors []error

	for _, userID := range req.UserIDs {
		singleReq := &models.NotificationRequest{
			UserID:    userID,
			Type:      req.Type,
			Category:  req.Category,
			Priority:  req.Priority,
			Title:     req.Title,
			Body:      req.Body,
			Data:      req.Data,
			ImageURL:  req.ImageURL,
			ActionURL: req.ActionURL,
		}

		notification, err := s.Send(ctx, singleReq)
		if err != nil {
			errors = append(errors, fmt.Errorf("user %s: %w", userID, err))
		} else {
			notificationIDs = append(notificationIDs, notification.ID.Hex())
		}
	}

	return notificationIDs, errors
}

// GetNotifications retrieves notifications for a user
func (s *NotificationService) GetNotifications(ctx context.Context, userID string, limit, offset int) ([]*models.Notification, error) {
	return s.notificationRepo.GetByUserID(ctx, userID, limit, offset)
}

// GetUnreadNotifications retrieves unread notifications
func (s *NotificationService) GetUnreadNotifications(ctx context.Context, userID string) ([]*models.Notification, error) {
	return s.notificationRepo.GetUnreadByUserID(ctx, userID)
}

// GetUnreadCount returns count of unread notifications
func (s *NotificationService) GetUnreadCount(ctx context.Context, userID string) (int64, error) {
	return s.notificationRepo.CountUnreadByUserID(ctx, userID)
}

// MarkAsRead marks a notification as read
func (s *NotificationService) MarkAsRead(ctx context.Context, notificationID string) error {
	return s.notificationRepo.MarkAsRead(ctx, notificationID)
}

// MarkAllAsRead marks all notifications as read for a user
func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID string) error {
	return s.notificationRepo.MarkAllAsRead(ctx, userID)
}

// RegisterDevice registers a device for push notifications
func (s *NotificationService) RegisterDevice(ctx context.Context, req *models.DeviceRegistrationRequest) (*models.Device, error) {
	device := &models.Device{
		UserID:      req.UserID,
		DeviceToken: req.DeviceToken,
		Platform:    req.Platform,
		DeviceModel: req.DeviceModel,
		OSVersion:   req.OSVersion,
		AppVersion:  req.AppVersion,
		PushEnabled: true,
	}

	if err := s.deviceRepo.Create(ctx, device); err != nil {
		return nil, fmt.Errorf("failed to register device: %w", err)
	}

	return device, nil
}

// UnregisterDevice removes a device
func (s *NotificationService) UnregisterDevice(ctx context.Context, token string) error {
	return s.deviceRepo.Delete(ctx, token)
}

// GetPreferences retrieves user notification preferences
func (s *NotificationService) GetPreferences(ctx context.Context, userID string) (*models.NotificationPreferences, error) {
	return s.prefsRepo.GetByUserID(ctx, userID)
}

// UpdatePreferences updates user notification preferences
func (s *NotificationService) UpdatePreferences(ctx context.Context, userID string, req *models.PreferencesUpdateRequest) (*models.NotificationPreferences, error) {
	prefs, err := s.prefsRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if req.PushEnabled != nil {
		prefs.PushEnabled = *req.PushEnabled
	}
	if req.EmailEnabled != nil {
		prefs.EmailEnabled = *req.EmailEnabled
	}
	if req.SMSEnabled != nil {
		prefs.SMSEnabled = *req.SMSEnabled
	}
	if req.InAppEnabled != nil {
		prefs.InAppEnabled = *req.InAppEnabled
	}
	if req.QuietHoursEnabled != nil {
		prefs.QuietHoursEnabled = *req.QuietHoursEnabled
	}
	if req.QuietHoursStart != "" {
		prefs.QuietHoursStart = req.QuietHoursStart
	}
	if req.QuietHoursEnd != "" {
		prefs.QuietHoursEnd = req.QuietHoursEnd
	}
	if req.Timezone != "" {
		prefs.Timezone = req.Timezone
	}
	if req.CategorySettings != nil {
		prefs.CategorySettings = models.CategorySettingsJSON{CategorySettings: *req.CategorySettings}
	}
	if req.EmailDigestEnabled != nil {
		prefs.EmailDigestEnabled = *req.EmailDigestEnabled
	}
	if req.EmailDigestFrequency != "" {
		prefs.EmailDigestFrequency = req.EmailDigestFrequency
	}

	if err := s.prefsRepo.Upsert(ctx, prefs); err != nil {
		return nil, err
	}

	return prefs, nil
}

// createNotification creates a notification from request
func (s *NotificationService) createNotification(req *models.NotificationRequest) *models.Notification {
	notification := &models.Notification{
		ID:        primitive.NewObjectID(),
		UserID:    req.UserID,
		Type:      req.Type,
		Category:  req.Category,
		Priority:  req.Priority,
		Title:     req.Title,
		Body:      req.Body,
		Data:      req.Data,
		ImageURL:  req.ImageURL,
		ActionURL: req.ActionURL,
		Status:    models.StatusPending,
	}

	if notification.Priority == "" {
		notification.Priority = models.PriorityNormal
	}

	if req.ExpiresIn > 0 {
		expiry := time.Now().Add(time.Duration(req.ExpiresIn) * time.Second)
		notification.ExpiresAt = &expiry
	}

	return notification
}

// sendPush sends a push notification
func (s *NotificationService) sendPush(ctx context.Context, req *models.NotificationRequest, notification *models.Notification) error {
	// Get device tokens
	tokens := req.DeviceTokens
	if len(tokens) == 0 {
		var err error
		tokens, err = s.deviceRepo.GetTokensByUserID(ctx, req.UserID, req.Platform)
		if err != nil {
			return fmt.Errorf("failed to get device tokens: %w", err)
		}
	}

	if len(tokens) == 0 {
		return fmt.Errorf("no device tokens found for user")
	}

	pushNotification := &models.PushNotification{
		Notification: *notification,
		DeviceTokens: tokens,
		Platform:     req.Platform,
		Badge:        req.Badge,
		Sound:        req.Sound,
	}

	return s.providers.SendPushNotification(ctx, pushNotification)
}

// sendEmail sends an email notification
func (s *NotificationService) sendEmail(ctx context.Context, req *models.NotificationRequest, notification *models.Notification) error {
	emailNotification := &models.EmailNotification{
		Notification: *notification,
		ToEmail:      req.ToEmail,
		ToName:       req.ToName,
		Subject:      req.Subject,
		HTMLBody:     req.HTMLBody,
		TemplateID:   req.TemplateID,
		TemplateData: req.TemplateData,
	}

	if emailNotification.ToEmail == "" {
		return fmt.Errorf("email address is required")
	}

	return s.providers.SendEmailNotification(ctx, emailNotification)
}

// sendSMS sends an SMS notification
func (s *NotificationService) sendSMS(ctx context.Context, req *models.NotificationRequest, notification *models.Notification) error {
	smsNotification := &models.SMSNotification{
		Notification: *notification,
		PhoneNumber:  req.PhoneNumber,
	}

	if smsNotification.PhoneNumber == "" {
		return fmt.Errorf("phone number is required")
	}

	return s.providers.SendSMSNotification(ctx, smsNotification)
}

// sendInApp saves in-app notification (already saved, just update status)
func (s *NotificationService) sendInApp(ctx context.Context, notification *models.Notification) error {
	// In-app notifications are stored, no external send needed
	notification.Status = models.StatusDelivered
	now := time.Now()
	notification.DeliveredAt = &now
	return nil
}

// isQuietHours checks if quiet hours are active for a user
func (s *NotificationService) isQuietHours(ctx context.Context, userID string) bool {
	prefs, err := s.prefsRepo.GetByUserID(ctx, userID)
	if err != nil || !prefs.QuietHoursEnabled {
		return false
	}

	// Parse quiet hours
	loc, err := time.LoadLocation(prefs.Timezone)
	if err != nil {
		loc = time.UTC
	}

	now := time.Now().In(loc)
	currentTime := now.Format("15:04")

	// Simple comparison (doesn't handle overnight ranges perfectly)
	if prefs.QuietHoursStart <= prefs.QuietHoursEnd {
		return currentTime >= prefs.QuietHoursStart && currentTime < prefs.QuietHoursEnd
	}
	// Overnight range (e.g., 22:00 to 08:00)
	return currentTime >= prefs.QuietHoursStart || currentTime < prefs.QuietHoursEnd
}

// isRateLimited checks if user has exceeded rate limit
func (s *NotificationService) isRateLimited(ctx context.Context, userID string, notifType models.NotificationType) bool {
	key := fmt.Sprintf("ratelimit:%s:%s", userID, notifType)
	count, err := s.redis.Get(ctx, key).Int()
	if err == redis.Nil {
		return false
	}
	if err != nil {
		s.log.Warnf("Failed to check rate limit: %v", err)
		return false
	}

	// Limits per hour
	limits := map[models.NotificationType]int{
		models.NotificationTypePush:  100,
		models.NotificationTypeEmail: 20,
		models.NotificationTypeSMS:   10,
		models.NotificationTypeInApp:  200,
	}

	limit, ok := limits[notifType]
	if !ok {
		limit = 50
	}

	return count >= limit
}

// incrementRateLimit increments rate limit counter
func (s *NotificationService) incrementRateLimit(ctx context.Context, userID string, notifType models.NotificationType) {
	key := fmt.Sprintf("ratelimit:%s:%s", userID, notifType)
	s.redis.Incr(ctx, key)
	s.redis.Expire(ctx, key, time.Hour)
}

// RetryFailed retries failed notifications
func (s *NotificationService) RetryFailed(ctx context.Context) error {
	notifications, err := s.notificationRepo.GetPending(ctx, 100)
	if err != nil {
		return err
	}

	for _, n := range notifications {
		req := &models.NotificationRequest{
			UserID:    n.UserID,
			Type:      n.Type,
			Category:  n.Category,
			Priority:  n.Priority,
			Title:     n.Title,
			Body:      n.Body,
			Data:      n.Data,
			ImageURL:  n.ImageURL,
			ActionURL: n.ActionURL,
		}

		_, err := s.Send(ctx, req)
		if err != nil {
			s.notificationRepo.IncrementRetryCount(ctx, n.ID.Hex(), err.Error())
		}
	}

	return nil
}
