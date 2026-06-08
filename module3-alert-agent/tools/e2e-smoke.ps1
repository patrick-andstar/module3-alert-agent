param(
  [string]$BaseUrl = "http://127.0.0.1:9090",
  [switch]$UseRunningServer,
  [switch]$SkipSchema,
  [switch]$PreflightOnly
)

$ErrorActionPreference = "Stop"

function Require-Env {
  param([string]$Name, [switch]$Secret)
  $value = [Environment]::GetEnvironmentVariable($Name, "Process")
  if ([string]::IsNullOrWhiteSpace($value)) {
    throw "Missing required environment variable: $Name"
  }
  if ($Secret) {
    Write-Host "$Name=<set>"
  } else {
    Write-Host "$Name=$value"
  }
  return $value
}

function Find-MySqlClient {
  $cmd = Get-Command mysql.exe -ErrorAction SilentlyContinue
  if ($cmd) {
    return $cmd.Source
  }
  $mysqlHome = [Environment]::GetEnvironmentVariable("MYSQL_HOME", "Process")
  if (-not [string]::IsNullOrWhiteSpace($mysqlHome)) {
    $candidate = Join-Path $mysqlHome "bin\mysql.exe"
    if (Test-Path $candidate) {
      return $candidate
    }
  }
  throw "mysql.exe not found in PATH or MYSQL_HOME\bin"
}

function Invoke-Json {
  param(
    [string]$Method,
    [string]$Path,
    [object]$Body = $null
  )
  $headers = @{}
  $token = [Environment]::GetEnvironmentVariable("ADMIN_API_TOKEN", "Process")
  if (-not [string]::IsNullOrWhiteSpace($token)) {
    $headers["Authorization"] = "Bearer $token"
  }
  $uri = "$BaseUrl$Path"
  if ($null -eq $Body) {
    return Invoke-RestMethod -Method $Method -Uri $uri -Headers $headers
  }
  $json = $Body | ConvertTo-Json -Depth 12
  return Invoke-RestMethod -Method $Method -Uri $uri -Headers $headers -ContentType "application/json" -Body $json
}

function Wait-Healthz {
  $deadline = (Get-Date).AddSeconds(45)
  do {
    try {
      $health = Invoke-RestMethod -Method GET -Uri "$BaseUrl/healthz"
      if ($health.status -eq "ok") {
        return
      }
    } catch {
      Start-Sleep -Milliseconds 500
    }
  } while ((Get-Date) -lt $deadline)
  throw "Service did not become healthy at $BaseUrl"
}

function Get-BaseUrlPort {
  $uri = [Uri]$BaseUrl
  if ($uri.Port -gt 0) {
    return $uri.Port
  }
  if ($uri.Scheme -eq "https") {
    return 443
  }
  return 80
}

function Get-ListeningPids {
  param([int]$Port)
  $lines = netstat -ano | Select-String ":$Port\s"
  $pids = @()
  foreach ($line in $lines) {
    $parts = ($line.ToString().Trim() -split "\s+")
    if ($parts.Length -ge 5 -and $parts[3] -eq "LISTENING") {
      $pids += [int]$parts[4]
    }
  }
  return $pids | Select-Object -Unique
}

function Assert-PortFree {
  param([int]$Port)
  $pids = @(Get-ListeningPids -Port $Port)
  if ($pids.Count -gt 0) {
    throw "Port $Port is already in use by process id(s): $($pids -join ', '). Stop the old service or pass -UseRunningServer."
  }
}

function Stop-Listeners {
  param([int]$Port)
  foreach ($processId in @(Get-ListeningPids -Port $Port)) {
    Stop-Process -Id $processId -Force -ErrorAction SilentlyContinue
  }
}

function Invoke-MySqlFile {
  param(
    [string]$Path,
    [string]$MySql,
    [string]$HostName,
    [string]$PortValue,
    [string]$UserName,
    [string]$DatabaseName,
    [string]$PasswordValue
  )
  $oldMysqlPwd = [Environment]::GetEnvironmentVariable("MYSQL_PWD", "Process")
  [Environment]::SetEnvironmentVariable("MYSQL_PWD", $PasswordValue, "Process")
  try {
    Get-Content -Raw -Path $Path | & $MySql --host=$HostName --port=$PortValue --user=$UserName --database=$DatabaseName
  } finally {
    [Environment]::SetEnvironmentVariable("MYSQL_PWD", $oldMysqlPwd, "Process")
  }
  if ($LASTEXITCODE -ne 0) {
    throw "Failed to apply $Path"
  }
  Write-Host "Applied $Path"
}

Push-Location (Split-Path $PSScriptRoot)
try {
  $mysqlHost = Require-Env "MYSQL_HOST"
  $mysqlPort = Require-Env "MYSQL_PORT"
  $mysqlUser = Require-Env "MYSQL_USER"
  $mysqlPassword = Require-Env "MYSQL_PASSWORD" -Secret
  $mysqlDatabase = Require-Env "MYSQL_DATABASE"
  Require-Env "ARK_CHAT_MODEL" | Out-Null
  Require-Env "ARK_API_KEY" -Secret | Out-Null

  $mysql = Find-MySqlClient
  Write-Host "mysql.exe=$mysql"

  if ($PreflightOnly) {
    Write-Host "Preflight complete."
    exit 0
  }

  if (-not $SkipSchema) {
    Invoke-MySqlFile -Path "sql/schema.sql" -MySql $mysql -HostName $mysqlHost -PortValue $mysqlPort -UserName $mysqlUser -DatabaseName $mysqlDatabase -PasswordValue $mysqlPassword
    Invoke-MySqlFile -Path "sql/upgrade.sql" -MySql $mysql -HostName $mysqlHost -PortValue $mysqlPort -UserName $mysqlUser -DatabaseName $mysqlDatabase -PasswordValue $mysqlPassword
  }

  $serverProcess = $null
  $serverPort = Get-BaseUrlPort
  if (-not $UseRunningServer) {
    Assert-PortFree -Port $serverPort
    $serverProcess = Start-Process -FilePath "go" -ArgumentList @("run", "./cmd/server") -WorkingDirectory (Get-Location).Path -PassThru -WindowStyle Hidden
  }

  try {
    Wait-Healthz

    $html = Invoke-WebRequest -Uri "$BaseUrl/" -UseBasicParsing
    if ($html.Content -notmatch "DLP Alert Agent Console") {
      throw "Frontend HTML did not contain the console title"
    }
    $app = Invoke-WebRequest -Uri "$BaseUrl/app.js" -UseBasicParsing
    foreach ($scenario in @("whitelist_drop", "dedup_merge", "seed_false_positive", "confirmed_false_positive", "uncertain_candidate", "empty_recall_agent_judgement", "true_alert")) {
      if ($app.Content -notmatch $scenario) {
        throw "Frontend app.js missing scenario: $scenario"
      }
    }

    $stamp = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
    $suffix = [Guid]::NewGuid().ToString("N").Substring(0, 8)
    $hostId = "host-e2e-$suffix"
    $userId = "user-e2e-$suffix"
    $processName = "e2e-browser-$suffix.exe"
    $processPath = "C:/DLP/E2E/$processName"
    $scenarioKey = "customer|upload|$processName|internal-crm.company.com"

    Invoke-Json -Method POST -Path "/api/whitelist" -Body @{
      rule_name = "e2e-backup-$suffix"
      logic = "OR"
      process_name = "backup-e2e.exe"
      enabled = $true
    } | Out-Null

    $dropResult = Invoke-Json -Method POST -Path "/api/client/events" -Body @{
      host_id = $hostId
      events = @(@{
        event_id = "evt-e2e-whitelist-$suffix"
        host_id = $hostId
        user_id = $userId
        file_path = "C:/e2e/customer.xlsx"
        file_hash = "hash-e2e-whitelist"
        sensitive = $true
        sensitive_type = "customer"
        risk_level = "high"
        process_name = "backup-e2e.exe"
        process_path = "C:/Backup/backup-e2e.exe"
        target = "D:/Backup/customer.xlsx"
        operation = "upload"
        timestamp = $stamp
        sensitive_file_id = "file-e2e-whitelist"
      })
    }
    if ($dropResult.dropped -lt 1) {
      throw "Whitelist scenario did not drop the event"
    }

    Invoke-Json -Method POST -Path "/api/false-positives" -Body @{
      scenario_key = $scenarioKey
      user_id = $userId
      sensitive_type = "customer"
      risk_level = "low"
      process_name = $processName
      process_path = $processPath
      target = "internal-crm.company.com"
      operation = "upload"
      reason = "e2e normal crm upload"
      hit_count = 1
    } | Out-Null

    $alertEventId = "evt-e2e-alert-$suffix"
    $alertResult = Invoke-Json -Method POST -Path "/api/client/events" -Body @{
      host_id = $hostId
      events = @(@{
        event_id = $alertEventId
        host_id = $hostId
        user_id = $userId
        file_path = "C:/Users/E2E/Desktop/customer.xlsx"
        file_hash = "hash-e2e-alert"
        sensitive = $true
        sensitive_type = "customer"
        risk_level = "high"
        process_name = $processName
        process_path = $processPath
        target = "internal-crm.company.com"
        operation = "upload"
        timestamp = $stamp
        sensitive_file_id = "file-e2e-alert"
      })
    }
    if ($alertResult.accepted -lt 1) {
      $alertResultJson = $alertResult | ConvertTo-Json -Depth 12
      throw "E2E alert was not accepted by /api/client/events: $alertResultJson"
    }

    $alerts = Invoke-Json -Method POST -Path "/api/alerts/query" -Body @{
      event_id = $alertEventId
      page = 1
      page_size = 20
      order_by = "timestamp"
      order = "desc"
    }
    if ($alerts.total -lt 1 -or $alerts.data.Count -lt 1) {
      $alertResultJson = $alertResult | ConvertTo-Json -Depth 12
      $alertsJson = $alerts | ConvertTo-Json -Depth 12
      throw "Alert query did not return the E2E alert. Ingest response: $alertResultJson Query response: $alertsJson"
    }
    if ([string]::IsNullOrWhiteSpace($alerts.data[0].agent_verdict)) {
      throw "E2E alert did not include agent_verdict"
    }
    if ($alerts.data[0].agent_explanation -match "agent analysis failed") {
      throw "Agent analysis failed during E2E smoke: $($alerts.data[0].agent_explanation)"
    }
    if ($alerts.data[0].agent_confidence -le 0) {
      throw "E2E alert did not include a positive agent_confidence"
    }

    $fps = Invoke-Json -Method GET -Path "/api/false-positives"
    if (@($fps | Where-Object { $_.scenario_key -eq $scenarioKey }).Count -lt 1) {
      throw "False-positive library did not include the seeded scenario"
    }

    $whitelist = Invoke-Json -Method GET -Path "/api/whitelist"
    if (@($whitelist | Where-Object { $_.rule_name -eq "e2e-backup-$suffix" }).Count -lt 1) {
      throw "Whitelist API did not include the E2E rule"
    }

    Write-Host "E2E smoke passed against $BaseUrl"
  } finally {
    if ($serverProcess -and -not $serverProcess.HasExited) {
      Stop-Process -Id $serverProcess.Id -Force
    }
    if (-not $UseRunningServer) {
      Stop-Listeners -Port $serverPort
    }
  }
} finally {
  Pop-Location
}
