# ==============================================================================
# Trestle — Unified One-Command Setup & Runner (PowerShell / Windows)
# ==============================================================================
# Supports: Windows (PowerShell 5.1 / PowerShell 7+), macOS, Linux
# ==============================================================================

[CmdletBinding()]
param (
    [Parameter(Position = 0)]
    [string]$Command = "",
    [switch]$Interactive
)

$ErrorActionPreference = "Stop"

# Ensure we are in the repository root
if (-not (Test-Path "go.mod") -or -not (Test-Path "web")) {
    Write-Host "==> Trestle repository not detected in current directory." -ForegroundColor Yellow
    if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
        Write-Error "[ERROR] 'git' is required to clone the repository."
        exit 1
    }
    Write-Host "==> Cloning https://github.com/maruf-pfc/trestle.git..." -ForegroundColor Cyan
    git clone https://github.com/maruf-pfc/trestle.git trestle
    Set-Location trestle
    & .\run.ps1 @PSBoundParameters
    exit $LASTEXITCODE
}

$RepoRoot = $PSScriptRoot
if ($RepoRoot) { Set-Location $RepoRoot }

# --- Helper Functions ---

function Test-PortInUse([string]$HostAddress, [int]$Port) {
    try {
        $tcpClient = New-Object System.Net.Sockets.TcpClient
        $connect = $tcpClient.BeginConnect($HostAddress, $Port, $null, $null)
        $wait = $connect.AsyncWaitHandle.WaitOne(800, $false)
        if ($wait) {
            $tcpClient.EndConnect($connect)
            $tcpClient.Close()
            return $true
        }
        $tcpClient.Close()
        return $false
    }
    catch {
        return $false
    }
}

function Find-FreePort([string]$HostAddress, [int]$StartPort) {
    $port = $StartPort
    while ($port -le 65535) {
        if (-not (Test-PortInUse -HostAddress $HostAddress -Port $port)) {
            return $port
        }
        $port++
    }
    return $StartPort
}

function Get-ComposeCommand {
    if (Get-Command docker -ErrorAction SilentlyContinue) {
        $check = docker compose version 2>$null
        if ($LASTEXITCODE -eq 0) { return "docker compose" }
    }
    if (Get-Command podman -ErrorAction SilentlyContinue) {
        $check = podman compose version 2>$null
        if ($LASTEXITCODE -eq 0) { return "podman compose" }
    }
    return $null
}

$ComposeCmd = Get-ComposeCommand

# --- Command Routing ---

switch ($Command.ToLower()) {
    "stop" {
        Write-Host "Stopping Trestle services..." -ForegroundColor Cyan
        if ($ComposeCmd) {
            Invoke-Expression "$ComposeCmd stop db"
            Write-Host "✓ Database container stopped." -ForegroundColor Green
        }
        exit 0
    }
    "db" {
        if (-not $ComposeCmd) {
            Write-Error "[ERROR] Docker or Podman Compose is required to run the database."
            exit 1
        }
        Invoke-Expression "$ComposeCmd up -d db"
        Write-Host "✓ Database service started." -ForegroundColor Green
        exit 0
    }
    "migrate" {
        if (Test-Path ".env") { Get-Content ".env" | ForEach-Object { if ($_ -match '^([^#=]+)=(.*)$') { [System.Environment]::SetEnvironmentVariable($matches[1].Trim(), $matches[2].Trim()) } } }
        go run . migrate
        exit $LASTEXITCODE
    }
    "bootstrap" {
        if (Test-Path ".env") { Get-Content ".env" | ForEach-Object { if ($_ -match '^([^#=]+)=(.*)$') { [System.Environment]::SetEnvironmentVariable($matches[1].Trim(), $matches[2].Trim()) } } }
        go run . bootstrap
        exit $LASTEXITCODE
    }
    "help" {
        Write-Host "Usage: .\run.ps1 [command] [-Interactive]"
        Write-Host ""
        Write-Host "Commands:"
        Write-Host "  (none)        Setup environment, database, frontend, migrate, and start Trestle"
        Write-Host "  stop          Stop the background database container"
        Write-Host "  db            Start only the PostgreSQL database container"
        Write-Host "  migrate       Run database migrations"
        Write-Host "  bootstrap     Run operator account bootstrap"
        Write-Host "  help          Display this help message"
        Write-Host ""
        Write-Host "Options:"
        Write-Host "  -Interactive  Prompt for custom operator credentials during bootstrap"
        exit 0
    }
}

# ==============================================================================
# MAIN SETUP & RUN FLOW
# ==============================================================================

Write-Host ""
Write-Host "==================================================================" -ForegroundColor Cyan
Write-Host "                 TRESTLE — ONE-COMMAND SETUP                      " -ForegroundColor Cyan
Write-Host "==================================================================" -ForegroundColor Cyan
Write-Host ""

# 1. Check Prerequisites
Write-Host "[1/6] Checking system tools and prerequisites..."
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Error "[ERROR] Go toolchain (go) is not installed. Visit https://go.dev/dl/"
    exit 1
}
$GoVer = (go env GOVERSION)
Write-Host "  ✓ Go toolchain found ($GoVer)" -ForegroundColor Green

if (-not (Get-Command node -ErrorAction SilentlyContinue) -or -not (Get-Command npm -ErrorAction SilentlyContinue)) {
    Write-Error "[ERROR] Node.js and npm are required. Visit https://nodejs.org"
    exit 1
}
$NodeVer = (node --version)
Write-Host "  ✓ Node.js & npm found ($NodeVer)" -ForegroundColor Green

if ($ComposeCmd) {
    Write-Host "  ✓ Container runtime found ($ComposeCmd)" -ForegroundColor Green
} else {
    Write-Host "  ! Docker/Podman Compose not detected. An existing local PostgreSQL database must be configured in .env." -ForegroundColor Yellow
}

# 2. Port Collision Detection & Environment
Write-Host ""
Write-Host "[2/6] Resolving ports and configuring environment..."

$EnvSettings = @{}
if (Test-Path ".env") {
    Get-Content ".env" | ForEach-Object {
        if ($_ -match '^\s*([^#=]+)=(.*)$') {
            $key = $matches[1].Trim()
            $val = $matches[2].Trim()
            $EnvSettings[$key] = $val
            [System.Environment]::SetEnvironmentVariable($key, $val)
        }
    }
}

# Database Port
$TargetPgPort = if ($EnvSettings.ContainsKey("POSTGRES_PORT")) { [int]$EnvSettings["POSTGRES_PORT"] } else { 5432 }
if (Test-PortInUse -HostAddress "127.0.0.1" -Port $TargetPgPort) {
    # Check if our own container is running
    $OurDbRunning = $false
    if (Get-Command docker -ErrorAction SilentlyContinue) {
        $status = (docker inspect trestle-db-1 --format '{{.State.Status}}' 2>$null)
        if ($status -eq "running") { $OurDbRunning = $true }
    }
    if ($OurDbRunning) {
        $ResolvedPgPort = $TargetPgPort
        Write-Host "  ✓ Reusing running Trestle database on 127.0.0.1:$ResolvedPgPort" -ForegroundColor Green
    } else {
        $ResolvedPgPort = Find-FreePort -HostAddress "127.0.0.1" -StartPort 5433
        Write-Host "  ! Port $TargetPgPort is in use by another service/database." -ForegroundColor Yellow
        Write-Host "  ➜ Auto-selected free port $ResolvedPgPort for Trestle PostgreSQL." -ForegroundColor Cyan
    }
} else {
    $ResolvedPgPort = $TargetPgPort
    Write-Host "  ✓ Database port 127.0.0.1:$ResolvedPgPort is available" -ForegroundColor Green
}

# Web Server Port
$TargetWebPort = 3000
if ($EnvSettings.ContainsKey("TRESTLE_BIND_ADDR")) {
    $parts = $EnvSettings["TRESTLE_BIND_ADDR"].Split(":")
    if ($parts.Length -gt 1) { $TargetWebPort = [int]$parts[-1] }
}

if (Test-PortInUse -HostAddress "127.0.0.1" -Port $TargetWebPort) {
    $ResolvedWebPort = Find-FreePort -HostAddress "127.0.0.1" -StartPort ($TargetWebPort + 1)
    Write-Host "  ! Port $TargetWebPort is in use by another application." -ForegroundColor Yellow
    Write-Host "  ➜ Auto-selected free port $ResolvedWebPort for Trestle web server." -ForegroundColor Cyan
} else {
    $ResolvedWebPort = $TargetWebPort
    Write-Host "  ✓ Web application port 127.0.0.1:$ResolvedWebPort is available" -ForegroundColor Green
}

# Database Password
$PgPassword = if ($EnvSettings.ContainsKey("POSTGRES_PASSWORD")) { $EnvSettings["POSTGRES_PASSWORD"] } else {
    $bytes = New-Object byte[] 24
    [System.Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
    [System.BitConverter]::ToString($bytes).Replace("-", "").ToLower()
}

$BindAddr = "127.0.0.1:$ResolvedWebPort"
$DbUrl = "postgres://trestle:${PgPassword}@127.0.0.1:${ResolvedPgPort}/trestle"

# Write .env
@"
# Trestle local configuration
# Generated automatically by run.ps1 — preserves custom values when present.
POSTGRES_PASSWORD=$PgPassword
POSTGRES_PORT=$ResolvedPgPort
DATABASE_URL=$DbUrl
TRESTLE_BIND_ADDR=$BindAddr
TRESTLE_DEV_MODE=true
TRESTLE_MAX_DB_CONNECTIONS=5
TRESTLE_DB_TIMEOUT_MS=2000
"@ | Out-File -FilePath ".env" -Encoding utf8

[System.Environment]::SetEnvironmentVariable("POSTGRES_PASSWORD", $PgPassword)
[System.Environment]::SetEnvironmentVariable("POSTGRES_PORT", "$ResolvedPgPort")
[System.Environment]::SetEnvironmentVariable("DATABASE_URL", $DbUrl)
[System.Environment]::SetEnvironmentVariable("TRESTLE_BIND_ADDR", $BindAddr)
[System.Environment]::SetEnvironmentVariable("TRESTLE_DEV_MODE", "true")
Write-Host "  ✓ Configuration saved to .env" -ForegroundColor Green

# 3. Start Database
Write-Host ""
Write-Host "[3/6] Starting PostgreSQL database..."
if ($ComposeCmd) {
    Invoke-Expression "$ComposeCmd up -d db"
    Write-Host "  ➜ Waiting for PostgreSQL on port $ResolvedPgPort to be ready..." -NoNewline
    $ready = $false
    for ($i = 0; $i -lt 30; $i++) {
        $check = Invoke-Expression "$ComposeCmd exec -T db pg_isready -U trestle -d trestle 2>&1"
        if ($LASTEXITCODE -eq 0) { $ready = $true; break }
        Start-Sleep -Seconds 1
        Write-Host "." -NoNewline
    }
    Write-Host ""
    if (-not $ready) {
        Write-Error "[ERROR] PostgreSQL did not become ready within 30 seconds."
        exit 1
    }
    Write-Host "  ✓ PostgreSQL is healthy and accepting connections." -ForegroundColor Green
} else {
    if (-not (Test-PortInUse -HostAddress "127.0.0.1" -Port $ResolvedPgPort)) {
        Write-Error "[ERROR] No database is running at 127.0.0.1:$ResolvedPgPort."
        exit 1
    }
    Write-Host "  ✓ Local PostgreSQL detected on 127.0.0.1:$ResolvedPgPort." -ForegroundColor Green
}

# 4. Frontend Build
Write-Host ""
Write-Host "[4/6] Verifying frontend assets..."
if (-not (Test-Path "web/node_modules")) {
    Write-Host "  ➜ Installing frontend dependencies..." -ForegroundColor Cyan
    npm ci --prefix web --ignore-scripts
}
if (-not (Test-Path "web/dist") -or -not (Test-Path "web/dist/index.html")) {
    Write-Host "  ➜ Building frontend production bundle..." -ForegroundColor Cyan
    npm run build --prefix web
}
Write-Host "  ✓ Frontend assets ready in web/dist." -ForegroundColor Green

# 5. Database Migrations
Write-Host ""
Write-Host "[5/6] Applying database migrations..."
go run . migrate
if ($LASTEXITCODE -ne 0) {
    Write-Error "[ERROR] Database migrations failed."
    exit $LASTEXITCODE
}
Write-Host "  ✓ Database schema is up to date." -ForegroundColor Green

# 6. Instance Bootstrap
Write-Host ""
Write-Host "[6/6] Checking instance bootstrap..."
if (-not $Interactive) {
    [System.Environment]::SetEnvironmentVariable("TRESTLE_BOOTSTRAP_USERNAME", "admin")
    [System.Environment]::SetEnvironmentVariable("TRESTLE_BOOTSTRAP_PASSWORD", "adminpassword123")
    [System.Environment]::SetEnvironmentVariable("TRESTLE_BOOTSTRAP_WORKSPACE_NAME", "Main Workspace")
    [System.Environment]::SetEnvironmentVariable("TRESTLE_BOOTSTRAP_WORKSPACE_SLUG", "main")
}

go run . bootstrap
# Exit code 0 on fresh bootstrap or "already initialized"

# Launch Server
Write-Host ""
Write-Host "==================================================================" -ForegroundColor Cyan
Write-Host "  🚀 TRESTLE IS READY AND RUNNING!" -ForegroundColor Green
Write-Host "==================================================================" -ForegroundColor Cyan
Write-Host "  Web Application:  http://$BindAddr" -ForegroundColor Cyan
Write-Host "  Database:         127.0.0.1:$ResolvedPgPort (db: trestle)" -ForegroundColor Cyan
if (-not $Interactive) {
    Write-Host "  Default Admin:    username: admin / password: adminpassword123" -ForegroundColor Cyan
}
Write-Host "==================================================================" -ForegroundColor Cyan
Write-Host "Press Ctrl+C to stop the application server."
Write-Host "(To stop the background database container later, run: .\run.ps1 stop)"
Write-Host ""

go run .
