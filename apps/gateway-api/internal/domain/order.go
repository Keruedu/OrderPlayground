package domain

import (
	"errors"
	"strings"
	"time"
)

const (
	OrderStatusPending   = "PENDING"
	OrderStatusCompleted = "COMPLETED"
	OrderStatusFailed    = "FAILED"
	OrderStatusCanceled  = "CANCELED"

	WorkflowStatusRunning   = "RUNNING"
	WorkflowStatusCompleted = "COMPLETED"
	WorkflowStatusFailed    = "FAILED"
	WorkflowStatusCanceled  = "CANCELED"
)

type Order struct {
	ID           string        `json:"id"`
	CustomerName string        `json:"customer_name"`
	Currency     string        `json:"currency"`
	Status       string        `json:"status"`
	CreatedBy    string        `json:"created_by"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	Items        []OrderItem   `json:"items"`
	Workflow     *WorkflowRun  `json:"workflow,omitempty"`
	AuditSummary *AuditSummary `json:"audit_summary,omitempty"`
}

type OrderItem struct {
	ID       int64   `json:"id"`
	OrderID  string  `json:"order_id"`
	SKU      string  `json:"sku"`
	Quantity int32   `json:"quantity"`
	Price    float64 `json:"price"`
}

type WorkflowRun struct {
	ID         int64     `json:"id"`
	OrderID    string    `json:"order_id"`
	WorkflowID string    `json:"workflow_id"`
	RunID      string    `json:"run_id"`
	Status     string    `json:"status"`
	StartedAt  time.Time `json:"started_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type AuditEvent struct {
	OrderID   string         `json:"order_id" bson:"orderId"`
	Type      string         `json:"type" bson:"type"`
	Message   string         `json:"message" bson:"message"`
	Actor     string         `json:"actor" bson:"actor"`
	Source    string         `json:"source" bson:"source"`
	CreatedAt time.Time      `json:"created_at" bson:"createdAt"`
	Metadata  map[string]any `json:"metadata,omitempty" bson:"metadata,omitempty"`
}

type AuditSummary struct {
	Count      int        `json:"count"`
	LastType   string     `json:"last_type,omitempty"`
	LastAt     *time.Time `json:"last_at,omitempty"`
	LastSource string     `json:"last_source,omitempty"`
}

type RequestLog struct {
	RequestID string    `bson:"requestId"`
	Path      string    `bson:"path"`
	Method    string    `bson:"method"`
	Status    int       `bson:"status"`
	LatencyMS int64     `bson:"latencyMs"`
	User      string    `bson:"user,omitempty"`
	CreatedAt time.Time `bson:"createdAt"`
}

type CreateOrderRequest struct {
	CustomerName string            `json:"customer_name"`
	Currency     string            `json:"currency"`
	Items        []CreateOrderItem `json:"items"`
}

type CreateOrderItem struct {
	SKU      string  `json:"sku"`
	Quantity int32   `json:"quantity"`
	Price    float64 `json:"price"`
}

func (r CreateOrderRequest) Validate() error {
	if strings.TrimSpace(r.CustomerName) == "" {
		return errors.New("customer_name is required")
	}
	if strings.TrimSpace(r.Currency) == "" {
		return errors.New("currency is required")
	}
	if len(r.Items) == 0 {
		return errors.New("at least one item is required")
	}
	for _, item := range r.Items {
		if strings.TrimSpace(item.SKU) == "" {
			return errors.New("item sku is required")
		}
		if item.Quantity <= 0 {
			return errors.New("item quantity must be positive")
		}
		if item.Price <= 0 {
			return errors.New("item price must be positive")
		}
	}
	return nil
}

type CreateOrderResponse struct {
	OrderID    string    `json:"order_id"`
	Status     string    `json:"status"`
	WorkflowID string    `json:"workflow_id"`
	CreatedBy  string    `json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
}

type UserClaims struct {
	Subject           string
	Username          string
	Roles             []string
	RawTokenExpiresAt time.Time
}

func (u UserClaims) HasAnyRole(roles ...string) bool {
	roleSet := map[string]struct{}{}
	for _, role := range u.Roles {
		roleSet[role] = struct{}{}
	}
	for _, role := range roles {
		if _, ok := roleSet[role]; ok {
			return true
		}
	}
	return false
}

type WorkflowInput struct {
	OrderID      string            `json:"order_id"`
	CustomerName string            `json:"customer_name"`
	Currency     string            `json:"currency"`
	Items        []CreateOrderItem `json:"items"`
	CreatedBy    string            `json:"created_by"`
}
