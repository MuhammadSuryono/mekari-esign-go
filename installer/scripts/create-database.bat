@echo off
setlocal enabledelayedexpansion

set INSTALL_DIR=%~1
if "%INSTALL_DIR%"=="" set INSTALL_DIR=%~dp0..

echo ============================================
echo Creating Application Database
echo ============================================

set PG_BIN=%INSTALL_DIR%\pgsql\bin
set PGPASSWORD=postgres123

echo Waiting for PostgreSQL to be ready...

:: Try multiple times
set RETRIES=10
set COUNT=0

:CHECK_PG
set /a COUNT+=1
"%PG_BIN%\pg_isready.exe" -h localhost -p 5432 >nul 2>&1
if %errorlevel% equ 0 goto PG_READY

if %COUNT% lss %RETRIES% (
    echo Waiting for PostgreSQL... (attempt %COUNT%/%RETRIES%)
    timeout /t 2 /nobreak >nul
    goto CHECK_PG
)

echo ERROR: PostgreSQL is not running after %RETRIES% attempts
exit /b 1

:PG_READY
echo PostgreSQL is ready!

:: Create database if not exists
echo Creating database mekari_esign...
"%PG_BIN%\psql.exe" -h localhost -U postgres -tc "SELECT 1 FROM pg_database WHERE datname = 'mekari_esign'" 2>nul | find "1" >nul 2>&1
if %errorlevel% neq 0 (
    "%PG_BIN%\createdb.exe" -h localhost -U postgres mekari_esign 2>nul
    if %errorlevel% equ 0 (
        echo Database created successfully!
    ) else (
        echo Warning: Could not create database. You may need to create it manually.
        echo Command: "%PG_BIN%\createdb.exe" -h localhost -U postgres mekari_esign
    )
) else (
    echo Database already exists.
)

exit /b 0

