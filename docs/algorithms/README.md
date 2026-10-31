# Algorithms Overview

TrainSync depends on a set of coordinated algorithms that turn commuter demand, station timing, routing, and vehicle status into usable trip decisions.

---

## Core algorithm families

### Matching and grouping
Used to determine which riders can be pooled together and how drivers should serve them.

### Routing and sequence optimization
Determines the most efficient route or drop order for a group of riders.

### ETA prediction
Estimates when a rider, vehicle, or pool will arrive at the station, bay, or destination.

### Pricing and fairness allocation
Allocates the trip price among riders based on route contribution, time, and occupancy.

### Safety and trust policies
Determines when a trip should be restricted, escalated, or reviewed.

---

## Core decision loop

1. ingest station, route, and rider demand signals
2. predict vehicle and rider timing
3. cluster compatible demand into pooled trips
4. assign a vehicle with suitable capacity and route fit
5. calculate fair pricing and expected wait times
6. monitor trip execution and update analytics for future optimization

---

## Data inputs

- rider origin and destination corridor
- station queue and arrival timing
- vehicle location, availability, and capacity
- path and network constraints
- trip penalties such as walk distance or detour length
- safety and trust requirements

---

## Optimization goals

The algorithms are designed to improve:

- trip occupancy and utilization
- pickup reliability
- fairness and rider savings
- driver revenue per hour
- city-level congestion and empty vehicle movement

---

## Key implementation focus

The repository includes simulation, matching, and operational structures for these algorithms, and they are expected to be evaluated with real scenario data before production rollout.

## JavaScript reference engines

Import the supported domain functions from `frontend/scripts/engine/index.mjs`.

- `frontend/scripts/engine/matching.mjs` ranks candidates and rejects safety, capacity, wait, detour, or pickup-window violations.
- `frontend/scripts/engine/routing.mjs` computes a minimum-time directed route and a bounded ETA from supplied network edges.
- `frontend/scripts/engine/fare.mjs` calculates fares in integer cents to avoid floating-point currency drift.
- `frontend/scripts/simulation/scenario.mjs` generates repeatable synthetic demand and groups riders by corridor and pickup window.
- `frontend/scripts/payments/lifecycle.mjs` validates payment state transitions; `frontend/scripts/payments/ledger.mjs` calculates a balance from append-only entries.
- `frontend/tests/unit/engines.test.mjs` contains Node's built-in unit tests for these modules.

These modules are deterministic domain logic, not production integrations. Authentication, payment authorization/capture, ledger persistence, concurrency control, and payment-provider webhooks must be enforced by trusted backend services. No API credentials belong in browser code. A payment processor cannot be selected or connected until the operator chooses a self-hosted gateway and supplies a supported acquiring configuration; a local ledger does not move real funds. Matching weights, fare rates, and simulation assumptions require calibration against verified project data before operational use.
