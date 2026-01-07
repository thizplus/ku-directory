@echo off
REM Clear All Data Script (Batch)
REM This script clears both database and log files

echo ========================================
echo  CLEAR ALL DATA SCRIPT
echo ========================================
echo.

REM 1. Clear Log Files
echo [1/2] Clearing log files...

set "SCRIPT_DIR=%~dp0"
set "LOG_DIR=%SCRIPT_DIR%..\logs"

if exist "%LOG_DIR%" (
    del /q "%LOG_DIR%\*.log" 2>nul
    echo   Log files cleared
) else (
    mkdir "%LOG_DIR%"
    echo   Log directory created
)

echo.

REM 2. Clear Database
echo [2/2] Clearing database...

set PGPASSWORD=n147369
psql -h localhost -U postgres -d ku_db -f "%SCRIPT_DIR%clear_all_data.sql"

if %ERRORLEVEL% EQU 0 (
    echo   Database cleared successfully
) else (
    echo   Error clearing database
)

echo.
echo ========================================
echo  DONE
echo ========================================
pause
