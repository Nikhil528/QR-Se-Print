# QR Se Print Agent — Final Native Dashboard

- Dashboard is rendered inside the EXE using native Win32 controls and painting.
- Chrome/Edge is not used for the dashboard.
- PowerShell is not used by the agent for login, tray, printer enumeration, or queue diagnostics.
- Windows printer list is read through Winspool APIs.
- Login is a native Windows dialog.
- Tray is native Windows Shell_NotifyIcon.
- Dashboard opens automatically when the agent starts.
- Closing the dashboard hides it to the tray.
- Existing server API and SumatraPDF print flow are retained.
- Default server: https://bvv-djql.onrender.com
