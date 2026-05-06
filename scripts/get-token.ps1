param(
  [Parameter(Mandatory = $true)]
  [string]$Username,
  [Parameter(Mandatory = $true)]
  [string]$Password
)

$body = @{
  client_id = "gateway-api"
  grant_type = "password"
  username = $Username
  password = $Password
}

Invoke-RestMethod `
  -Method Post `
  -Uri "http://localhost:8081/realms/order-playground/protocol/openid-connect/token" `
  -ContentType "application/x-www-form-urlencoded" `
  -Body $body

