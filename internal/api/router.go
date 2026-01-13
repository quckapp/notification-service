package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/quickchat/notification-service/internal/config"
	"github.com/quickchat/notification-service/internal/service"
	"github.com/sirupsen/logrus"
)

// NewRouter creates a new Gin router
func NewRouter(svc *service.NotificationService, cfg *config.Config, log *logrus.Logger) *gin.Engine {
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Middleware
	router.Use(gin.Recovery())
	router.Use(requestLogger(log))
	router.Use(corsMiddleware())

	// Health check endpoints
	router.GET("/health", healthCheck())
	router.GET("/ready", readinessCheck(svc))

	// Metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API routes
	api := router.Group("/api/v1")
	{
		// Notification handlers
		handler := NewNotificationHandler(svc, log)

		// Send notifications
		api.POST("/notifications", handler.Send)
		api.POST("/notifications/bulk", handler.SendBulk)

		// Get notifications
		api.GET("/notifications/:user_id", handler.GetNotifications)
		api.GET("/notifications/:user_id/unread", handler.GetUnreadNotifications)
		api.GET("/notifications/:user_id/count", handler.GetUnreadCount)

		// Mark as read
		api.PUT("/notifications/:id/read", handler.MarkAsRead)
		api.PUT("/notifications/:user_id/read-all", handler.MarkAllAsRead)

		// Device management
		api.POST("/devices", handler.RegisterDevice)
		api.DELETE("/devices/:token", handler.UnregisterDevice)
		api.PUT("/devices/:token/active", handler.UpdateDeviceActive)

		// Preferences
		api.GET("/preferences/:user_id", handler.GetPreferences)
		api.PUT("/preferences/:user_id", handler.UpdatePreferences)
	}

	return router
}

// requestLogger middleware logs requests
func requestLogger(log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		log.WithFields(logrus.Fields{
			"status":     status,
			"method":     c.Request.Method,
			"path":       path,
			"query":      query,
			"latency":    latency,
			"ip":         c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		}).Info("request")
	}
}

// corsMiddleware handles CORS
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// healthCheck returns service health
func healthCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "notification-service",
			"timestamp": time.Now().UTC(),
		})
	}
}

// readinessCheck checks if service is ready
func readinessCheck(svc *service.NotificationService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Add actual readiness checks (DB connections, etc.)
		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
		})
	}
}
