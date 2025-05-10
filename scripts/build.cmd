@echo off
REM scripts/build.cmd
REM Fixed build script with proper path handling for spaces

echo === Building BTC Force with CUDA GPU Support ===

REM Set CUDA path with proper quoting
set "CUDA_PATH=%ProgramFiles%\NVIDIA GPU Computing Toolkit\CUDA\v12.9"
if not exist "%CUDA_PATH%" (
    set "CUDA_PATH=%ProgramFiles%\NVIDIA GPU Computing Toolkit\CUDA\v12.9"
)

if not exist "%CUDA_PATH%" (
    echo Error: CUDA not found
    echo Checked paths:
    echo - %ProgramFiles%\NVIDIA GPU Computing Toolkit\CUDA\v12.9
    echo - %ProgramFiles%\NVIDIA GPU Computing Toolkit\CUDA\v12.9
    exit /b 1
)

echo CUDA found at: %CUDA_PATH%

REM Check for gcc
where gcc >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo Error: gcc not found in PATH
    echo Please install MinGW-w64 and add it to PATH
    exit /b 1
)

REM Set environment for CUDA compilation with proper quoting
set CGO_ENABLED=1

REM Use escape characters for paths with spaces
set "CGO_CFLAGS=-I\"%CUDA_PATH%\include\""
set "CGO_LDFLAGS=-L\"%CUDA_PATH%\lib\x64\" -lcudart -lcuda"

REM Alternative: Use short path names
for %%i in ("%CUDA_PATH%") do set "CUDA_SHORT=%%~si"
set "CGO_CFLAGS=-I%CUDA_SHORT%\include"
set "CGO_LDFLAGS=-L%CUDA_SHORT%\lib\x64 -lcudart -lcuda"

echo Using CUDA short path: %CUDA_SHORT%
echo CGO_CFLAGS: %CGO_CFLAGS%
echo CGO_LDFLAGS: %CGO_LDFLAGS%

echo.
echo Building btcforce.exe with GPU support...
go build -v -x -o btcforce-gpu.exe cmd\btcforce\main.go

if %ERRORLEVEL% EQU 0 (
    echo Build successful!
    echo Executable: btcforce-gpu.exe
) else (
    echo Build failed!
    echo.
    echo Troubleshooting tips:
    echo 1. Make sure CUDA is properly installed
    echo 2. Verify gcc is in your PATH
    echo 3. Try using the simple build instead: build-simple.cmd
    exit /b 1
)