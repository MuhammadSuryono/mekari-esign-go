@echo off
setlocal enabledelayedexpansion

set INSTALL_DIR=%~1
if "%INSTALL_DIR%"=="" set INSTALL_DIR=%~dp0..

echo ============================================
echo Installing PostgreSQL as Windows Service
echo ============================================

set PG_BIN=%INSTALL_DIR%\pgsql\bin
set PG_DATA=%INSTALL_DIR%\data\postgres
set LOG_DIR=%INSTALL_DIR%\logs

echo Install directory: %INSTALL_DIR%
echo PostgreSQL binaries: %PG_BIN%
echo Data directory: %PG_DATA%

:: Create directories
if not exist "%LOG_DIR%" mkdir "%LOG_DIR%"

:: Check if data directory is initialized
if not exist "%PG_DATA%\PG_VERSION" (
    echo ERROR: PostgreSQL data directory not initialized!
    echo Please run init-postgres.bat first.
    exit /b 1
)

:: Remove existing service if exists
sc query MekariPostgres >nul 2>&1
if %errorlevel% equ 0 (
    echo Removing existing PostgreSQL service...
    net stop MekariPostgres >nul 2>&1
    "%PG_BIN%\pg_ctl.exe" unregister -N MekariPostgres >nul 2>&1
    timeout /t 2 /nobreak >nul
)

:: Register PostgreSQL service using pg_ctl (native way, no NSSM needed)
echo Registering PostgreSQL service...
"%PG_BIN%\pg_ctl.exe" register -N MekariPostgres -D "%PG_DATA%" -S auto -w

if %errorlevel% neq 0 (
    echo WARNING: pg_ctl register failed, trying alternative method...
    
    :: Try with NSSM as fallback
    set NSSM=%INSTALL_DIR%\tools\nssm.exe
    
    if not exist "!NSSM!" (
        where nssm >nul 2>&1
        if %errorlevel% equ 0 (
            for /f "tokens=*" %%i in ('where nssm') do set NSSM=%%i
        )
    )
    
    if exist "!NSSM!" (
        echo Using NSSM fallback...
        "!NSSM!" install MekariPostgres "%PG_BIN%\postgres.exe" "-D \"%PG_DATA%\""
        "!NSSM!" set MekariPostgres AppDirectory "%PG_BIN%"
        "!NSSM!" set MekariPostgres DisplayName "Mekari PostgreSQL Server"
        "!NSSM!" set MekariPostgres Start SERVICE_AUTO_START
        "!NSSM!" set MekariPostgres AppStdout "%LOG_DIR%\postgresql-stdout.log"
        "!NSSM!" set MekariPostgres AppStderr "%LOG_DIR%\postgresql-stderr.log"
        "!NSSM!" set MekariPostgres AppExit Default Restart
        "!NSSM!" set MekariPostgres AppRestartDelay 5000
    ) else (
        echo ERROR: Could not register PostgreSQL service
        exit /b 1
    )
)

:: Set service description
sc description MekariPostgres "PostgreSQL database server for Mekari E-Sign Service" >nul 2>&1

echo PostgreSQL service installed successfully!
echo Starting PostgreSQL service...
net start MekariPostgres

:: Wait for PostgreSQL to be ready
echo Waiting for PostgreSQL to start...
timeout /t 3 /nobreak >nul

:: Check if running
"%PG_BIN%\pg_isready.exe" -h localhost -p 5432 >nul 2>&1
if %errorlevel% equ 0 (
    echo PostgreSQL is running and ready!
) else (
    echo WARNING: PostgreSQL may still be starting...
)

exit /b 0
