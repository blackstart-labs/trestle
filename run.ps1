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
    if (-not (Test-Path "trestle/go.mod")) {
        Write-Host "==> Cloning https://github.com/blackstart-labs/trestle.git..." -ForegroundColor Cyan
        git clone https://github.com/blackstart-labs/trestle.git trestle
    }
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

# --- Dependency installation ---
# Missing or unsuitable Go/Node are installed into .tools (no admin rights); Docker via winget.
$OnWindows = $env:OS -eq "Windows_NT"
$ToolsDir = Join-Path (Get-Location) ".tools"
$PathSep = [System.IO.Path]::PathSeparator
if (Test-Path "$ToolsDir/go/bin") { $env:Path = "$ToolsDir/go/bin$PathSep$env:Path" }
Get-ChildItem $ToolsDir -Directory -Filter "node-*" -ErrorAction SilentlyContinue | ForEach-Object { $env:Path = "$($_.FullName)$PathSep$env:Path" }

function Get-Version([string]$Text) {
    try { return [version]($Text.Trim().TrimStart("v", "g", "o") -replace '[^0-9.].*$', '') } catch { return $null }
}

function Install-Zip([string]$Url, [string]$Name) {
    if (-not $OnWindows) { throw "Automatic install of $Name is supported on Windows only; on macOS/Linux use ./run.sh" }
    New-Item -ItemType Directory -Force $ToolsDir | Out-Null
    $zip = Join-Path $ToolsDir "$Name.zip"
    Write-Host "  ➜ Downloading $Url..." -ForegroundColor Cyan
    $ProgressPreference = "SilentlyContinue"
    Invoke-WebRequest -Uri $Url -OutFile $zip -UseBasicParsing
    Expand-Archive -Path $zip -DestinationPath $ToolsDir -Force
    Remove-Item $zip
}

$Arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }

# Go 1.21+ fetches the exact toolchain pinned in go.mod by itself; older or absent Go gets installed.
function Initialize-Go {
    if (Get-Command go -ErrorAction SilentlyContinue) {
        $v = Get-Version (go env GOVERSION)
        if ($v -and $v -ge [version]"1.21") { return }
    }
    $want = (Select-String -Path go.mod -Pattern '^toolchain go(.+)$').Matches.Groups[1].Value
    if (-not $want) { $want = (Select-String -Path go.mod -Pattern '^go (.+)$').Matches.Groups[1].Value }
    Remove-Item -Recurse -Force "$ToolsDir/go" -ErrorAction SilentlyContinue
    Install-Zip "https://go.dev/dl/go$want.windows-$Arch.zip" "go"
    $env:Path = "$ToolsDir/go/bin$PathSep$env:Path"
}

# web/.npmrc sets engine-strict, so Node and npm must match web/package.json engines exactly.
function Test-NodeOk {
    if (-not (Get-Command node -ErrorAction SilentlyContinue) -or -not (Get-Command npm -ErrorAction SilentlyContinue)) { return $false }
    $n = Get-Version (node --version); $m = Get-Version (npm --version)
    return ($n -and $m -and $n.Major -eq 24 -and $n -ge [version]"24.21.0" -and $m.Major -eq 11 -and $m -ge [version]"11.19.0")
}

function Initialize-Node {
    if (Test-NodeOk) { return }
    $want = (Get-Content .node-version -Raw).Trim()
    $nodeArch = if ($Arch -eq "arm64") { "arm64" } else { "x64" }
    $dir = "node-v$want-win-$nodeArch"
    Write-Host "  ➜ Installing Node.js $want into .tools (the frontend requires Node 24 and npm 11)..." -ForegroundColor Cyan
    if (-not (Test-Path "$ToolsDir/$dir")) { Install-Zip "https://nodejs.org/dist/v$want/$dir.zip" "node" }
    $env:Path = "$ToolsDir/$dir$PathSep$env:Path"
    if (-not (Test-NodeOk)) {
        $npm = (Get-Content web/package.json -Raw | ConvertFrom-Json).packageManager
        npm install -g $npm
    }
}

# Installs Docker Desktop when no engine exists and starts it when it is stopped.
function Initialize-Docker {
    if (-not (Get-ComposeCommand)) {
        # A local PostgreSQL configured in .env needs no container engine.
        $pgPort = 5432
        if (Test-Path ".env") {
            $line = Select-String -Path .env -Pattern '^POSTGRES_PORT=(\d+)' | Select-Object -First 1
            if ($line) { $pgPort = [int]$line.Matches.Groups[1].Value }
        }
        if (Test-PortInUse -HostAddress "127.0.0.1" -Port $pgPort) {
            Write-Host "  ! No container engine; using the PostgreSQL already listening on port $pgPort." -ForegroundColor Yellow
            return
        }
        if (-not (Get-Command winget -ErrorAction SilentlyContinue)) {
            throw "Docker is required. Install Docker Desktop: https://docs.docker.com/desktop/"
        }
        Write-Host "  ➜ Installing Docker Desktop (you may be asked to approve it)..." -ForegroundColor Cyan
        winget install -e --id Docker.DockerDesktop --accept-source-agreements --accept-package-agreements
        $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + $PathSep + [System.Environment]::GetEnvironmentVariable("Path", "User")
        if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
            throw "Docker Desktop was installed but needs a sign-out or restart (and WSL 2). Restart, then run this script again."
        }
    }
    if (-not (Get-Command docker -ErrorAction SilentlyContinue)) { return }
    docker info *> $null
    if ($LASTEXITCODE -eq 0) { return }
    # docker.exe lives in <Docker Desktop>\resources\bin, two levels below the app.
    $desktop = Join-Path (Split-Path (Split-Path (Split-Path (Get-Command docker).Source))) "Docker Desktop.exe"
    if ($OnWindows -and (Test-Path $desktop)) {
        Write-Host "  ➜ Starting Docker Desktop..." -ForegroundColor Cyan
        Start-Process $desktop
    }
    Write-Host "  ➜ Waiting for the Docker engine..." -NoNewline
    for ($i = 0; $i -lt 90; $i++) {
        docker info *> $null
        if ($LASTEXITCODE -eq 0) { Write-Host ""; return }
        Start-Sleep -Seconds 2
        Write-Host "." -NoNewline
    }
    Write-Host ""
    throw "The Docker engine is not running. Start Docker Desktop and run this script again."
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
Initialize-Go
Write-Host "  ✓ Go toolchain ready ($(go env GOVERSION))" -ForegroundColor Green

Initialize-Node
Write-Host "  ✓ Node.js $(node --version) & npm $(npm --version) ready" -ForegroundColor Green

Initialize-Docker
$ComposeCmd = Get-ComposeCommand
if ($ComposeCmd) { Write-Host "  ✓ Container runtime ready ($ComposeCmd)" -ForegroundColor Green }

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
    if ($ComposeCmd) {
        $running = Invoke-Expression "$ComposeCmd ps -q --status running db 2>`$null"
        if ($running) { $OurDbRunning = $true }
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
        $check = Invoke-Expression "$ComposeCmd exec -T db pg_isready -h 127.0.0.1 -U trestle -d trestle 2>&1"
        if ($LASTEXITCODE -eq 0) { $ready = $true; break }
        Start-Sleep -Seconds 1
        Write-Host "." -NoNewline
    }
    Write-Host ""
    if (-not $ready) {
        Write-Error "[ERROR] PostgreSQL did not become ready within 30 seconds."
        exit 1
    }
    # Postgres reads POSTGRES_PASSWORD only when the volume is first created; keep it in sync with .env.
    $compose = $ComposeCmd -split " "
    "ALTER USER trestle WITH PASSWORD '$PgPassword';" | & $compose[0] $compose[1..9] exec -T db psql -q -U trestle -d trestle | Out-Null
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
$dependenciesChanged = $false
$dependencyFiles = @("package.json", "package-lock.json")
$dependenciesMatch = $true
foreach ($file in $dependencyFiles) {
    $installed = "web/node_modules/.trestle-$file"
    if (-not (Test-Path $installed) -or (Get-FileHash "web/$file").Hash -ne (Get-FileHash $installed).Hash) {
        $dependenciesMatch = $false
        break
    }
}
if (-not (Test-Path "web/node_modules/vite") -or -not $dependenciesMatch) {
    Write-Host "  ➜ Installing frontend dependencies..." -ForegroundColor Cyan
    npm ci --prefix web --ignore-scripts
    if ($LASTEXITCODE -ne 0) { throw "Installing frontend dependencies failed." }
    $dependenciesChanged = $true
}
$distIndex = "web/dist/index.html"
$stale = (Test-Path $distIndex) -and (Get-ChildItem web/src -Recurse -File | Where-Object { $_.LastWriteTime -gt (Get-Item $distIndex).LastWriteTime } | Select-Object -First 1)
if ($dependenciesChanged -or -not (Test-Path $distIndex) -or $stale) {
    Write-Host "  ➜ Building frontend production bundle..." -ForegroundColor Cyan
    npm run build --prefix web
    if ($LASTEXITCODE -ne 0) { throw "Frontend build failed." }
}
if ($dependenciesChanged) {
    # Record only after a successful build, so a failed setup is retried.
    foreach ($file in $dependencyFiles) {
        Copy-Item -LiteralPath "web/$file" -Destination "web/node_modules/.trestle-$file"
    }
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
