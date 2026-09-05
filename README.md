# QR Se Print — Render Final

Recommended deployment: Render Web Service + persistent disk. The application is intentionally kept as one backend/frontend service so the same origin handles `/api`, `/admin`, `/register`, `/print/:shopId`, uploads and QR generation.

## Render environment variables
- `ADMIN_USER` = your super admin username
- `ADMIN_PASSWORD` = a strong super admin password
- `RAZORPAY_KEY_ID` = Razorpay Key ID
- `RAZORPAY_KEY_SECRET` = Razorpay Key Secret
- `DATA_DIR=/var/data`

Do not put Razorpay Secret in frontend code.

## Super Admin
Open `/superadmin` and sign in with the Render `ADMIN_USER` / `ADMIN_PASSWORD` values. Admin can view shops, agent status, plans and reset passwords.

## Shop registration
Open `/register`. Demo creates an immediate Shop ID + password, valid for 24 hours and 10 prints. Starter/Pro/Premium are represented as lifetime plans and start in `pending_payment` until the commercial activation/payment flow is connected.

## Persistence
Do not use an ephemeral filesystem for production shop data. The included Render configuration uses a persistent disk at `/var/data`. If you use a different storage provider, set `DATA_DIR` accordingly.


## Paid Plan Registration / Razorpay

Set these Render Environment Variables for Starter/Pro/Premium registration payments:
- `PLATFORM_RAZORPAY_KEY_ID` = Razorpay Key ID
- `PLATFORM_RAZORPAY_KEY_SECRET` = Razorpay Key Secret

Fallback names `RAZORPAY_KEY_ID` and `RAZORPAY_KEY_SECRET` are also accepted.

Demo registration remains free and creates credentials immediately. Paid plans create Shop ID/password only after server-side Razorpay signature verification.


## Central settings — `data/db.json`

All important non-secret application settings are kept under the `settings` object in `data/db.json`. Edit these values directly; the server only fills missing keys and does **not** overwrite existing values.

Main sections:
- `settings.plans` — Demo/Starter/Pro/Premium price, duration, print limit and advance access
- `settings.demo` — demo enable/disable, duration, print limit and demo prices
- `settings.defaults.shop` — default shop prices, payment mode, duplex prices and printer defaults for newly created shops
- `settings.server` — upload/request limits
- `settings.health` — agent online timeout
- `settings.uploads` — automatic uploaded-file cleanup
- `settings.agent` — agent poll interval/version returned at agent login
- `settings.printers.models` — printer model list
- `settings.features.advance` — advance-feature list
- `settings.payments` — setup fee/currency
- `settings.security` — configurable password minimums

After manually editing `data/db.json`, restart the server/Render service so the new settings are loaded. Existing shops keep their own saved settings; changing `defaults.shop` affects new shops and does not reset existing shops.

**Secrets stay out of `db.json`**: admin password and Razorpay/Cashfree secrets should remain in environment variables or per-shop secret fields.
