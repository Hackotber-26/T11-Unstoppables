# Deployment: Vercel, Render, and Neon

This deploys the static frontend to Vercel, the Go API to Render, and PostgreSQL to Neon. The Render blueprint uses a paid instance and persistent disk for uploaded driver documents. This is a deployable production baseline, not a certification: operators must configure and verify the external providers, privacy/retention policy, backups, monitoring, and incident response before accepting real users or identity documents.

## What you need

- GitHub account with this repository pushed to a GitHub repository.
- Accounts at [Vercel](https://vercel.com/), [Render](https://render.com/), and [Neon](https://neon.tech/). The Render service and persistent disk incur charges.
- A controlled OSRM service over HTTPS, and a Google Maps Platform project with Routes API enabled and billing configured.
- A base64-encoded 32-byte document-encryption key generated locally (for example, `openssl rand -base64 32`).

Do not put database URLs, API keys, passwords, or JWT secrets in source files or chat. Render creates the JWT secret; Neon provides the database connection string.

## 1. Create the Neon database

1. Create a Neon project in the nearest available region (Singapore is a reasonable choice for Bengaluru).
2. Create or select a PostgreSQL database.
3. Copy its connection string from the Neon dashboard. Use the connection string as provided, including `sslmode=require` or stronger. Keep it private.

## 2. Deploy the static site to Vercel

1. In Vercel, import the GitHub repository.
2. Set **Root Directory** to `frontend`; leave the framework preset as **Other** and do not override the build or output settings from [vercel.json](./frontend/vercel.json).
3. Deploy. The site can be opened before the API is configured, but API health will remain unavailable until the later steps.
4. Copy the production site origin, for example `https://your-project.vercel.app` (no path or trailing slash).

Set the Vercel environment variable `NIMMA_API_BASE` to the Render API origin followed by `/api/v1` after the API is deployed. The Vercel build writes this public URL to `deployment-config.js`; it is not a secret.

## 3. Deploy the Go API to Render

1. In Render, choose **New → Blueprint** and connect the same GitHub repository. Render will read the root `render.yaml`.
2. When prompted, set `DATABASE_URL` to the Neon connection string, and set `APP_CORS_ORIGINS` to the exact Vercel production origin copied above.
3. Set `OSRM_BASE_URL` to your controlled HTTPS OSRM endpoint, `GOOGLE_MAPS_API_KEY` to a server-side key restricted to the Routes API, and `DRIVER_DOCUMENTS_ENCRYPTION_KEY` to the base64 encryption key. Configure Google key restrictions and quotas in Google Cloud; never expose the Google key or document-encryption key in Vercel.
4. Create and deploy the Blueprint. The container applies database migrations and starts the API. Demo data is not seeded in production. Render generates `JWT_SECRET`; keep it private.
5. Copy the API's Render origin, for example `https://nimma-saarthi-api.onrender.com`.

### Create the first administrator

Production does not seed demo users. To bootstrap access, temporarily add `INITIAL_ADMIN_EMAIL`, `INITIAL_ADMIN_NAME`, and `INITIAL_ADMIN_PASSWORD` as private environment variables on the Render service, open its Shell, and run `/app/bootstrap-admin` once. The command creates an administrator only if none exists and refuses to overwrite an existing one. Remove all three bootstrap variables immediately after success, then sign in through the admin login page. Do not put the password in shell command arguments or source control.

## 4. Connect and redeploy the frontend

In Vercel project settings, add `NIMMA_API_BASE` for Production and Preview environments with the Render origin plus `/api/v1`, for example:

```text
https://nimma-saarthi-api.onrender.com/api/v1
```

Redeploy the frontend. Then visit the Vercel site and confirm the API status changes to connected.

## Operational constraints

- The Render service mounts persistent disk storage for driver verification documents; the API currently requires a single instance because Render disks are single-instance and do not provide shared storage.
- Migrations run at API startup. Demo seeding is intentionally disabled in production. Keep the Render instance count at one and use a separate database for development/demo data.
- SMS verification remains disabled until Twilio Verify credentials are added as Render environment variables.
- Use a Neon plan with backups, suitable connection limits, and a region appropriate to your users. Configure and test restore procedures before production use.
- Vercel preview deployments are not automatically allowed by the API CORS policy. Add an exact trusted origin to `APP_CORS_ORIGINS` in Render only if you intentionally need one.

Configure `APP_CORS_ORIGINS` with the exact HTTPS Vercel production origin(s), without wildcards or paths. Add an exact preview origin only if needed.

Google Routes transit itineraries are schedule estimates, not a guaranteed live train-arrival feed. They depend on Google's regional transit coverage and may include walking or bus legs. Results are fetched at request time; no stale schedule is substituted when the provider fails. Google requires an enabled Routes API project and billing. The API key remains server-side in Render.

Driver documents are encrypted with AES-256-GCM before being written to the persistent disk. Keep the encryption key in Render's secret environment, back it up separately, and do not rotate it without re-encrypting existing files. The disk still requires independent backups and retention controls. For multi-instance scaling, replace the local document store with a private shared object-storage provider before increasing the instance count. Card payments/payouts and transit-authority realtime feeds are not implemented.
