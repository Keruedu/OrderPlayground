param(
  [string]$GatewayBaseUrl = "http://localhost:8080",
  [string]$KeycloakBaseUrl = "http://localhost:8081",
  [string]$Realm = "order-playground",
  [string]$ClientId = "gateway-api",
  [string]$Username = "user1",
  [string]$Password = "user1pass",
  [int]$TimeoutSeconds = 45
)

$ErrorActionPreference = "Stop"

function Get-AccessToken {
  param(
    [string]$KeycloakBaseUrl,
    [string]$Realm,
    [string]$ClientId,
    [string]$Username,
    [string]$Password
  )

  $response = Invoke-RestMethod `
    -Method Post `
    -Uri "$KeycloakBaseUrl/realms/$Realm/protocol/openid-connect/token" `
    -ContentType "application/x-www-form-urlencoded" `
    -Body @{
      client_id  = $ClientId
      grant_type = "password"
      username   = $Username
      password   = $Password
    }

  return $response.access_token
}

function Wait-OrderStatus {
  param(
    [string]$GatewayBaseUrl,
    [string]$Token,
    [string]$OrderId,
    [string]$ExpectedStatus,
    [int]$TimeoutSeconds
  )

  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)

  do {
    $order = Invoke-RestMethod `
      -Method Get `
      -Uri "$GatewayBaseUrl/api/orders/$OrderId" `
      -Headers @{ Authorization = "Bearer $Token" }

    if ($order.status -eq $ExpectedStatus) {
      return $order
    }

    Start-Sleep -Seconds 2
  } while ((Get-Date) -lt $deadline)

  throw "Order $OrderId did not reach $ExpectedStatus within $TimeoutSeconds seconds. Last status: $($order.status)"
}

function Assert-StatusCode {
  param(
    [scriptblock]$Request,
    [int]$ExpectedStatusCode
  )

  try {
    & $Request | Out-Null
    throw "Expected HTTP $ExpectedStatusCode but request succeeded."
  } catch {
    $actualStatusCode = [int]$_.Exception.Response.StatusCode
    if ($actualStatusCode -ne $ExpectedStatusCode) {
      throw "Expected HTTP $ExpectedStatusCode but got HTTP $actualStatusCode."
    }
    return $actualStatusCode
  }
}

Write-Host "Checking gateway health..."
$health = Invoke-RestMethod "$GatewayBaseUrl/healthz"
if ($health.status -ne "ok") {
  throw "Gateway health check failed. Expected status=ok, got status=$($health.status)"
}

Write-Host "Getting access token for $Username..."
$userToken = Get-AccessToken `
  -KeycloakBaseUrl $KeycloakBaseUrl `
  -Realm $Realm `
  -ClientId $ClientId `
  -Username $Username `
  -Password $Password

Write-Host "Running happy path: quantity <= 5 should become COMPLETED..."
$happyBody = @{
  customer_name = "Happy Path User"
  currency      = "USD"
  items         = @(
    @{ sku = "BOOK-001"; quantity = 2; price = 15.5 }
  )
} | ConvertTo-Json -Depth 5

$happyCreate = Invoke-RestMethod `
  -Method Post `
  -Uri "$GatewayBaseUrl/api/orders" `
  -Headers @{ Authorization = "Bearer $userToken" } `
  -ContentType "application/json" `
  -Body $happyBody

$happyOrder = Wait-OrderStatus `
  -GatewayBaseUrl $GatewayBaseUrl `
  -Token $userToken `
  -OrderId $happyCreate.order_id `
  -ExpectedStatus "COMPLETED" `
  -TimeoutSeconds $TimeoutSeconds

Write-Host "Running failure path: quantity > 5 should become FAILED..."
$failureBody = @{
  customer_name = "Failure Path User"
  currency      = "USD"
  items         = @(
    @{ sku = "BULK-999"; quantity = 9; price = 99.0 }
  )
} | ConvertTo-Json -Depth 5

$failureCreate = Invoke-RestMethod `
  -Method Post `
  -Uri "$GatewayBaseUrl/api/orders" `
  -Headers @{ Authorization = "Bearer $userToken" } `
  -ContentType "application/json" `
  -Body $failureBody

$failureOrder = Wait-OrderStatus `
  -GatewayBaseUrl $GatewayBaseUrl `
  -Token $userToken `
  -OrderId $failureCreate.order_id `
  -ExpectedStatus "FAILED" `
  -TimeoutSeconds $TimeoutSeconds

Write-Host "Running auth path: POST /api/orders without token should return 401..."
$noTokenStatus = Assert-StatusCode `
  -ExpectedStatusCode 401 `
  -Request {
    Invoke-RestMethod `
      -Method Post `
      -Uri "$GatewayBaseUrl/api/orders" `
      -ContentType "application/json" `
      -Body $happyBody
  }

Write-Host "Running auth path: user role calling admin route should return 403..."
$userAdminStatus = Assert-StatusCode `
  -ExpectedStatusCode 403 `
  -Request {
    Invoke-RestMethod `
      -Method Get `
      -Uri "$GatewayBaseUrl/api/admin/orders" `
      -Headers @{ Authorization = "Bearer $userToken" }
  }

[ordered]@{
  gateway = "ok"
  happy_path = [ordered]@{
    order_id = $happyOrder.id
    expected = "COMPLETED"
    actual = $happyOrder.status
  }
  failure_path = [ordered]@{
    order_id = $failureOrder.id
    expected = "FAILED"
    actual = $failureOrder.status
  }
  auth_path_no_token = [ordered]@{
    expected = 401
    actual = $noTokenStatus
  }
  auth_path_user_admin = [ordered]@{
    expected = 403
    actual = $userAdminStatus
  }
} | ConvertTo-Json -Depth 10
