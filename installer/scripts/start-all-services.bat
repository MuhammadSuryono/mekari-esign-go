@echo off
echo ============================================
echo Starting All Mekari E-Sign Services
echo ============================================

echo Starting Redis...
net start MekariRedis
if %errorlevel% neq 0 echo Warning: Failed to start Redis

timeout /t 2 /nobreak >nul

echo Starting PostgreSQL...
net start MekariPostgres
if %errorlevel% neq 0 echo Warning: Failed to start PostgreSQL

timeout /t 3 /nobreak >nul

echo Starting Mekari E-Sign...
net start MekariEsign
if %errorlevel% neq 0 echo Warning: Failed to start MekariEsign

echo.
echo All services started!
echo.
pause

