package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/quickchat/notification-service/internal/api"
	"github.com/quickchat/notification-service/internal/config"
	"github.com/quickchat/notification-service/internal/consumer"
	"github.com/quickchat/notification-service/internal/db"
	"github.com/quickchat/notification-service/internal/providers"
	"github.com/quickchat/notification-service/internal/service"
	"github.com/quickchat/notification-service/internal/worker"
	"github.com/sirupsen/logrus"
)

func main() {
	// Initialize logger
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetLevel(logrus.InfoLevel)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set log level from config
	level, _ := logrus.ParseLevel(cfg.LogLevel)
	log.SetLevel(level)

	log.Info("Starting QuickChat Notification Service...")

	// Initialize databases
	mysqlDB, err := db.NewMySQL(cfg.MySQL)
	if err != nil {
		log.Fatalf("Failed to connect to MySQL: %v", err)
	}
	defer mysqlDB.Close()

	mongoDB, err := db.NewMongoDB(cfg.MongoDB)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoDB.Close()

	redisClient, err := db.NewRedis(cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Initialize notification providers
	providerManager := providers.NewManager(cfg, log)
	if err := providerManager.Initialize(); err != nil {
		log.Fatalf("Failed to initialize providers: %v", err)
	}

	// Initialize notification service
	notificationService := service.NewNotificationService(
		mysqlDB,
		mongoDB,
		redisClient,
		providerManager,
		log,
	)

	// Initialize worker pool
	workerPool := worker.NewPool(cfg.Worker.PoolSize, notificationService, log)
	workerPool.Start()
	defer workerPool.Stop()

	// Initialize Kafka consumer
	kafkaConsumer := consumer.NewKafkaConsumer(cfg.Kafka, workerPool, log)
	go kafkaConsumer.Start()
	defer kafkaConsumer.Stop()

	// Initialize API router
	router := api.NewRouter(notificationService, cfg, log)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Infof("Server starting on port %s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Errorf("Server forced to shutdown: %v", err)
	}

	log.Info("Server stopped")
}
