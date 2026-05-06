param(
  [Parameter(Mandatory = $true)]
  [string]$Token,
  [Parameter(Mandatory = $true)]
  [string]$OrderId
)

Invoke-RestMethod `
  -Method Get `
  -Uri "http://localhost:8080/api/orders/$OrderId/events" `
  -Headers @{ Authorization = "Bearer $Token" }
