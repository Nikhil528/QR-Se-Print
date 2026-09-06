# QR Se Print Agent — Enhanced UI Build

Server: https://bvv-djql.onrender.com

## Features
- Demo-style dashboard UI
- Shop ID / server / version / sync status
- B&W, Color, 4x6, A3 and Duplex printer selection
- Windows printer auto-detection and refresh
- Save Printer / Sync Now
- Local dashboard at http://127.0.0.1:17845/
- Windows system-tray helper with reconnect, logs, update, settings and counter-approval toggle
- HKCU auto-start after Windows restart
- Existing print routing and Sumatra printing preserved
- No duplicate-print fallback

## Build
Run `build-windows.bat` on Windows with Go 1.23+ installed. It creates `QRSePrintAgent.exe`.


## v10.1 Login fix
The EXE now embeds login.ps1 and tray.ps1 and extracts them automatically, so running the EXE by itself no longer depends on those files being beside it. Default server: https://bvv-djql.onrender.com
