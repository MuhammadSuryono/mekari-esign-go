@echo off
echo ============================================
echo Stopping All Mekari E-Sign Services
echo ============================================

echo Stopping Mekari E-Sign...
net stop MekariEsign 2>nul

echo Stopping PostgreSQL...
net stop MekariPostgres 2>nul

echo Stopping Redis...
net stop MekariRedis 2>nul

echo.
echo All services stopped!
echo.
pause

