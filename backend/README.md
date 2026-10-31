# Backend Overview

The backend is the operational core of TrainSync. It contains the Go services, domain modules, migrations, queries, and simulation logic needed to orchestrate transit-aware mobility.

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
