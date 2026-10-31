# API Documentation

The TrainSync API layer is responsible for rider actions, driver operations, trip lifecycle transitions, realtime coordination, and operational monitoring.

---

## API goals

- expose a consistent interface for transport orchestration
- separate operational domains such as rider, driver, trip, and payment flows
- support realtime updates for live fleet and trip status
- support auditability for incidents, wallet events, and pricing decisions

---

## Core API domains

### Authentication
- sign-in and onboarding flows
- session and token lifecycle
- consent and profile verification

### Rider API
- trip creation and pool request
- pickup and bay confirmation
- trip status monitoring
- fare and payment retrieval

### Driver API
- availability and shift status
- trip assignment and acceptance
- pickup, route, and completion updates
- earnings and trip history

### Operations API
- dashboard metrics and incident review
- fleet visibility and station health
- route and bay management
- safety and policy review

### Payment API
- wallet balance queries
- fare breakdowns
- settlement and ledger history
- subscription or commuter plans

---

## Communication patterns

### REST
Used for transactional actions such as user creation, trip requests, and admin queries.

### WebSocket / realtime
Used for route updates, driver assignment changes, rider trip events, and dashboard state changes.

### Event-driven updates
Operational state transitions are expected to emit events that support logging, analytics, and notification triggers.

---

## OpenAPI contract

The repository contains the OpenAPI specification in [openapi.yml](openapi.yml). That file is the canonical contract for the public API surface and should be kept aligned with backend behavior during development.

---

## Recommended API practices

- validate all request payloads before mutation
- treat rider, driver, and operator identities distinctly
- ensure payment amounts and trip states are auditable
- keep realtime events consistent with underlying persisted trip state
- use idempotent semantics where retries are likely

---

## Endpoint design principles

- descriptive resource paths
- explicit status handling for trip lifecycle transitions
- clear separation between read and mutation operations
- consistent error response formats
- minimal exposure of internal operational details to clients
