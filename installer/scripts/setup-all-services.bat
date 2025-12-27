@echo off
setlocal enabledelayedexpansion

:: ==============================================
:: Mekari E-Sign - Complete Service Setup
:: Run this script as Administrator
:: ==============================================

set INSTALL_DIR=%~1
if "%INSTALL_DIR%"=="" set INSTALL_DIR=%~dp0..

echo ============================================
echo Mekari E-Sign - Complete Service Setup
echo ============================================
echo.
echo Install directory: %INSTALL_DIR%
echo.

:: Check for admin privileges
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo ERROR: This script requires Administrator privileges!
    echo Please right-click and select "Run as Administrator"
    pause
    exit /b 1
)

:: Create all directories
echo [1/6] Creating directories...
if not exist "%INSTALL_DIR%\data" mkdir "%INSTALL_DIR%\data"
if not exist "%INSTALL_DIR%\data\postgres" mkdir "%INSTALL_DIR%\data\postgres"
if not exist "%INSTALL_DIR%\data\redis" mkdir "%INSTALL_DIR%\data\redis"
if not exist "%INSTALL_DIR%\logs" mkdir "%INSTALL_DIR%\logs"
if not exist "%INSTALL_DIR%\documents" mkdir "%INSTALL_DIR%\documents"
if not exist "%INSTALL_DIR%\documents\ready" mkdir "%INSTALL_DIR%\documents\ready"
if not exist "%INSTALL_DIR%\documents\progress" mkdir "%INSTALL_DIR%\documents\progress"
if not exist "%INSTALL_DIR%\documents\finish" mkdir "%INSTALL_DIR%\documents\finish"
echo Done.

:: Check/Install NSSM
echo.
echo [2/6] Checking NSSM (Service Manager)...
set NSSM=%INSTALL_DIR%\tools\nssm.exe
if not exist "%NSSM%" (
    where nssm >nul 2>&1
    if %errorlevel% equ 0 (
        for /f "tokens=*" %%i in ('where nssm') do set NSSM=%%i
        echo Found NSSM at: !NSSM!
    ) else (
        echo NSSM not found. Installing via Chocolatey...
        where choco >nul 2>&1
        if %errorlevel% equ 0 (
            choco install nssm -y --no-progress
            for /f "tokens=*" %%i in ('where nssm 2^>nul') do set NSSM=%%i
        ) else (
            echo ERROR: Chocolatey not found. Please install NSSM manually.
            echo   1. Download from: https://nssm.cc/download
            echo   2. Copy nssm.exe to: %INSTALL_DIR%\tools\
            pause
            exit /b 1
        )
    )
) else (
    echo Found NSSM at: %NSSM%
)

:: Initialize PostgreSQL
echo.
echo [3/6] Initializing PostgreSQL...
call "%INSTALL_DIR%\scripts\init-postgres.bat" "%INSTALL_DIR%"
if %errorlevel% neq 0 (
    echo WARNING: PostgreSQL initialization had issues.
)

:: Install PostgreSQL Service
echo.
echo [4/6] Installing PostgreSQL service...
call "%INSTALL_DIR%\scripts\install-postgres-service.bat" "%INSTALL_DIR%"
if %errorlevel% neq 0 (
    echo WARNING: PostgreSQL service installation had issues.
)

:: Wait for PostgreSQL
timeout /t 3 /nobreak >nul

:: Create database
echo.
echo [5/6] Creating database...
call "%INSTALL_DIR%\scripts\create-database.bat" "%INSTALL_DIR%"

:: Install Redis Service
echo.
echo [6/6] Installing Redis service...
call "%INSTALL_DIR%\scripts\install-redis-service.bat" "%INSTALL_DIR%"
if %errorlevel% neq 0 (
    echo WARNING: Redis service installation had issues.
)

:: Summary
echo.
echo ============================================
echo Setup Complete!
echo ============================================
echo.
echo Services installed:

sc query MekariPostgres >nul 2>&1
if %errorlevel% equ 0 (
    echo   [OK] MekariPostgres
) else (
    echo   [FAIL] MekariPostgres
)

sc query MekariRedis >nul 2>&1
if %errorlevel% equ 0 (
    echo   [OK] MekariRedis
) else (
    echo   [FAIL] MekariRedis
)

echo.
echo Next steps:
echo   1. Edit configuration: notepad "%INSTALL_DIR%\config.yml"
echo   2. Install main service: "%INSTALL_DIR%\mekari-esign.exe" -install
echo   3. Start main service: "%INSTALL_DIR%\mekari-esign.exe" -start
echo.
pause

exit /b 0

