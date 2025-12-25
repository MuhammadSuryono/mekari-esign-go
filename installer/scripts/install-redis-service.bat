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
echo NSSM path: %NSSM%
echo Redis directory: %REDIS_DIR%

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
        echo logfile %LOG_DIR:\=/%/redis.log
        echo loglevel notice
    ) > "%REDIS_DIR%\redis.conf"
)

:: Create data directory
if not exist "%REDIS_DATA%" mkdir "%REDIS_DATA%"

:: Check if service already exists
sc query MekariRedis >nul 2>&1
if %errorlevel% equ 0 (
    echo Redis service already exists. Removing old service...
    "%NSSM%" stop MekariRedis >nul 2>&1
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

echo Redis service installed successfully!
exit /b 0

