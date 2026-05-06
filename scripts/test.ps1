param(
  [string]$GatewayBaseUrl = "http://localhost:8080",
  [string]$KeycloakBaseUrl = "http://localhost:8081",
  [string]$Realm = "order-playground",
  [string]$ClientId = "gateway-api",
  [string]$Username = "user1",
  [string]$Password = "user1pass",
  [int]$TimeoutSeconds = 45
)

$root = Split-Path -Parent $PSScriptRoot
$testScript = Join-Path $root "tests\e2e\order-scenarios.ps1"

& $testScript `
  -GatewayBaseUrl $GatewayBaseUrl `
  -KeycloakBaseUrl $KeycloakBaseUrl `
  -Realm $Realm `
  -ClientId $ClientId `
  -Username $Username `
  -Password $Password `
  -TimeoutSeconds $TimeoutSeconds
