@echo off
REM scripts/build-cuda.cmd
REM Complete build script for CUDA GPU support

echo === Building BTC Force with CUDA GPU Support ===

REM First, let's create the necessary directory structure
echo Creating directory structure...
mkdir internal\gpu\cuda_headers 2>nul
mkdir internal\gpu\cuda_libs 2>nul

REM Find CUDA installation
echo Locating CUDA installation...
set "CUDA_PATH="
set "CUDA_PATH=%ProgramFiles%\NVIDIA GPU Computing Toolkit\CUDA\v12.9"

if "%CUDA_PATH%"=="" (
    echo Error: CUDA not found in standard locations
    echo Please install CUDA toolkit from NVIDIA
    exit /b 1
)

echo Found CUDA at: %CUDA_PATH%

REM Copy CUDA libraries to local directory to avoid path issues
echo Copying CUDA libraries...
copy "%CUDA_PATH%\lib\x64\cudart64_*.dll" internal\gpu\cuda_libs\ >nul 2>&1
copy "%CUDA_PATH%\lib\x64\cudart.lib" internal\gpu\cuda_libs\cudart64_120.lib >nul 2>&1
copy "%CUDA_PATH%\lib\x64\cuda.lib" internal\gpu\cuda_libs\ >nul 2>&1

REM Copy essential CUDA headers
echo Copying CUDA headers...
copy "%CUDA_PATH%\include\cuda_runtime.h" internal\gpu\cuda_headers\ >nul 2>&1
copy "%CUDA_PATH%\include\cuda_runtime_api.h" internal\gpu\cuda_headers\ >nul 2>&1
copy "%CUDA_PATH%\include\cuda.h" internal\gpu\cuda_headers\ >nul 2>&1
copy "%CUDA_PATH%\include\driver_types.h" internal\gpu\cuda_headers\ >nul 2>&1
copy "%CUDA_PATH%\include\host_defines.h" internal\gpu\cuda_headers\ >nul 2>&1
copy "%CUDA_PATH%\include\device_types.h" internal\gpu\cuda_headers\ >nul 2>&1
copy "%CUDA_PATH%\include\vector_types.h" internal\gpu\cuda_headers\ >nul 2>&1

REM Step 1: Compile CUDA kernels
echo.
echo Step 1: Compiling CUDA kernels...
cd internal\gpu
nvcc -c gpu_kernels.cu -o gpu_kernels.obj
if %ERRORLEVEL% NEQ 0 (
    echo Failed to compile CUDA kernels
    cd ..\..
    exit /b 1
)

REM Step 2: Compile C wrapper
echo Step 2: Compiling C wrapper...
gcc -c gpu_cuda_wrapper.c -o gpu_cuda_wrapper.obj -I"%CUDA_PATH%\include" -Icuda_headers
if %ERRORLEVEL% NEQ 0 (
    echo Failed to compile C wrapper
    cd ..\..
    exit /b 1
)

REM Step 3: Create static library
echo Step 3: Creating static library...
ar rcs libgpu_cuda.a gpu_kernels.obj gpu_cuda_wrapper.obj
if %ERRORLEVEL% NEQ 0 (
    echo Failed to create static library
    cd ..\..
    exit /b 1
)

cd ..\..

REM Step 4: Build Go application with CGO
echo.
echo Step 4: Building Go application...
set CGO_ENABLED=1
set CGO_CFLAGS=-I"%CD%\internal\gpu\cuda_headers" -I"%CD%\internal\gpu"
set CGO_LDFLAGS=-L"%CD%\internal\gpu" -L"%CD%\internal\gpu\cuda_libs" -lgpu_cuda -lcudart64_120 -lcuda

echo Building with CGO settings:
echo CGO_CFLAGS=%CGO_CFLAGS%
echo CGO_LDFLAGS=%CGO_LDFLAGS%

go build -tags cuda -v -o btcforce-gpu.exe cmd\btcforce\main.go

if %ERRORLEVEL% EQU 0 (
    echo.
    echo Build successful!
    echo Executable: btcforce-gpu.exe
    echo.
    echo GPU support is now enabled. Make sure to:
    echo 1. Copy CUDA runtime DLLs to the executable directory
    echo 2. Set USE_GPU=true in your .env file
    echo.
    
    REM Copy required DLLs
    echo Copying required CUDA DLLs...
    copy "%CUDA_PATH%\bin\cudart64_*.dll" . >nul 2>&1
    
) else (
    echo.
    echo Build failed!
    echo.
    echo Troubleshooting:
    echo 1. Check if CUDA is properly installed
    echo 2. Verify nvcc and gcc are in PATH
    echo 3. Try building without GPU: scripts\build-nocgo.cmd
    exit /b 1
)