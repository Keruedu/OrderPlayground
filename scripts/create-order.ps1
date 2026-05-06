param(
  [Parameter(Mandatory = $true)]
  [string]$Token
)

$body = @{
  customer_name = "Nguyen Trung"
  currency = "USD"
  items = @(
    @{ sku = "BOOK-001"; quantity = 1; price = 15.50 },
    @{ sku = "PEN-002"; quantity = 2; price = 4.25 }
  )
} | ConvertTo-Json -Depth 5

Invoke-RestMethod `
  -Method Post `
  -Uri "http://localhost:8080/api/orders" `
  -Headers @{ Authorization = "Bearer $Token" } `
  -ContentType "application/json" `
  -Body $body

