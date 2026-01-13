package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/quickchat/notification-service/internal/models"
	"github.com/quickchat/notification-service/internal/service"
	"github.com/sirupsen/logrus"
)

// NotificationHandler handles notification HTTP requests
type NotificationHandler struct {
	service *service.NotificationService
	log     *logrus.Logger
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(svc *service.NotificationService, log *logrus.Logger) *NotificationHandler {
	return &NotificationHandler{
		service: svc,
		log:     log,
	}
}

// Send sends a notification
// @Summary Send a notification
// @Description Send a notification to a user
// @Tags notifications
// @Accept json
// @Produce json
// @Param notification body models.NotificationRequest true "Notification request"
// @Success 200 {object} models.Notification
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /notifications [post]
func (h *NotificationHandler) Send(c *gin.Context) {
	var req models.NotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	notification, err := h.service.Send(c.Request.Context(), &req)
	if err != nil {
		h.log.Errorf("Failed to send notification: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, notification)
}

// SendBulk sends notifications to multiple users
// @Summary Send bulk notifications
// @Description Send notifications to multiple users
// @Tags notifications
// @Accept json
// @Produce json
// @Param notification body models.BulkNotificationRequest true "Bulk notification request"
// @Success 200 {object} BulkSendResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /notifications/bulk [post]
func (h *NotificationHandler) SendBulk(c *gin.Context) {
	var req models.BulkNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	ids, errs := h.service.SendBulk(c.Request.Context(), &req)

	response := BulkSendResponse{
		Sent:   len(ids),
		Failed: len(errs),
		IDs:    ids,
	}

	if len(errs) > 0 {
		response.Errors = make([]string, len(errs))
		for i, err := range errs {
			response.Errors[i] = err.Error()
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetNotifications retrieves notifications for a user
// @Summary Get user notifications
// @Description Get notifications for a specific user
// @Tags notifications
// @Produce json
// @Param user_id path string true "User ID"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} models.Notification
// @Failure 500 {object} ErrorResponse
// @Router /notifications/{user_id} [get]
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	userID := c.Param("user_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	notifications, err := h.service.GetNotifications(c.Request.Context(), userID, limit, offset)
	if err != nil {
		h.log.Errorf("Failed to get notifications: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, notifications)
}

// GetUnreadNotifications retrieves unread notifications
// @Summary Get unread notifications
// @Description Get unread notifications for a user
// @Tags notifications
// @Produce json
// @Param user_id path string true "User ID"
// @Success 200 {array} models.Notification
// @Failure 500 {object} ErrorResponse
// @Router /notifications/{user_id}/unread [get]
func (h *NotificationHandler) GetUnreadNotifications(c *gin.Context) {
	userID := c.Param("user_id")

	notifications, err := h.service.GetUnreadNotifications(c.Request.Context(), userID)
	if err != nil {
		h.log.Errorf("Failed to get unread notifications: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, notifications)
}

// GetUnreadCount returns the count of unread notifications
// @Summary Get unread count
// @Description Get count of unread notifications for a user
// @Tags notifications
// @Produce json
// @Param user_id path string true "User ID"
// @Success 200 {object} UnreadCountResponse
// @Failure 500 {object} ErrorResponse
// @Router /notifications/{user_id}/count [get]
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID := c.Param("user_id")

	count, err := h.service.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		h.log.Errorf("Failed to get unread count: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, UnreadCountResponse{Count: count})
}

// MarkAsRead marks a notification as read
// @Summary Mark notification as read
// @Description Mark a specific notification as read
// @Tags notifications
// @Produce json
// @Param id path string true "Notification ID"
// @Success 200 {object} SuccessResponse
// @Failure 500 {object} ErrorResponse
// @Router /notifications/{id}/read [put]
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.MarkAsRead(c.Request.Context(), id); err != nil {
		h.log.Errorf("Failed to mark as read: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}

// MarkAllAsRead marks all notifications as read
// @Summary Mark all as read
// @Description Mark all notifications as read for a user
// @Tags notifications
// @Produce json
// @Param user_id path string true "User ID"
// @Success 200 {object} SuccessResponse
// @Failure 500 {object} ErrorResponse
// @Router /notifications/{user_id}/read-all [put]
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID := c.Param("user_id")

	if err := h.service.MarkAllAsRead(c.Request.Context(), userID); err != nil {
		h.log.Errorf("Failed to mark all as read: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}

// RegisterDevice registers a device for push notifications
// @Summary Register device
// @Description Register a device for push notifications
// @Tags devices
// @Accept json
// @Produce json
// @Param device body models.DeviceRegistrationRequest true "Device registration"
// @Success 200 {object} models.Device
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /devices [post]
func (h *NotificationHandler) RegisterDevice(c *gin.Context) {
	var req models.DeviceRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	device, err := h.service.RegisterDevice(c.Request.Context(), &req)
	if err != nil {
		h.log.Errorf("Failed to register device: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, device)
}

// UnregisterDevice removes a device
// @Summary Unregister device
// @Description Remove a device from push notifications
// @Tags devices
// @Produce json
// @Param token path string true "Device token"
// @Success 200 {object} SuccessResponse
// @Failure 500 {object} ErrorResponse
// @Router /devices/{token} [delete]
func (h *NotificationHandler) UnregisterDevice(c *gin.Context) {
	token := c.Param("token")

	if err := h.service.UnregisterDevice(c.Request.Context(), token); err != nil {
		h.log.Errorf("Failed to unregister device: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}

// UpdateDeviceActive updates device last active timestamp
// @Summary Update device active
// @Description Update device last active timestamp
// @Tags devices
// @Produce json
// @Param token path string true "Device token"
// @Success 200 {object} SuccessResponse
// @Failure 500 {object} ErrorResponse
// @Router /devices/{token}/active [put]
func (h *NotificationHandler) UpdateDeviceActive(c *gin.Context) {
	// This would be implemented with the device repository
	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}

// GetPreferences retrieves user preferences
// @Summary Get preferences
// @Description Get notification preferences for a user
// @Tags preferences
// @Produce json
// @Param user_id path string true "User ID"
// @Success 200 {object} models.NotificationPreferences
// @Failure 500 {object} ErrorResponse
// @Router /preferences/{user_id} [get]
func (h *NotificationHandler) GetPreferences(c *gin.Context) {
	userID := c.Param("user_id")

	prefs, err := h.service.GetPreferences(c.Request.Context(), userID)
	if err != nil {
		h.log.Errorf("Failed to get preferences: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, prefs)
}

// UpdatePreferences updates user preferences
// @Summary Update preferences
// @Description Update notification preferences for a user
// @Tags preferences
// @Accept json
// @Produce json
// @Param user_id path string true "User ID"
// @Param preferences body models.PreferencesUpdateRequest true "Preferences update"
// @Success 200 {object} models.NotificationPreferences
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /preferences/{user_id} [put]
func (h *NotificationHandler) UpdatePreferences(c *gin.Context) {
	userID := c.Param("user_id")

	var req models.PreferencesUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	prefs, err := h.service.UpdatePreferences(c.Request.Context(), userID, &req)
	if err != nil {
		h.log.Errorf("Failed to update preferences: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, prefs)
}

// Response types

type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessResponse struct {
	Success bool `json:"success"`
}

type UnreadCountResponse struct {
	Count int64 `json:"count"`
}

type BulkSendResponse struct {
	Sent   int      `json:"sent"`
	Failed int      `json:"failed"`
	IDs    []string `json:"ids,omitempty"`
	Errors []string `json:"errors,omitempty"`
}
