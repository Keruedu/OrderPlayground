param(
  [Parameter(Mandatory = $true)]
  [string]$Token
)

Invoke-RestMethod `
  -Method Get `
  -Uri "http://localhost:8080/api/admin/orders" `
  -Headers @{ Authorization = "Bearer $Token" }

