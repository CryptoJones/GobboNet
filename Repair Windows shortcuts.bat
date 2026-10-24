@echo off
rem This bypass applies only to this repair process, not the system policy.
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0repair-windows-shortcuts.ps1"
if errorlevel 1 echo Shortcut repair failed. Read the error above before continuing.
pause
