# QR Se Print Agent — Native Dashboard Build

- Dashboard is a native Windows window inside the EXE; it does not open Chrome/Edge for the dashboard.
- The local HTTP listener on 127.0.0.1:17845 is control-only for the tray and never serves the dashboard UI.
- Dashboard opens automatically when the agent starts.
- Closing the dashboard hides it to the tray; it does not stop the print agent.
- Tray Settings/double-click reopens the native dashboard.
- Plan/Upgrade and Contact Admin may intentionally open the external website.
- Default API server: https://bvv-djql.onrender.com
- Existing print engine and printer routing are retained.
