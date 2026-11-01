@echo off
title Secretaria Lucia (Telegram)
cd /d "%~dp0"
powershell -NoProfile -Command "if (Get-CimInstance Win32_Process | Where-Object { $_.Name -eq 'claude.exe' -and $_.CommandLine -match 'channels plugin:telegram' }) { exit 1 }"
if errorlevel 1 (
  echo A Lucia ja esta aberta em outra janela. Use aquela. Esta vai fechar.
  timeout /t 6 >nul
  exit /b
)
rem Encerra servidor do Telegram orfao de uma janela anterior (senao o bot fica bloqueado)
powershell -NoProfile -Command "Get-CimInstance Win32_Process | Where-Object { $_.Name -eq 'bun.exe' -and $_.CommandLine -match 'server.ts' -and $_.CommandLine -notmatch 'claude-mem' } | ForEach-Object { Stop-Process -Id $_.ProcessId -Force }"
timeout /t 2 >nul
set "PATH=%USERPROFILE%\.bun\bin;%PATH%"
for /f "usebackq tokens=1,* delims==" %%a in ("%USERPROFILE%\.claude\channels\lucia-token.env") do set "%%a=%%b"
claude --settings "%~dp0.claude\lucia.settings.json" --channels plugin:telegram@claude-plugins-official
