@echo off
REM scripts/monitor.cmd

setlocal enabledelayedexpansion

set PORT=%1
if "%PORT%"=="" set PORT=8177
set HOST=localhost

echo === BTC Force Monitor for Windows ===
echo Monitoring at http://%HOST%:%PORT%
echo.

:loop
cls
echo === BTC Force Monitor ===
echo Time: %date% %time%
echo.

REM Check if curl is available
where curl >nul 2>1
if %ERRORLEVEL% NEQ 0 (
    echo Error: curl is not installed or not in PATH
    echo Please install curl or add it to your PATH
    goto :end
)

REM Get runtime stats
echo Runtime Stats:
for /f "delims=" %%i in ('curl -s "http://%HOST%:%PORT%/runtime"') do set runtime=%%i
echo %runtime% | findstr /C:"goroutines" >nul
if %ERRORLEVEL% EQU 0 (
    echo %runtime%
) else (
    echo Failed to get runtime stats
)
echo.

REM Get main stats
echo Progress Stats:
for /f "delims=" %%i in ('curl -s "http://%HOST%:%PORT%/stats"') do set stats=%%i
echo %stats% | findstr /C:"total_visited" >nul
if %ERRORLEVEL% EQU 0 (
    echo %stats%
) else (
    echo Failed to get progress stats
)
echo.

REM Get worker details
echo Worker Status:
for /f "delims=" %%i in ('curl -s "http://%HOST%:%PORT%/workers"') do set workers=%%i
echo %workers% | findstr /C:"Worker" >nul
if %ERRORLEVEL% EQU 0 (
    echo %workers%
) else (
    echo No worker data available yet
)
echo.

echo Press Ctrl+C to exit
echo Refreshing in 5 seconds...
timeout /t 5 /nobreak >nul

goto :loop

:end
endlocal