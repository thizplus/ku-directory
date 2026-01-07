# Clear All Data Script (PowerShell)
# This script clears both database and log files

Write-Host "========================================" -ForegroundColor Cyan
Write-Host " CLEAR ALL DATA SCRIPT" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# 1. Clear Log Files
Write-Host "[1/2] Clearing log files..." -ForegroundColor Yellow

$logDir = Join-Path $PSScriptRoot "..\logs"
if (Test-Path $logDir) {
    $logFiles = Get-ChildItem -Path $logDir -Filter "*.log" -ErrorAction SilentlyContinue
    if ($logFiles.Count -gt 0) {
        foreach ($file in $logFiles) {
            Remove-Item $file.FullName -Force
            Write-Host "  Deleted: $($file.Name)" -ForegroundColor Gray
        }
        Write-Host "  Deleted $($logFiles.Count) log files" -ForegroundColor Green
    } else {
        Write-Host "  No log files found" -ForegroundColor Gray
    }
} else {
    Write-Host "  Log directory not found, creating..." -ForegroundColor Gray
    New-Item -ItemType Directory -Path $logDir -Force | Out-Null
}

Write-Host ""

# 2. Clear Database
Write-Host "[2/2] Clearing database..." -ForegroundColor Yellow

$sqlFile = Join-Path $PSScriptRoot "clear_all_data.sql"
if (Test-Path $sqlFile) {
    # Run the SQL script using psql
    $env:PGPASSWORD = "n147369"
    $result = & psql -h localhost -U postgres -d ku_db -f $sqlFile 2>&1

    if ($LASTEXITCODE -eq 0) {
        Write-Host "  Database cleared successfully" -ForegroundColor Green
        Write-Host ""
        Write-Host "  Table counts:" -ForegroundColor Cyan
        $result | ForEach-Object { Write-Host "    $_" }
    } else {
        Write-Host "  Error clearing database:" -ForegroundColor Red
        $result | ForEach-Object { Write-Host "    $_" -ForegroundColor Red }
    }
} else {
    Write-Host "  SQL file not found: $sqlFile" -ForegroundColor Red
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host " DONE" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Cyan
