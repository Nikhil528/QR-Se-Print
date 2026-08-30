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
