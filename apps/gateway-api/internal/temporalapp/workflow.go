package temporalapp

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"order-playground/apps/gateway-api/internal/domain"
	"order-playground/apps/gateway-api/internal/grpcclients"
	"order-playground/apps/gateway-api/internal/mongorepo"
	"order-playground/apps/gateway-api/internal/mysqlrepo"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const WorkflowName = "CreateOrderWorkflow"

type Starter struct {
	client    client.Client
	taskQueue string
}

func NewStarter(client client.Client, taskQueue string) *Starter {
	return &Starter{client: client, taskQueue: taskQueue}
}

func (s *Starter) StartCreateOrderWorkflow(ctx context.Context, input domain.WorkflowInput) (string, string, error) {
	workflowID := fmt.Sprintf("order-%s", input.OrderID)
	run, err := s.client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: s.taskQueue,
	}, WorkflowName, input)
	if err != nil {
		return "", "", err
	}
	return workflowID, run.GetRunID(), nil
}

type Activities struct {
	Orders    *mysqlrepo.Repo
	Audit     *mongorepo.Repo
	Inventory *grpcclients.InventoryClient
	Notifier  *grpcclients.NotifierClient
	Logger    *slog.Logger
}

func NewWorker(c client.Client, taskQueue string, activities *Activities) worker.Worker {
	w := worker.New(c, taskQueue, worker.Options{})
	w.RegisterWorkflowWithOptions(CreateOrderWorkflow, workflow.RegisterOptions{Name: WorkflowName})
	w.RegisterActivity(activities.ValidateOrderActivity)
	w.RegisterActivity(activities.WriteAuditEventActivity)
	w.RegisterActivity(activities.ReserveInventoryActivity)
	w.RegisterActivity(activities.ApprovePaymentActivity)
	w.RegisterActivity(activities.SendNotificationActivity)
	w.RegisterActivity(activities.IsOrderCanceledActivity)
	w.RegisterActivity(activities.MarkOrderCompletedActivity)
	w.RegisterActivity(activities.MarkWorkflowCanceledActivity)
	w.RegisterActivity(activities.MarkOrderFailedActivity)
	return w
}

func CreateOrderWorkflow(ctx workflow.Context, input domain.WorkflowInput) error {
	baseAO := workflow.ActivityOptions{
		StartToCloseTimeout: 20 * time.Second,
	}
	ctx = workflow.WithActivityOptions(ctx, baseAO)

	retryableAO := baseAO
	retryableAO.RetryPolicy = &temporal.RetryPolicy{
		InitialInterval:    1 * time.Second,
		BackoffCoefficient: 2,
		MaximumAttempts:    3,
	}

	if err := workflow.ExecuteActivity(ctx, "ValidateOrderActivity", input).Get(ctx, nil); err != nil {
		return markWorkflowFailed(ctx, input, err)
	}
	if err := workflow.ExecuteActivity(ctx, "WriteAuditEventActivity", domain.AuditEvent{
		OrderID:  input.OrderID,
		Type:     "ORDER_VALIDATED",
		Message:  "Order payload validated",
		Actor:    input.CreatedBy,
		Source:   "temporal",
		Metadata: map[string]any{"currency": input.Currency},
	}).Get(ctx, nil); err != nil {
		return err
	}

	retryCtx := workflow.WithActivityOptions(ctx, retryableAO)
	if err := workflow.ExecuteActivity(retryCtx, "ReserveInventoryActivity", input).Get(ctx, nil); err != nil {
		return markWorkflowFailed(ctx, input, err)
	}
	if err := workflow.ExecuteActivity(ctx, "WriteAuditEventActivity", domain.AuditEvent{
		OrderID: input.OrderID,
		Type:    "INVENTORY_RESERVED",
		Message: "Inventory reserved successfully",
		Actor:   input.CreatedBy,
		Source:  "inventory-service",
	}).Get(ctx, nil); err != nil {
		return err
	}

	if err := workflow.ExecuteActivity(ctx, "ApprovePaymentActivity", input).Get(ctx, nil); err != nil {
		return markWorkflowFailed(ctx, input, err)
	}
	if err := workflow.ExecuteActivity(ctx, "WriteAuditEventActivity", domain.AuditEvent{
		OrderID: input.OrderID,
		Type:    "PAYMENT_APPROVED",
		Message: "Payment approved by mock activity",
		Actor:   input.CreatedBy,
		Source:  "temporal",
	}).Get(ctx, nil); err != nil {
		return err
	}

	notifyCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 20 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2,
			MaximumAttempts:    2,
		},
	})
	if err := workflow.ExecuteActivity(notifyCtx, "SendNotificationActivity", input).Get(ctx, nil); err != nil {
		return markWorkflowFailed(ctx, input, err)
	}

	var canceled bool
	if err := workflow.ExecuteActivity(ctx, "IsOrderCanceledActivity", input.OrderID).Get(ctx, &canceled); err != nil {
		return err
	}
	if canceled {
		if err := workflow.ExecuteActivity(ctx, "MarkWorkflowCanceledActivity", input.OrderID).Get(ctx, nil); err != nil {
			return err
		}
		if err := workflow.ExecuteActivity(ctx, "WriteAuditEventActivity", domain.AuditEvent{
			OrderID: input.OrderID,
			Type:    "WORKFLOW_CANCELED",
			Message: "Workflow stopped because order was canceled",
			Actor:   input.CreatedBy,
			Source:  "temporal",
		}).Get(ctx, nil); err != nil {
			return err
		}
		return nil
	}

	if err := workflow.ExecuteActivity(ctx, "MarkOrderCompletedActivity", input.OrderID).Get(ctx, nil); err != nil {
		return err
	}
	if err := workflow.ExecuteActivity(ctx, "WriteAuditEventActivity", domain.AuditEvent{
		OrderID: input.OrderID,
		Type:    "ORDER_COMPLETED",
		Message: "Order workflow completed",
		Actor:   input.CreatedBy,
		Source:  "temporal",
	}).Get(ctx, nil); err != nil {
		return err
	}

	return nil
}

func markWorkflowFailed(ctx workflow.Context, input domain.WorkflowInput, cause error) error {
	_ = workflow.ExecuteActivity(ctx, "MarkOrderFailedActivity", input.OrderID).Get(ctx, nil)
	_ = workflow.ExecuteActivity(ctx, "WriteAuditEventActivity", domain.AuditEvent{
		OrderID:  input.OrderID,
		Type:     "ORDER_FAILED",
		Message:  cause.Error(),
		Actor:    input.CreatedBy,
		Source:   "temporal",
		Metadata: map[string]any{"error": cause.Error()},
	}).Get(ctx, nil)
	return cause
}

func (a *Activities) ValidateOrderActivity(ctx context.Context, input domain.WorkflowInput) error {
	a.Logger.InfoContext(ctx, "validating order", "order_id", input.OrderID)
	return domain.CreateOrderRequest{
		CustomerName: input.CustomerName,
		Currency:     input.Currency,
		Items:        input.Items,
	}.Validate()
}

func (a *Activities) WriteAuditEventActivity(ctx context.Context, event domain.AuditEvent) error {
	return a.Audit.WriteAuditEvent(ctx, event)
}

func (a *Activities) ReserveInventoryActivity(ctx context.Context, input domain.WorkflowInput) error {
	resp, err := a.Inventory.ReserveInventory(ctx, input.OrderID, input.Items)
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("%s", resp.Message)
	}
	return nil
}

func (a *Activities) ApprovePaymentActivity(ctx context.Context, input domain.WorkflowInput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("payment approved", "order_id", input.OrderID)
	return nil
}

func (a *Activities) SendNotificationActivity(ctx context.Context, input domain.WorkflowInput) error {
	resp, err := a.Notifier.SendOrderNotification(ctx, input.OrderID, "ORDER_COMPLETED", input.CreatedBy)
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("notification was not accepted")
	}
	return nil
}

func (a *Activities) IsOrderCanceledActivity(ctx context.Context, orderID string) (bool, error) {
	status, err := a.Orders.GetOrderStatus(ctx, orderID)
	if err != nil {
		return false, err
	}
	return status == domain.OrderStatusCanceled, nil
}

func (a *Activities) MarkOrderCompletedActivity(ctx context.Context, orderID string) error {
	if err := a.Orders.UpdateOrderStatus(ctx, orderID, domain.OrderStatusCompleted); err != nil {
		return err
	}
	return a.Orders.UpdateWorkflowRun(ctx, fmt.Sprintf("order-%s", orderID), "", domain.WorkflowStatusCompleted)
}

func (a *Activities) MarkWorkflowCanceledActivity(ctx context.Context, orderID string) error {
	return a.Orders.UpdateWorkflowRun(ctx, fmt.Sprintf("order-%s", orderID), "", domain.WorkflowStatusCanceled)
}

func (a *Activities) MarkOrderFailedActivity(ctx context.Context, orderID string) error {
	if err := a.Orders.UpdateOrderStatus(ctx, orderID, domain.OrderStatusFailed); err != nil {
		return err
	}
	return a.Orders.UpdateWorkflowRun(ctx, fmt.Sprintf("order-%s", orderID), "", domain.WorkflowStatusFailed)
}
