# QR Se Print — Complete V8

This is the unified version of the supplied QR Se Print project: existing customer/admin UI + Node API + Windows Print Agent.

## Render deployment
1. Push the **contents** of this folder to the GitHub repository root.
2. Render -> New Web Service -> select the repository.
3. Runtime: Docker. Root Directory: blank.
4. Render will use `Dockerfile`; port is 10000.
5. Health check: `/api/health`.

The service does NOT proxy InfinityFree. That avoids the InfinityFree JavaScript/AES anti-bot page that broke server-to-server calls.

## First test
Open:
- `/`
- `/api/health`
- `/api/index.php?route=health`

Expected health response is JSON HTTP 200.

## Demo login
Shop ID: DEMO
Password: 1234

## Windows Print Agent
1. Install Go 1.22+ on Windows for building from source, or use the included prebuilt `QRSePrintAgent.exe` if present.
2. Copy `agent-config.json.example` to `agent-config.json`.
3. Set `ServerURL` to your Render URL, ShopID and AgentToken.
4. Run the agent.
5. Set `Printer` to the exact Windows printer name, or leave blank for the current default printer.

## Important production note
V8 uses JSON/file storage so it is self-contained and easy to test. Render Free web-service disks are ephemeral, so this is **not production-persistent storage**. Before commercial use, connect the API to a persistent database and object/file storage. Online payments also need real gateway credentials and server-side verification.
