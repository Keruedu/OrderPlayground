package service

import (
	"context"
	"fmt"
	"time"

	"order-playground/apps/gateway-api/internal/domain"
)

type orderRepo interface {
	CreateOrder(context.Context, domain.Order) error
	GetOrder(context.Context, string) (*domain.Order, error)
	ListOrders(context.Context) ([]domain.Order, error)
	UpdateOrderStatus(context.Context, string, string) error
	CreateWorkflowRun(context.Context, domain.WorkflowRun) error
	UpdateWorkflowRun(context.Context, string, string, string) error
	GetWorkflowRunByWorkflowID(context.Context, string) (*domain.WorkflowRun, error)
}

type auditRepo interface {
	WriteAuditEvent(context.Context, domain.AuditEvent) error
	ListOrderEvents(context.Context, string) ([]domain.AuditEvent, error)
	SummarizeOrderEvents(context.Context, string) (*domain.AuditSummary, error)
}

type workflowStarter interface {
	StartCreateOrderWorkflow(context.Context, domain.WorkflowInput) (string, string, error)
}

type OrderService struct {
	orders    orderRepo
	audit     auditRepo
	workflows workflowStarter
	now       func() time.Time
	idgen     func() string
}

func NewOrderService(orderRepo orderRepo, auditRepo auditRepo, workflows workflowStarter, now func() time.Time, idgen func() string) *OrderService {
	return &OrderService{
		orders:    orderRepo,
		audit:     auditRepo,
		workflows: workflows,
		now:       now,
		idgen:     idgen,
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, user domain.UserClaims, req domain.CreateOrderRequest) (*domain.CreateOrderResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	now := s.now().UTC()
	order := domain.Order{
		ID:           s.idgen(),
		CustomerName: req.CustomerName,
		Currency:     req.Currency,
		Status:       domain.OrderStatusPending,
		CreatedBy:    user.Username,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	for _, item := range req.Items {
		order.Items = append(order.Items, domain.OrderItem{
			OrderID:  order.ID,
			SKU:      item.SKU,
			Quantity: item.Quantity,
			Price:    item.Price,
		})
	}

	if err := s.orders.CreateOrder(ctx, order); err != nil {
		return nil, err
	}

	if err := s.audit.WriteAuditEvent(ctx, domain.AuditEvent{
		OrderID:   order.ID,
		Type:      "ORDER_CREATED",
		Message:   "Order accepted by gateway",
		Actor:     user.Username,
		Source:    "gateway-api",
		CreatedAt: now,
	}); err != nil {
		return nil, err
	}

	workflowID, runID, err := s.workflows.StartCreateOrderWorkflow(ctx, domain.WorkflowInput{
		OrderID:      order.ID,
		CustomerName: order.CustomerName,
		Currency:     order.Currency,
		Items:        req.Items,
		CreatedBy:    user.Username,
	})
	if err != nil {
		return nil, err
	}

	if err := s.orders.CreateWorkflowRun(ctx, domain.WorkflowRun{
		OrderID:    order.ID,
		WorkflowID: workflowID,
		RunID:      runID,
		Status:     domain.WorkflowStatusRunning,
		StartedAt:  now,
		UpdatedAt:  now,
	}); err != nil {
		return nil, err
	}

	return &domain.CreateOrderResponse{
		OrderID:    order.ID,
		Status:     order.Status,
		WorkflowID: workflowID,
		CreatedBy:  user.Username,
		CreatedAt:  now,
	}, nil
}

func (s *OrderService) GetOrder(ctx context.Context, orderID string) (*domain.Order, error) {
	order, err := s.orders.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	summary, err := s.audit.SummarizeOrderEvents(ctx, orderID)
	if err == nil {
		order.AuditSummary = summary
	}
	return order, nil
}

func (s *OrderService) ListOrders(ctx context.Context) ([]domain.Order, error) {
	return s.orders.ListOrders(ctx)
}

func (s *OrderService) GetEvents(ctx context.Context, orderID string) ([]domain.AuditEvent, error) {
	return s.audit.ListOrderEvents(ctx, orderID)
}

func (s *OrderService) CancelOrder(ctx context.Context, orderID string, user domain.UserClaims) error {
	if err := s.orders.UpdateOrderStatus(ctx, orderID, domain.OrderStatusCanceled); err != nil {
		return err
	}
	return s.audit.WriteAuditEvent(ctx, domain.AuditEvent{
		OrderID:   orderID,
		Type:      "ORDER_CANCELED",
		Message:   fmt.Sprintf("Order canceled by %s", user.Username),
		Actor:     user.Username,
		Source:    "gateway-api",
		CreatedAt: s.now().UTC(),
	})
}

func (s *OrderService) GetWorkflow(ctx context.Context, workflowID string) (*domain.WorkflowRun, error) {
	return s.orders.GetWorkflowRunByWorkflowID(ctx, workflowID)
}
