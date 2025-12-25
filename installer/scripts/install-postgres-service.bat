@echo off
setlocal enabledelayedexpansion

set INSTALL_DIR=%~1
if "%INSTALL_DIR%"=="" set INSTALL_DIR=%~dp0..

echo ============================================
echo Installing PostgreSQL as Windows Service
echo ============================================

set NSSM=%INSTALL_DIR%\tools\nssm.exe
set PG_BIN=%INSTALL_DIR%\pgsql\bin
set PG_DATA=%INSTALL_DIR%\data\postgres
set LOG_DIR=%INSTALL_DIR%\logs

echo Install directory: %INSTALL_DIR%
echo NSSM path: %NSSM%
echo PostgreSQL binaries: %PG_BIN%
echo Data directory: %PG_DATA%

:: Check if service already exists
sc query MekariPostgres >nul 2>&1
if %errorlevel% equ 0 (
    echo PostgreSQL service already exists. Removing old service...
    "%NSSM%" stop MekariPostgres >nul 2>&1
    "%NSSM%" remove MekariPostgres confirm >nul 2>&1
    timeout /t 2 /nobreak >nul
)

:: Install service using NSSM
echo Installing PostgreSQL service...
"%NSSM%" install MekariPostgres "%PG_BIN%\pg_ctl.exe" "start -D ""%PG_DATA%"" -w -l ""%LOG_DIR%\postgresql.log"""

if %errorlevel% neq 0 (
    echo ERROR: Failed to install PostgreSQL service
    exit /b 1
)

:: Configure service
"%NSSM%" set MekariPostgres AppDirectory "%PG_BIN%"
"%NSSM%" set MekariPostgres DisplayName "Mekari PostgreSQL Server"
"%NSSM%" set MekariPostgres Description "PostgreSQL database server for Mekari E-Sign Service"
"%NSSM%" set MekariPostgres Start SERVICE_AUTO_START
"%NSSM%" set MekariPostgres AppStopMethodConsole 30000
"%NSSM%" set MekariPostgres AppStopMethodWindow 30000
"%NSSM%" set MekariPostgres AppExit Default Restart
"%NSSM%" set MekariPostgres AppRestartDelay 5000

:: Set stop command
"%NSSM%" set MekariPostgres AppEvents Stop/Pre "pg_ctl.exe stop -D ""%PG_DATA%"" -m fast"

echo PostgreSQL service installed successfully!
exit /b 0

