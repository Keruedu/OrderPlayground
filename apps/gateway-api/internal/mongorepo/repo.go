package mongorepo

import (
	"context"
	"time"

	"order-playground/apps/gateway-api/internal/domain"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Repo struct {
	db *mongo.Database
}

func New(db *mongo.Database) *Repo {
	return &Repo{db: db}
}

func (r *Repo) WriteAuditEvent(ctx context.Context, event domain.AuditEvent) error {
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	_, err := r.db.Collection("audit_events").InsertOne(ctx, event)
	return err
}

func (r *Repo) ListOrderEvents(ctx context.Context, orderID string) ([]domain.AuditEvent, error) {
	cursor, err := r.db.Collection("audit_events").Find(ctx, bson.M{"orderId": orderID}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var events []domain.AuditEvent
	if err := cursor.All(ctx, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (r *Repo) SummarizeOrderEvents(ctx context.Context, orderID string) (*domain.AuditSummary, error) {
	events, err := r.ListOrderEvents(ctx, orderID)
	if err != nil {
		return nil, err
	}
	summary := &domain.AuditSummary{Count: len(events)}
	if len(events) == 0 {
		return summary, nil
	}
	last := events[len(events)-1]
	summary.LastType = last.Type
	summary.LastSource = last.Source
	summary.LastAt = &last.CreatedAt
	return summary, nil
}

func (r *Repo) WriteRequestLog(ctx context.Context, entry domain.RequestLog) error {
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	_, err := r.db.Collection("request_logs").InsertOne(ctx, entry)
	return err
}

func (r *Repo) Ping(ctx context.Context) error {
	return r.db.Client().Ping(ctx, nil)
}
