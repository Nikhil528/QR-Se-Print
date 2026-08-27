@echo off
setlocal
if not exist QRSePrintAgent.exe (
  echo Building QR Se Print Agent...
  go build -ldflags="-s -w" -o QRSePrintAgent.exe main.go
  if errorlevel 1 goto :fail
)
copy /Y agent-config.json.example agent-config.json >nul 2>nul
if not exist agent-config.json echo Edit agent-config.json before first run.
echo Build complete: QRSePrintAgent.exe
goto :eof
:fail
echo Build failed.
exit /b 1
