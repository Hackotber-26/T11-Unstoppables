# Developer Getting Started Guide

This guide helps engineers and contributors understand how to work effectively with the TrainSync codebase.

---

## Repository overview

The project combines:

- backend services in Go
- frontend experience flows in browser-based assets
- simulation and dataset tooling
- deployment and infrastructure definitions
- product and architecture documentation

---

## Recommended setup

### Prerequisites

- Go toolchain
- a PostgreSQL-compatible database for storage work
- Redis for caching and realtime coordination patterns
- a local static server for frontend testing
- a working terminal with access to the repository root

### Backend setup

```bash
cd backend
go mod download
```

Common runtime entry points include:

```bash
cd backend
go run ./cmd/api
go run ./cmd/worker
go run ./cmd/simulator
```

### Frontend setup

```bash
cd frontend
python -m http.server 8000
```

Then open the app at:

```text
http://localhost:8000
```

---

## Codebase navigation

### backend/
Contains API services, domain modules, migrations, and simulation logic.

### frontend/
Contains experiences for rider, driver, admin, and public flows.

### docs/
Contains product, architecture, simulation, and decision documentation.

### datasets/
Contains route, station, scenario, and synthetic data assets.

### infrastructure/
Contains runtime and monitoring configuration.

---

## Working conventions

- keep domain modules isolated and explicit
- avoid mixing realtime coordination logic with presentation logic
- document assumptions around fare and safety calculations
- prefer scenario-driven validation for mobility changes
- keep code and documentation aligned when product behavior shifts

---

## Pull request checklist

Before opening a PR, confirm that:

- the scope is clearly defined
- relevant validation steps were run
- the change does not conflict with the simulation-first operating model
- documentation is updated if product or architecture behavior changes
- safety or payment impacts have been considered

---

## Typical workflows

### Implementing a service change
1. read the relevant domain architecture documentation
2. validate the behavior with the narrowest relevant local test or simulation
3. update analytics, docs, and any affected API contract if needed

### Tuning a matching or routing rule
1. review the algorithm-specific docs
2. run a scenario-based simulation
3. compare performance against baseline metrics
4. document the decision and trade-offs
