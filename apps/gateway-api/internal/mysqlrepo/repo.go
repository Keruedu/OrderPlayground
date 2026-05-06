package mysqlrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"order-playground/apps/gateway-api/internal/domain"
)

type Repo struct {
	db *sql.DB
}

func New(db *sql.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) CreateOrder(ctx context.Context, order domain.Order) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO orders (id, customer_name, currency, status, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		order.ID, order.CustomerName, order.Currency, order.Status, order.CreatedBy, order.CreatedAt, order.UpdatedAt,
	)
	if err != nil {
		return err
	}

	for _, item := range order.Items {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO order_items (order_id, sku, quantity, price)
			VALUES (?, ?, ?, ?)`,
			order.ID, item.SKU, item.Quantity, item.Price,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *Repo) GetOrder(ctx context.Context, orderID string) (*domain.Order, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, customer_name, currency, status, created_by, created_at, updated_at
		FROM orders WHERE id = ?`, orderID)

	var order domain.Order
	if err := row.Scan(&order.ID, &order.CustomerName, &order.Currency, &order.Status, &order.CreatedBy, &order.CreatedAt, &order.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("order not found")
		}
		return nil, err
	}

	itemsRows, err := r.db.QueryContext(ctx, `
		SELECT id, order_id, sku, quantity, price FROM order_items WHERE order_id = ? ORDER BY id ASC`, orderID)
	if err != nil {
		return nil, err
	}
	defer itemsRows.Close()

	for itemsRows.Next() {
		var item domain.OrderItem
		if err := itemsRows.Scan(&item.ID, &item.OrderID, &item.SKU, &item.Quantity, &item.Price); err != nil {
			return nil, err
		}
		order.Items = append(order.Items, item)
	}

	workflow, err := r.GetWorkflowRunByOrderID(ctx, orderID)
	if err == nil {
		order.Workflow = workflow
	}

	return &order, nil
}

func (r *Repo) ListOrders(ctx context.Context) ([]domain.Order, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, customer_name, currency, status, created_by, created_at, updated_at
		FROM orders ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		var order domain.Order
		if err := rows.Scan(&order.ID, &order.CustomerName, &order.Currency, &order.Status, &order.CreatedBy, &order.CreatedAt, &order.UpdatedAt); err != nil {
			return nil, err
		}
		workflow, err := r.GetWorkflowRunByOrderID(ctx, order.ID)
		if err == nil {
			order.Workflow = workflow
		}
		orders = append(orders, order)
	}
	return orders, nil
}

func (r *Repo) UpdateOrderStatus(ctx context.Context, orderID, status string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE orders SET status = ?, updated_at = ? WHERE id = ?`, status, time.Now().UTC(), orderID)
	return err
}

func (r *Repo) GetOrderStatus(ctx context.Context, orderID string) (string, error) {
	row := r.db.QueryRowContext(ctx, `SELECT status FROM orders WHERE id = ?`, orderID)

	var status string
	if err := row.Scan(&status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("order not found")
		}
		return "", err
	}
	return status, nil
}

func (r *Repo) CreateWorkflowRun(ctx context.Context, workflow domain.WorkflowRun) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO workflow_runs (order_id, workflow_id, run_id, status, started_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		workflow.OrderID, workflow.WorkflowID, workflow.RunID, workflow.Status, workflow.StartedAt, workflow.UpdatedAt,
	)
	return err
}

func (r *Repo) UpdateWorkflowRun(ctx context.Context, workflowID, runID, status string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE workflow_runs SET run_id = CASE WHEN ? = '' THEN run_id ELSE ? END, status = ?, updated_at = ?
		WHERE workflow_id = ?`,
		runID, runID, status, time.Now().UTC(), workflowID,
	)
	return err
}

func (r *Repo) GetWorkflowRunByOrderID(ctx context.Context, orderID string) (*domain.WorkflowRun, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, order_id, workflow_id, run_id, status, started_at, updated_at
		FROM workflow_runs
		WHERE order_id = ?
		ORDER BY started_at DESC LIMIT 1`, orderID)
	var workflow domain.WorkflowRun
	if err := row.Scan(&workflow.ID, &workflow.OrderID, &workflow.WorkflowID, &workflow.RunID, &workflow.Status, &workflow.StartedAt, &workflow.UpdatedAt); err != nil {
		return nil, err
	}
	return &workflow, nil
}

func (r *Repo) GetWorkflowRunByWorkflowID(ctx context.Context, workflowID string) (*domain.WorkflowRun, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, order_id, workflow_id, run_id, status, started_at, updated_at
		FROM workflow_runs
		WHERE workflow_id = ?`, workflowID)
	var workflow domain.WorkflowRun
	if err := row.Scan(&workflow.ID, &workflow.OrderID, &workflow.WorkflowID, &workflow.RunID, &workflow.Status, &workflow.StartedAt, &workflow.UpdatedAt); err != nil {
		return nil, err
	}
	return &workflow, nil
}

func (r *Repo) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}
