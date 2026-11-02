# Backend Architecture

The backend is implemented in Go and organized around operational domains rather than a single monolithic controller.

---

## Core technology stack

- Go modules and service-oriented architecture
- PostgreSQL as the canonical transactional store
- Redis for async coordination, caching, and realtime patterns
- REST APIs and realtime event delivery via sockets

---

## Domain modules

The repository identifies the following backend functional areas:

- authentication and identity
- analytics and reporting
- audit and event tracking
- bays and pickup operations
- driver and vehicle management
- fares and pricing logic
- incidents and safety workflows
- matching and dispatch orchestration
- metro and transit coordination
- notifications and communication
- payments and wallet processing
- profiles and user metadata
- realtime operations
- riders and trip planning
- routing and path optimization
- SafHER safety policies
- simulation and operational experimentation
- stations and trip origins
- subscriptions and recurring travel plans
- trips and ride lifecycle
- users and access control
- vehicles and fleet readiness
- wallet account ledger and transaction state

---

## Service boundaries

The backend is intended to separate responsibilities across domains so that business flows remain understandable and testable.

### API layer
Handles HTTP requests, authentication, and request validation.

### Domain services
Contain matching, routing, fare, trip, and payment work.

### Persistence layer
Manages database read/write operations, migrations, and query logic.

### Realtime layer
Publishes state changes and operational events to connected clients.

### Simulation layer
Provides scenario-driven analysis for system behavior and optimization.

---

## Operational flow

A typical trip lifecycle includes:

1. rider request and demand shaping
2. route and corridor detection
3. pool formation or matching
4. vehicle assignment and bay selection
5. trip execution and status updates
6. payment and settlement
7. incident handling and review
8. analytics record creation

---

## Design goals

- clear domain separation
- reliable event-driven updates
- safety-aware decision making
- testable operational flows
- predictable scaling for geographic and transit demand growth