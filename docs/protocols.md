# Auth và Protocol Notes

## OIDC, OAuth2, SSO, SAML2

- `OAuth2`: framework cấp quyền và access token.
- `OIDC`: lớp xác thực nằm trên OAuth2; dùng để biết user là ai.
- `SSO`: trải nghiệm đăng nhập một lần cho nhiều ứng dụng.
- `SAML2`: chuẩn federation/auth enterprise truyền thống, không implement trong playground này.

Trong project này:
- Keycloak là Identity Provider.
- Gateway verify access token JWT từ Keycloak.
- Routes `/api/*` yêu cầu token hợp lệ.
- Routes `/api/admin/*` còn yêu cầu role `admin`.

## HTTP/1.1, HTTP/2, gRPC

- Client gọi Gateway bằng JSON/HTTP.
- Gateway expose REST-style endpoints bằng Gin.
- Inventory và Notifier dùng gRPC, chạy trên transport HTTP/2.
- Temporal không phải protocol giao tiếp end-user; nó là workflow orchestration layer.

## Những thứ nên tự thử

1. Gọi `/api/orders` không có token để xem `401`.
2. Login bằng `user1`, gọi `/api/admin/orders` để xem `403`.
3. Tạo order có quantity `6` để inventory reject và workflow fail.
4. Theo dõi trạng thái order trong MySQL và audit trail trong MongoDB.
5. Mở Temporal UI để xem workflow run và retry behavior.

