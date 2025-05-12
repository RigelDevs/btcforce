@echo off
setlocal

:: Use short path names to avoid space issues
for %%i in ("C:\Program Files\NVIDIA GPU Computing Toolkit\CUDA\v12.9") do set CUDA_PATH=%%~si
echo Using CUDA path: %CUDA_PATH%

:: Set PATH
set PATH=%CUDA_PATH%\bin;%PATH%

:: Set CGO environment variables with quotes and escaped quotes
set CGO_CFLAGS=-I%CUDA_PATH%\include
set CGO_LDFLAGS=-L%CUDA_PATH%\lib\x64 -lcudart -lcuda

:: Display the settings
echo CGO_CFLAGS=%CGO_CFLAGS%
echo CGO_LDFLAGS=%CGO_LDFLAGS%

:: Compile and run the Go program
go run cmd\btcforce\main.go

echo Running Program completed.