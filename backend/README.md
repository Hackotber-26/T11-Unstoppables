# Backend Overview

The backend is the operational core of TrainSync. It contains the Go services, domain modules, migrations, queries, and simulation logic needed to orchestrate transit-aware mobility.

## Local demo accounts

The development Docker Compose stack applies migrations and runs the idempotent demo seed before starting the API. The default local-only password is `NimmaDemo2026!` (override it with `DEMO_PASSWORD` in the Compose environment):

| Workspace | Email |
| --- | --- |
| Rider | `rider.demo@nimma.local` |
| Driver | `driver.demo@nimma.local` |
| Admin | `admin.demo@nimma.local` |

These are development credentials; do not use them in a deployed environment. Existing demo account passwords are not changed when the seed is rerun. Seeded data includes an INR wallet balance, a verified demo driver and vehicle, and sample trips.

---

## Responsibilities

- user and profile management
- rider, driver, and operator workflows
- trip lifecycle orchestration
- matching and pooling logic
- routing and ETA estimation
- pricing and wallet processing
- safety and incident handling
- analytics and reporting
- simulation and scenario experimentation

---

## Main directories

### cmd/
Entry points for executable services such as API, worker, simulator, seed, migration, and pooler flows.

### internal/
The domain-driven implementation of the platform, including auth, analytics, drivers, fares, trips, payments, matching, routing, stations, realtime, simulation, and safety modules.

### migrations/
Schema evolution and database versioning logic.

### queries/
Operational and reporting SQL used by the backend and data services.

### tests/
Test suites for integration and validation of service behavior.

---

## Typical runtime services

- API service for rider and operator interaction
- worker process for asynchronous jobs and task execution
- simulator for demand, trip, and fleet testing
- seed logic for initializing baseline scenario data
- pooler for grouping riders or trip candidates
- migration tooling for schema operations

The Docker Compose `migrate` service applies versioned schema changes before the API starts. Migration `000003_namma_metro_station_catalog.sql` seeds the operational Purple, Green, and Yellow line station catalog and its ordered stops. The API exposes the active catalog at `GET /api/v1/stations`, aggregate dashboard counts at `GET /api/v1/public/metrics`, and authenticated driver profile/availability routes.

## Driver operations and provider setup

- Phone verification uses Twilio Verify. Set `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN`, and `TWILIO_VERIFY_SERVICE_SID` in the local backend environment; the API keeps OTP verification unavailable unless all three are configured. Never commit or share the credentials.
- Rider transit planning uses Google Routes API transit itineraries. Configure `GOOGLE_MAPS_API_KEY` on the backend only, enable Routes API and billing in Google Cloud, and restrict the key to that API with quotas. `GET /api/v1/transit/route?originId=majestic&destinationId=indiranagar&departureTime=2026-04-04T10:00:00Z` is authenticated for riders and uses the station catalog. The response includes provider retrieval time; it is a schedule estimate, may include transfers, and is not guaranteed real-time arrival data. No schedule is fabricated or served from stale cache when Google has no transit leg or is unavailable.
- Driver documents (PDF, JPEG, or PNG; maximum 10 MB) are written to `DRIVER_DOCUMENTS_PATH`, outside the web root. The Compose stack mounts a private local volume. Production requires `DRIVER_DOCUMENTS_ENCRYPTION_KEY`: generate a 32-byte key encoded as base64 and keep it in the host's secret manager. Files use authenticated AES-256-GCM encryption at rest. Back up the key separately from the disk and configure retention controls. The current Render setup uses one persistent disk and one API instance; use private shared object storage before scaling horizontally.
- Driver GPS sharing requires explicit browser permission and the driver's active trip tracking action. Booking pickup/drop-off pins are routed by the backend OSRM service; fare quotes and saved trip distances are calculated from that road route, never from a rider-entered distance. For local demos, Compose defaults to the public OSRM demo server at `https://router.project-osrm.org`; route coordinates are sent to that third party, which may rate-limit or change availability, so do not use it for sensitive or production traffic. Set `OSRM_BASE_URL` to an OSRM instance you control for deployment. To run the optional local router, set `OSRM_BASE_URL=http://router:5000` and start `docker compose -f infrastructure/docker/docker-compose.yml --profile routing up --build -d`; its first start downloads Geofabrik's Southern Zone OpenStreetMap extract (which includes Bengaluru) and preprocesses it, requiring substantial disk space, memory, and CPU. Follow Geofabrik and OpenStreetMap attribution/licensing requirements. A pool route preview displays anonymous drop-off points and computes this rider's extra distance/time to their own stop.
- Subscription purchases debit the rider's INR wallet once and last 30 days. Cancellation stops the subscription immediately and is not refunded; discounted rides are limited by the plan's monthly allowance. No recurring payment processor is connected.
- Driver earnings are estimates derived from completed fares using a 15% platform commission. They are not a payout balance: bank transfers/payouts require a payment provider and are not implemented.

---

## Design principles

- domain modules remain explicit and separated
- matching and fare logic should be auditable
- simulation output must be treated as operational evidence
- realtime and persistence concerns remain distinct

## Queries and tests

Storage SQL is embedded from `backend/queries/*.sql`; tests assert that every SQL file is referenced by storage. PostgreSQL integration tests prepare the complete SQL catalog and exercise persisted account, trip, wallet, and matching operations.

From `backend/`:

```powershell
go test ./...
go test ./tests/load -bench=. -benchmem
```

Integration tests skip unless `TEST_DATABASE_URL` is set. Use only a disposable local/test database with project migrations applied:

```powershell
$env:TEST_DATABASE_URL = "postgres://postgres:local-postgres-password@localhost:5432/trainsync_test"
go test ./tests/integration -count=1 -v
```

Property and vector tests are deterministic. Load benchmark timings depend on the machine and are not release thresholds.
