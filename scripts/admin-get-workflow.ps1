param(
  [Parameter(Mandatory = $true)]
  [string]$Token,
  [Parameter(Mandatory = $true)]
  [string]$WorkflowId
)

Invoke-RestMethod `
  -Method Get `
  -Uri "http://localhost:8080/api/admin/workflows/$WorkflowId" `
  -Headers @{ Authorization = "Bearer $Token" }

