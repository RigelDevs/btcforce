@echo off
REM scripts/debug-api.cmd

setlocal enabledelayedexpansion

set PORT=%1
if "%PORT%"=="" set PORT=8177
set HOST=localhost

echo === BTC Force API Debug ===
echo Testing endpoints at http://%HOST%:%PORT%
echo.

REM Check if curl is available
where curl >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo Error: curl is not installed or not in PATH
    echo Please install curl or add it to your PATH
    goto :end
)

echo 1. Testing /health endpoint:
curl -s "http://%HOST%:%PORT%/health"
echo.
echo.

echo 2. Testing /runtime endpoint:
curl -s "http://%HOST%:%PORT%/runtime"
echo.
echo.

echo 3. Testing /stats endpoint:
curl -s "http://%HOST%:%PORT%/stats"
echo.
echo.

echo 4. Testing /workers endpoint:
curl -s "http://%HOST%:%PORT%/workers"
echo.
echo.

echo 5. Checking if server is running:
curl -s --head "http://%HOST%:%PORT%/health" | findstr /r "HTTP/1\.[01] [23].." >nul
if %ERRORLEVEL% EQU 0 (
    echo √ Server is responding
) else (
    echo × Server is not responding or not running
    echo Make sure btcforce is running on port %PORT%
)

:end
endlocal