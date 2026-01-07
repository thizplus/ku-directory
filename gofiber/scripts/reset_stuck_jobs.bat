@echo off
REM Reset Stuck Jobs Script (Batch)
REM Use this when jobs are stuck without clearing all data

echo ========================================
echo  RESET STUCK JOBS
echo ========================================
echo.

set PGPASSWORD=n147369
psql -h localhost -U postgres -d ku_db -f "%~dp0reset_stuck_jobs.sql"

if %ERRORLEVEL% EQU 0 (
    echo   Jobs reset successfully
) else (
    echo   Error resetting jobs
)

echo.
echo ========================================
echo  DONE
echo ========================================
pause
