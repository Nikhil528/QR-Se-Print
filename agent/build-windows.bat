@echo off
setlocal
if not exist go.mod (
  echo Go module not found.
  exit /b 1
)
echo Building QR Se Print Agent with native embedded dashboard...
go build -ldflags="-s -w -H=windowsgui" -o QRSePrintAgent.exe main.go
if errorlevel 1 goto :fail
echo Build complete: QRSePrintAgent.exe
goto :eof
:fail
echo Build failed.
exit /b 1
