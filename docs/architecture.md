# Architecture Notes

## Flow chính

1. Client login qua Keycloak, nhận access token.
2. Client gọi `POST /api/orders`.
3. Gateway verify JWT bằng JWKS từ Keycloak.
4. Gateway ghi order `PENDING` vào MySQL.
5. Gateway ghi `ORDER_CREATED` vào MongoDB.
6. Gateway start `CreateOrderWorkflow` trong Temporal.
7. Workflow:
   - validate input
   - reserve inventory qua gRPC
   - approve payment mock
   - send notification qua gRPC
   - update MySQL
   - ghi audit events vào MongoDB

## Mapping khái niệm

- `OIDC`: layer xác thực dựa trên OAuth2; Keycloak dùng để login và cấp token.
- `OAuth2`: framework cấp quyền và token.
- `SSO`: trải nghiệm đăng nhập một lần; được tạo bởi Keycloak.
- `SAML2`: không implement trong playground này, chỉ được nhắc tới trong tài liệu học.
- `HTTP/1.1`: public API JSON từ Gin.
- `HTTP/2`: transport nền cho gRPC.
- `gRPC`: internal communication giữa gateway và các internal services.

## Connection pool

### MySQL

- `MaxOpenConns=20`
- `MaxIdleConns=10`
- `ConnMaxLifetime=30m`
- `ConnMaxIdleTime=5m`

### MongoDB

- `maxPoolSize=20`
- `minPoolSize=5`
- `maxIdleTimeMS=300000`

Các giá trị này không phải best default cho production, nhưng đủ để demo pool, idle connection và cách tuning trong app Go.

