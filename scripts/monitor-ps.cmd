@echo off
REM scripts/monitor-ps.cmd
REM Launches the PowerShell monitor

powershell.exe -ExecutionPolicy Bypass -File "%~dp0monitor.ps1" %*