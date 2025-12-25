@echo off
setlocal enabledelayedexpansion

set INSTALL_DIR=%~1
if "%INSTALL_DIR%"=="" set INSTALL_DIR=%~dp0..

echo ============================================
echo Uninstalling All Mekari E-Sign Services
echo ============================================

set NSSM=%INSTALL_DIR%\tools\nssm.exe

:: Stop all services
echo Stopping services...
net stop MekariEsign 2>nul
net stop MekariPostgres 2>nul
net stop MekariRedis 2>nul

timeout /t 3 /nobreak >nul

:: Uninstall main service
echo Removing MekariEsign service...
"%INSTALL_DIR%\mekari-esign.exe" -uninstall 2>nul

:: Uninstall PostgreSQL service
echo Removing PostgreSQL service...
"%NSSM%" remove MekariPostgres confirm 2>nul

:: Uninstall Redis service
echo Removing Redis service...
"%NSSM%" remove MekariRedis confirm 2>nul

:: Remove scheduled task
echo Removing scheduled update task...
schtasks /delete /tn "MekariEsignUpdater" /f 2>nul

echo All services removed successfully!
exit /b 0

