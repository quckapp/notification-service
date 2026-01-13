package repository

import (
	"context"
	"time"

	"github.com/quickchat/notification-service/internal/db"
	"github.com/quickchat/notification-service/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const notificationsCollection = "notifications"

// NotificationRepository handles notification persistence
type NotificationRepository struct {
	mongo *db.MongoDB
}

// NewNotificationRepository creates a new notification repository
func NewNotificationRepository(mongo *db.MongoDB) *NotificationRepository {
	return &NotificationRepository{mongo: mongo}
}

// Create creates a new notification
func (r *NotificationRepository) Create(ctx context.Context, notification *models.Notification) error {
	notification.CreatedAt = time.Now()
	notification.UpdatedAt = time.Now()

	result, err := r.mongo.Collection(notificationsCollection).InsertOne(ctx, notification)
	if err != nil {
		return err
	}

	notification.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// GetByID retrieves a notification by ID
func (r *NotificationRepository) GetByID(ctx context.Context, id string) (*models.Notification, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var notification models.Notification
	err = r.mongo.Collection(notificationsCollection).FindOne(ctx, bson.M{"_id": objectID}).Decode(&notification)
	if err != nil {
		return nil, err
	}

	return &notification, nil
}

// GetByUserID retrieves notifications for a user
func (r *NotificationRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Notification, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cursor, err := r.mongo.Collection(notificationsCollection).Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var notifications []*models.Notification
	if err := cursor.All(ctx, &notifications); err != nil {
		return nil, err
	}

	return notifications, nil
}

// GetUnreadByUserID retrieves unread notifications for a user
func (r *NotificationRepository) GetUnreadByUserID(ctx context.Context, userID string) ([]*models.Notification, error) {
	filter := bson.M{
		"user_id": userID,
		"status":  bson.M{"$nin": []string{string(models.StatusRead)}},
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.mongo.Collection(notificationsCollection).Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var notifications []*models.Notification
	if err := cursor.All(ctx, &notifications); err != nil {
		return nil, err
	}

	return notifications, nil
}

// Update updates a notification
func (r *NotificationRepository) Update(ctx context.Context, notification *models.Notification) error {
	notification.UpdatedAt = time.Now()

	_, err := r.mongo.Collection(notificationsCollection).UpdateOne(
		ctx,
		bson.M{"_id": notification.ID},
		bson.M{"$set": notification},
	)
	return err
}

// UpdateStatus updates notification status
func (r *NotificationRepository) UpdateStatus(ctx context.Context, id string, status models.NotificationStatus) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	switch status {
	case models.StatusSent:
		now := time.Now()
		update["$set"].(bson.M)["sent_at"] = &now
	case models.StatusDelivered:
		now := time.Now()
		update["$set"].(bson.M)["delivered_at"] = &now
	case models.StatusRead:
		now := time.Now()
		update["$set"].(bson.M)["read_at"] = &now
	}

	_, err = r.mongo.Collection(notificationsCollection).UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

// MarkAsRead marks a notification as read
func (r *NotificationRepository) MarkAsRead(ctx context.Context, id string) error {
	return r.UpdateStatus(ctx, id, models.StatusRead)
}

// MarkAllAsRead marks all notifications for a user as read
func (r *NotificationRepository) MarkAllAsRead(ctx context.Context, userID string) error {
	now := time.Now()
	_, err := r.mongo.Collection(notificationsCollection).UpdateMany(
		ctx,
		bson.M{
			"user_id": userID,
			"status":  bson.M{"$ne": string(models.StatusRead)},
		},
		bson.M{
			"$set": bson.M{
				"status":     models.StatusRead,
				"read_at":    &now,
				"updated_at": now,
			},
		},
	)
	return err
}

// Delete deletes a notification
func (r *NotificationRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.mongo.Collection(notificationsCollection).DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

// DeleteOlderThan deletes notifications older than duration
func (r *NotificationRepository) DeleteOlderThan(ctx context.Context, duration time.Duration) (int64, error) {
	cutoff := time.Now().Add(-duration)

	result, err := r.mongo.Collection(notificationsCollection).DeleteMany(
		ctx,
		bson.M{"created_at": bson.M{"$lt": cutoff}},
	)
	if err != nil {
		return 0, err
	}

	return result.DeletedCount, nil
}

// GetPending retrieves pending notifications for retry
func (r *NotificationRepository) GetPending(ctx context.Context, limit int) ([]*models.Notification, error) {
	filter := bson.M{
		"status":      models.StatusPending,
		"retry_count": bson.M{"$lt": 3},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: 1}}).
		SetLimit(int64(limit))

	cursor, err := r.mongo.Collection(notificationsCollection).Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var notifications []*models.Notification
	if err := cursor.All(ctx, &notifications); err != nil {
		return nil, err
	}

	return notifications, nil
}

// IncrementRetryCount increments retry count and updates error
func (r *NotificationRepository) IncrementRetryCount(ctx context.Context, id string, errMsg string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = r.mongo.Collection(notificationsCollection).UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{
			"$inc": bson.M{"retry_count": 1},
			"$set": bson.M{
				"error":      errMsg,
				"updated_at": time.Now(),
			},
		},
	)
	return err
}

// CountByUserID counts notifications for a user
func (r *NotificationRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	return r.mongo.Collection(notificationsCollection).CountDocuments(ctx, bson.M{"user_id": userID})
}

// CountUnreadByUserID counts unread notifications for a user
func (r *NotificationRepository) CountUnreadByUserID(ctx context.Context, userID string) (int64, error) {
	filter := bson.M{
		"user_id": userID,
		"status":  bson.M{"$ne": string(models.StatusRead)},
	}
	return r.mongo.Collection(notificationsCollection).CountDocuments(ctx, filter)
}

// EnsureIndexes creates necessary indexes
func (r *NotificationRepository) EnsureIndexes(ctx context.Context) error {
	collection := r.mongo.Collection(notificationsCollection)

	indexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}, {Key: "retry_count", Value: 1}}},
		{Keys: bson.D{{Key: "created_at", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(7776000)}, // 90 days TTL
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}
