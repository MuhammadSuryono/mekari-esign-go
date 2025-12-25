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
timeout /t 3 /nobreak >nul

:: Check if PostgreSQL is running
"%PG_BIN%\pg_isready.exe" -h localhost -p 5432 >nul 2>&1
if %errorlevel% neq 0 (
    echo Waiting for PostgreSQL to start...
    timeout /t 5 /nobreak >nul
    "%PG_BIN%\pg_isready.exe" -h localhost -p 5432 >nul 2>&1
    if %errorlevel% neq 0 (
        echo ERROR: PostgreSQL is not running
        exit /b 1
    )
)

:: Create database if not exists
echo Creating database mekari_esign...
"%PG_BIN%\psql.exe" -h localhost -U postgres -tc "SELECT 1 FROM pg_database WHERE datname = 'mekari_esign'" | find "1" >nul 2>&1
if %errorlevel% neq 0 (
    "%PG_BIN%\createdb.exe" -h localhost -U postgres mekari_esign
    if %errorlevel% equ 0 (
        echo Database created successfully!
    ) else (
        echo Warning: Could not create database. You may need to create it manually.
    )
) else (
    echo Database already exists.
)

exit /b 0

