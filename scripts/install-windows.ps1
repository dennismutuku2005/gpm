# Go Process Manager (GPM) Windows Installer
# Run this script as Administrator

$ErrorActionPreference = "Stop"

Write-Host "=== GPM Windows Installation Start ===" -ForegroundColor Cyan

# Check if running as Admin
$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Error "Please run this PowerShell script as Administrator!"
    Exit 1
}

# Compile GPM
Write-Host "Compiling GPM..." -ForegroundColor Yellow
& go build -o gpm.exe cmd/gpm/main.go

# Install folder
$InstallDir = "C:\Program Files\GPM"
Write-Host "Creating installation directory: $InstallDir..." -ForegroundColor Yellow
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
}

Write-Host "Copying binary..." -ForegroundColor Yellow
Copy-Item gpm.exe -Destination "$InstallDir\gpm.exe" -Force

# Add to system PATH
Write-Host "Configuring environment PATH..." -ForegroundColor Yellow
$MachinePath = [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::Machine)
if ($MachinePath -notsplit ';' -contains $InstallDir) {
    $NewMachinePath = $MachinePath + ";" + $InstallDir
    [Environment]::SetEnvironmentVariable("Path", $NewMachinePath, [EnvironmentVariableTarget]::Machine)
    Write-Host "Added $InstallDir to system environment PATH." -ForegroundColor Green
} else {
    Write-Host "$InstallDir is already in environment PATH." -ForegroundColor Gray
}

# Refresh PATH in current session
$env:Path = [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::Machine) + ";" + [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::User)

# Install Windows Service
Write-Host "Configuring GPM Windows Service automatic startup..." -ForegroundColor Yellow
# Run gpm.exe directly from the installation path
& "$InstallDir\gpm.exe" startup

# Verify
Write-Host "`nVerifying GPM installation..." -ForegroundColor Yellow
& gpm version
& gpm status

Write-Host "`n=== GPM Windows Installation Completed Successfully! ===" -ForegroundColor Green
Write-Host "Please restart your terminal/powershell to refresh the environment variables and run 'gpm' globally." -ForegroundColor Cyan
