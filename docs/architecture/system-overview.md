# System Overview

## Architecture philosophy

NimmaSaarthi is designed with a layered architecture that separates product experience from orchestration intelligence and physical deployment constraints.

The project follows a simulation-first pattern to validate behavior before scaling into live operations.

---

## Experience layer

This layer provides the user-facing interfaces:

- rider app for trip requests and live updates
- driver app for assignment and route execution
- operations dashboard for dispatch and monitoring
- admin dashboard for analytics and governance

The experience layer is intentionally lightweight and event-driven so that behavior is driven by shared platform logic instead of UI-only state.

---

## Intelligence layer

The intelligence layer is the core of the operating system.

### ETA engine
Predicts arrival and pickup timing using transit flow, path data, and live conditions.

### Matching engine
Clusters riders by origin, destination corridor, and route compatibility to build meaningful shared trips.

### Routing engine
Computes drop sequences, vehicle routing, and station-to-destination allocation.

### Fare engine
Applies pricing models based on contribution, distance, time, and pooling efficiency.

### Safety engine
Enforces policy logic for rider safety, trip review, issue tracking, and trust controls.

### Analytics engine
Measures occupancy, wait times, savings, and system health to support better decisions.

---

## Infrastructure layer

The platform works on a modern service pattern:

- Go backend services for domain logic and APIs
- PostgreSQL for transactional state and reporting
- Redis for caching, queue coordination, and realtime support
- WebSockets for live trip status and operational updates
- deployment, monitoring, and observability tooling through the infrastructure package

---

## Simulation layer

The simulation layer models mobility conditions before production deployment.

It includes:

- train and commuter event simulation
- rider generation and demand patterns
- driver availability simulation
- route and bay assignment scenarios
- performance analytics to test operating assumptions

---

## Architectural outcome

This layered design allows the project to treat transportation as a control system:

- demand is predicted
- supply is coordinated
- pricing reflects value delivered
- safety remains active at every step
- analytics improve the next operating cycle