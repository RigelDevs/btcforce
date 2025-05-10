@echo off
REM scripts/view-log.cmd

setlocal enabledelayedexpansion

echo === BTC Force Log Viewer ===
echo Starting btcforce with colored output...
echo.

REM Check if btcforce.exe exists
if not exist "btcforce.exe" (
    echo Error: btcforce.exe not found!
    echo Please run build.cmd first
    goto :end
)

REM Simple output without color parsing for Windows
btcforce.exe

:end
endlocal