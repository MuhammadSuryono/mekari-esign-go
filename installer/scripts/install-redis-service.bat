@echo off
setlocal enabledelayedexpansion

set INSTALL_DIR=%~1
if "%INSTALL_DIR%"=="" set INSTALL_DIR=%~dp0..

echo ============================================
echo Installing Redis as Windows Service
echo ============================================

set NSSM=%INSTALL_DIR%\tools\nssm.exe
set REDIS_DIR=%INSTALL_DIR%\redis
set REDIS_DATA=%INSTALL_DIR%\data\redis
set LOG_DIR=%INSTALL_DIR%\logs

echo Install directory: %INSTALL_DIR%
echo Redis directory: %REDIS_DIR%

:: Create directories
if not exist "%REDIS_DATA%" mkdir "%REDIS_DATA%"
if not exist "%LOG_DIR%" mkdir "%LOG_DIR%"

:: Create redis.conf if not exists
if not exist "%REDIS_DIR%\redis.conf" (
    echo Creating Redis configuration...
    (
        echo # Redis configuration for Mekari E-Sign
        echo bind 127.0.0.1
        echo port 6379
        echo daemonize no
        echo dir %REDIS_DATA:\=/%
        echo appendonly yes
        echo appendfilename "appendonly.aof"
        echo loglevel notice
    ) > "%REDIS_DIR%\redis.conf"
)

:: Check if NSSM exists, if not try to find or install
if not exist "%NSSM%" (
    echo NSSM not found at %NSSM%
    
    :: Try to find nssm in PATH (installed via chocolatey)
    where nssm >nul 2>&1
    if %errorlevel% equ 0 (
        for /f "tokens=*" %%i in ('where nssm') do set NSSM=%%i
        echo Found NSSM at: !NSSM!
    ) else (
        echo NSSM not found. Attempting to install via Chocolatey...
        where choco >nul 2>&1
        if %errorlevel% equ 0 (
            choco install nssm -y --no-progress
            for /f "tokens=*" %%i in ('where nssm 2^>nul') do set NSSM=%%i
        )
        
        if not exist "!NSSM!" (
            echo ERROR: NSSM not available. Please install manually:
            echo   choco install nssm -y
            exit /b 1
        )
    )
)

echo Using NSSM: %NSSM%

:: Remove existing service if exists
sc query MekariRedis >nul 2>&1
if %errorlevel% equ 0 (
    echo Removing existing Redis service...
    net stop MekariRedis >nul 2>&1
    "%NSSM%" remove MekariRedis confirm >nul 2>&1
    timeout /t 2 /nobreak >nul
)

:: Install service using NSSM
echo Installing Redis service...
"%NSSM%" install MekariRedis "%REDIS_DIR%\redis-server.exe" "%REDIS_DIR%\redis.conf"

if %errorlevel% neq 0 (
    echo ERROR: Failed to install Redis service
    exit /b 1
)

:: Configure service
"%NSSM%" set MekariRedis AppDirectory "%REDIS_DIR%"
"%NSSM%" set MekariRedis DisplayName "Mekari Redis Server"
"%NSSM%" set MekariRedis Description "Redis server for Mekari E-Sign Service"
"%NSSM%" set MekariRedis Start SERVICE_AUTO_START
"%NSSM%" set MekariRedis AppStdout "%LOG_DIR%\redis-stdout.log"
"%NSSM%" set MekariRedis AppStderr "%LOG_DIR%\redis-stderr.log"
"%NSSM%" set MekariRedis AppRotateFiles 1
"%NSSM%" set MekariRedis AppRotateBytes 10485760
"%NSSM%" set MekariRedis AppExit Default Restart
"%NSSM%" set MekariRedis AppRestartDelay 3000

echo Redis service installed successfully!
echo Starting Redis service...
net start MekariRedis

exit /b 0
