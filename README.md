# Order Playground

Backend-only playground để học `Go + Gin`, `MySQL`, `MongoDB`, `Temporal`, `Keycloak`, `HTTP/JSON`, `HTTP/2`, và `gRPC` thông qua một flow xử lý order end-to-end.

## Thành phần

- `apps/gateway-api`: public HTTP API, auth JWT từ Keycloak, truy cập MySQL/MongoDB, start Temporal workflow.
- `apps/inventory-service`: gRPC service mô phỏng reserve inventory.
- `apps/notifier-service`: gRPC service mô phỏng gửi notification.
- `mysql`: lưu order, items, workflow runs.
- `mongodb`: lưu audit events và request logs.
- `keycloak`: cấp token OIDC/OAuth2.
- `temporal` + `temporal-ui`: workflow engine và giao diện xem workflow.

## Cấu trúc

- `apps/`
- `pkg/proto`
- `pkg/transport`
- `infra/docker`
- `infra/keycloak`
- `infra/mysql`
- `docs`
- `scripts`

## Chạy nhanh

1. Copy `.env.example` thành `.env`
2. Chạy:

```powershell
docker compose -f infra/docker/docker-compose.yml --env-file .env up --build
```

3. Mở các địa chỉ:

- Gateway API: [http://localhost:8080/version](http://localhost:8080/version)
- Keycloak: [http://localhost:8081](http://localhost:8081)
- Temporal UI: [http://localhost:8088](http://localhost:8088)

## Demo flow

1. Lấy token:

```powershell
.\scripts\get-token.ps1 -Username user1 -Password user1pass
```

2. Tạo order:

```powershell
.\scripts\create-order.ps1 -Token "<access-token>"
```

3. Đọc order:

```powershell
.\scripts\get-order.ps1 -Token "<access-token>" -OrderId "<order-id>"
```

4. Xem events:

```powershell
.\scripts\get-events.ps1 -Token "<access-token>" -OrderId "<order-id>"
```

5. Với admin:

```powershell
.\scripts\admin-list-orders.ps1 -Token "<admin-access-token>"
.\scripts\admin-get-workflow.ps1 -Token "<admin-access-token>" -WorkflowId "<workflow-id>"
```

## Keycloak dev users

- `user1 / user1pass` với role `user`
- `admin1 / admin1pass` với role `admin`

## Ghi chú

- Repo có giữ `.proto` để học contract, nhưng không phụ thuộc `protoc` local. Bindings gRPC nội bộ được viết sẵn để build ngay trên máy Windows.
- Temporal worker chạy chung process với gateway-api trong v1 để giảm độ phức tạp.
- `docs/architecture.md` giải thích mapping khái niệm giữa auth, protocol và workflow.
- `docs/protocols.md` tóm tắt OIDC/OAuth2/SAML2/SSO và HTTP/gRPC trong playground.
