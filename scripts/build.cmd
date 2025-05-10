@echo off
REM scripts/build.cmd

echo === Building BTC Force for Windows ===

REM Check if CUDA is available
if exist "%CUDA_PATH%\bin\nvcc.exe" (
    echo CUDA found at %CUDA_PATH%
    set CGO_ENABLED=1
    set CGO_CFLAGS=-I"%CUDA_PATH%\include"
    set CGO_LDFLAGS=-L"%CUDA_PATH%\lib\x64" -lcudart -lcuda
) else (
    echo Warning: CUDA not found. Building without GPU support.
    set CGO_ENABLED=0
)

echo Building btcforce.exe...
go build -o btcforce.exe cmd\btcforce\main.go

if %ERRORLEVEL% EQU 0 (
    echo Build successful!
    echo Executable: btcforce.exe
) else (
    echo Build failed!
    exit /b 1
)

REM Build test-config if needed
echo.
echo Building test-config.exe...
go build -o test-config.exe cmd\test-config\main.go

if %ERRORLEVEL% EQU 0 (
    echo test-config.exe built successfully!
) else (
    echo Warning: test-config build failed
)