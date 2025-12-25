@echo off
setlocal enabledelayedexpansion

set INSTALL_DIR=%~1
if "%INSTALL_DIR%"=="" set INSTALL_DIR=%~dp0..

echo ============================================
echo Initializing PostgreSQL Database
echo ============================================

set PG_BIN=%INSTALL_DIR%\pgsql\bin
set PG_DATA=%INSTALL_DIR%\data\postgres

echo Install directory: %INSTALL_DIR%
echo PostgreSQL binaries: %PG_BIN%
echo Data directory: %PG_DATA%

:: Check if already initialized
if exist "%PG_DATA%\PG_VERSION" (
    echo PostgreSQL data directory already exists. Skipping initialization.
    exit /b 0
)

:: Create data directory
if not exist "%PG_DATA%" mkdir "%PG_DATA%"

:: Initialize database
echo Initializing PostgreSQL data directory...
"%PG_BIN%\initdb.exe" -D "%PG_DATA%" -U postgres -E UTF8 -A md5 --pwfile=NUL

if %errorlevel% neq 0 (
    echo ERROR: Failed to initialize PostgreSQL
    exit /b 1
)

:: Configure PostgreSQL
echo Configuring PostgreSQL...

:: Update postgresql.conf
echo.>> "%PG_DATA%\postgresql.conf"
echo # Custom settings for Mekari E-Sign >> "%PG_DATA%\postgresql.conf"
echo listen_addresses = 'localhost' >> "%PG_DATA%\postgresql.conf"
echo port = 5432 >> "%PG_DATA%\postgresql.conf"
echo logging_collector = on >> "%PG_DATA%\postgresql.conf"
echo log_directory = '%INSTALL_DIR:\=/%/logs' >> "%PG_DATA%\postgresql.conf"
echo log_filename = 'postgresql-%%Y-%%m-%%d.log' >> "%PG_DATA%\postgresql.conf"

:: Update pg_hba.conf for local connections
echo # Allow local connections >> "%PG_DATA%\pg_hba.conf"
echo host    all             all             127.0.0.1/32            md5 >> "%PG_DATA%\pg_hba.conf"

echo PostgreSQL initialized successfully!
exit /b 0

