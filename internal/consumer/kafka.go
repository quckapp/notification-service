package consumer

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/quickchat/notification-service/internal/config"
	"github.com/quickchat/notification-service/internal/models"
	"github.com/quickchat/notification-service/internal/worker"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// KafkaConsumer consumes notification messages from Kafka
type KafkaConsumer struct {
	reader *kafka.Reader
	pool   *worker.Pool
	log    *logrus.Logger
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

// KafkaMessage represents a notification message from Kafka
type KafkaMessage struct {
	EventType string                       `json:"event_type"`
	Payload   *models.NotificationRequest  `json:"payload"`
	Timestamp time.Time                    `json:"timestamp"`
}

// NewKafkaConsumer creates a new Kafka consumer
func NewKafkaConsumer(cfg config.KafkaConfig, pool *worker.Pool, log *logrus.Logger) *KafkaConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		Topic:          cfg.Topic,
		GroupID:        cfg.ConsumerGroup,
		MinBytes:       10e3,        // 10KB
		MaxBytes:       10e6,        // 10MB
		MaxWait:        time.Second,
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
	})

	ctx, cancel := context.WithCancel(context.Background())

	return &KafkaConsumer{
		reader: reader,
		pool:   pool,
		log:    log,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start starts consuming messages
func (c *KafkaConsumer) Start() {
	c.log.Info("Starting Kafka consumer...")
	c.wg.Add(1)
	go c.consume()
}

// Stop stops the consumer
func (c *KafkaConsumer) Stop() {
	c.log.Info("Stopping Kafka consumer...")
	c.cancel()
	c.wg.Wait()
	c.reader.Close()
	c.log.Info("Kafka consumer stopped")
}

// consume consumes messages from Kafka
func (c *KafkaConsumer) consume() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			msg, err := c.reader.ReadMessage(c.ctx)
			if err != nil {
				if c.ctx.Err() != nil {
					return
				}
				c.log.Errorf("Failed to read Kafka message: %v", err)
				continue
			}

			c.processMessage(msg)
		}
	}
}

// processMessage processes a single Kafka message
func (c *KafkaConsumer) processMessage(msg kafka.Message) {
	var kafkaMsg KafkaMessage
	if err := json.Unmarshal(msg.Value, &kafkaMsg); err != nil {
		c.log.Errorf("Failed to unmarshal Kafka message: %v", err)
		return
	}

	c.log.Debugf("Received notification event: %s for user %s", kafkaMsg.EventType, kafkaMsg.Payload.UserID)

	// Handle different event types
	switch kafkaMsg.EventType {
	case "send_notification":
		c.handleSendNotification(kafkaMsg.Payload)
	case "send_bulk_notification":
		c.handleBulkNotification(kafkaMsg.Payload)
	default:
		c.log.Warnf("Unknown event type: %s", kafkaMsg.EventType)
	}
}

// handleSendNotification handles single notification events
func (c *KafkaConsumer) handleSendNotification(req *models.NotificationRequest) {
	if err := c.pool.SubmitAsync(req); err != nil {
		c.log.Errorf("Failed to submit notification job: %v", err)
	}
}

// handleBulkNotification handles bulk notification events
func (c *KafkaConsumer) handleBulkNotification(req *models.NotificationRequest) {
	// For bulk notifications, we'd receive multiple user IDs
	// and fan out to individual notifications
	if err := c.pool.SubmitAsync(req); err != nil {
		c.log.Errorf("Failed to submit bulk notification job: %v", err)
	}
}
