@echo off
setlocal enabledelayedexpansion

set INSTALL_DIR=%~1
if "%INSTALL_DIR%"=="" set INSTALL_DIR=%~dp0..

echo ============================================
echo Initializing PostgreSQL Database
echo ============================================

set PG_BIN=%INSTALL_DIR%\pgsql\bin
set PG_DATA=%INSTALL_DIR%\data\postgres
set LOG_DIR=%INSTALL_DIR%\logs

echo Install directory: %INSTALL_DIR%
echo PostgreSQL binaries: %PG_BIN%
echo Data directory: %PG_DATA%

:: Create directories
if not exist "%PG_DATA%" mkdir "%PG_DATA%"
if not exist "%LOG_DIR%" mkdir "%LOG_DIR%"

:: Check if already initialized
if exist "%PG_DATA%\PG_VERSION" (
    echo PostgreSQL data directory already initialized.
    echo Skipping initialization.
    exit /b 0
)

:: Set permissions on data directory
echo Setting directory permissions...
icacls "%PG_DATA%" /grant Everyone:F /T >nul 2>&1

:: Create password file for non-interactive init
set PWFILE=%TEMP%\pgpass.txt
echo postgres123> "%PWFILE%"

:: Initialize database
echo Initializing PostgreSQL data directory...
"%PG_BIN%\initdb.exe" -D "%PG_DATA%" -U postgres -E UTF8 -A md5 --pwfile="%PWFILE%"

:: Clean up password file
del "%PWFILE%" >nul 2>&1

if %errorlevel% neq 0 (
    echo ERROR: Failed to initialize PostgreSQL
    exit /b 1
)

:: Configure PostgreSQL
echo Configuring PostgreSQL...

:: Append custom settings to postgresql.conf
echo.>> "%PG_DATA%\postgresql.conf"
echo # Custom settings for Mekari E-Sign>> "%PG_DATA%\postgresql.conf"
echo listen_addresses = 'localhost'>> "%PG_DATA%\postgresql.conf"
echo port = 5432>> "%PG_DATA%\postgresql.conf"
echo logging_collector = on>> "%PG_DATA%\postgresql.conf"
echo log_directory = 'log'>> "%PG_DATA%\postgresql.conf"
echo log_filename = 'postgresql-%%Y-%%m-%%d.log'>> "%PG_DATA%\postgresql.conf"

echo PostgreSQL initialized successfully!
echo Default credentials:
echo   Username: postgres
echo   Password: postgres123
echo.

exit /b 0
