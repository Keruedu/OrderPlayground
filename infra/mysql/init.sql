CREATE TABLE IF NOT EXISTS orders (
  id VARCHAR(36) PRIMARY KEY,
  customer_name VARCHAR(255) NOT NULL,
  currency VARCHAR(12) NOT NULL,
  status VARCHAR(32) NOT NULL,
  created_by VARCHAR(255) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS order_items (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  order_id VARCHAR(36) NOT NULL,
  sku VARCHAR(255) NOT NULL,
  quantity INT NOT NULL,
  price DECIMAL(12,2) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_order_items_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS workflow_runs (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  order_id VARCHAR(36) NOT NULL,
  workflow_id VARCHAR(255) NOT NULL,
  run_id VARCHAR(255) NOT NULL DEFAULT '',
  status VARCHAR(32) NOT NULL,
  started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE INDEX idx_orders_created_by_created_at ON orders(created_by, created_at);
CREATE INDEX idx_order_items_order_id ON order_items(order_id);
CREATE INDEX idx_workflow_runs_order_id ON workflow_runs(order_id);
CREATE INDEX idx_workflow_runs_workflow_id ON workflow_runs(workflow_id);

