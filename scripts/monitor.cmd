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
where curl >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo Error: curl is not installed or not in PATH
    echo Please install curl or add it to your PATH
    goto :end
)

REM Get runtime stats
echo Runtime Stats:
curl -s "http://%HOST%:%PORT%/runtime" > runtime.json
type runtime.json
echo.
echo.

REM Get main stats
echo Progress Stats:
curl -s "http://%HOST%:%PORT%/stats" > stats.json
type stats.json
echo.
echo.

REM Get worker details with better parsing
echo Worker Status:
curl -s "http://%HOST%:%PORT%/workers" > workers.json

REM Check if we got valid JSON with workers
findstr /C:"\"workers\"" workers.json >nul
if %ERRORLEVEL% EQU 0 (
    REM Try to parse worker count
    for /f "tokens=2 delims=:" %%a in ('findstr /C:"\"total\"" workers.json') do (
        set worker_count=%%a
        set worker_count=!worker_count:,=!
        set worker_count=!worker_count: =!
        echo Total Workers: !worker_count!
    )
    
    REM Display full worker data
    type workers.json
) else (
    echo No worker data available yet
    echo Raw response:
    type workers.json
)
echo.

echo Press Ctrl+C to exit
echo Refreshing in 5 seconds...
timeout /t 5 /nobreak >nul

goto :loop

:end
endlocal