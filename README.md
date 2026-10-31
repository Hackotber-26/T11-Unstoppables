# T11-Unstoppables

## NimmaSaarthi: Metro-Aware Mobility Orchestration

NimmaSaarthi is a mobility intelligence platform designed to solve the last-mile problem at metro stations and high-density transit nodes. Instead of asking riders to search for transport after they exit a station, the platform predicts demand, pools commuters in advance, assigns vehicles dynamically, and coordinates the full journey from station exit to final destination.

The repository combines a Go-based backend, a frontend experience layer, simulation tooling, analytics, and operational infrastructure for a complete urban mobility product concept.

---

## Why this project exists

Modern mass transit is efficient at moving large numbers of people across a city, but the final leg of the journey is often fragmented, expensive, and unreliable. Riders face long waits, unclear pricing, inconsistent safety, and poor coordination. Drivers sit idle between trips. Cities absorb additional congestion because empty or partially occupied vehicles are dispatched after demand appears.

NimmaSaarthi flips the model:

- predict rider demand before arrival
- form effective shared trips by corridor, destination cluster, and arrival timing
- assign vehicles and bays in advance
- use pricing, safety checks, and live ETA updates to reduce friction
- operate a closed-loop system that learns from trip patterns and operational data

---

## Product principles

### Match journeys, not destinations
Riders flowing from the same station or corridor are grouped based on route alignment rather than identical final destinations.

### Simulate before scale
The project includes training, simulator, and scenario tooling so operating decisions can be tested against real-world patterns before deployment.

### Safety by design
SafHER and related safety logic are built into the experience from planning to trip completion.

### Shared mobility economics
Fare calculation, driver incentives, and wallet operations reward pooling and efficient fleet use.

---

## Core capabilities

- Rider onboarding, trip requests, and matching
- Driver availability, assignment, and trip execution
- Pool formation for station-to-destination corridors
- Dynamic routing and fare computation
- Bay assignment and pickup orchestration
- Real-time notifications and event streams
- Payments, subscriptions, and wallet services
- Safety incident handling and auditing
- Simulation and analytics for optimization

---

## System overview

The solution is layered to separate experience, intelligence, infrastructure, and simulation concerns.

### Experience layer
- rider app
- driver app
- admin dashboard
- operations dashboard

### Intelligence layer
- ETA engine
- matching engine
- routing engine
- fare engine
- safety engine
- analytics engine

### Infrastructure layer
- Go backend services
- PostgreSQL persistence
- Redis cache and queue patterns
- WebSocket realtime updates
- monitoring and deployment tooling

### Simulation layer
- scenario and synthetic datasets
- train and rider simulators
- driver activity simulation
- analytics and optimization testing

---

## Repository structure

```text
.
├── backend/                # Go services, APIs, workers, simulation runtime
│   ├── cmd/
│   ├── internal/
│   ├── migrations/
│   ├── queries/
│   └── tests/
├── frontend/               # rider, driver, admin, public experience layer
│   ├── pages/
│   ├── public/
│   ├── scripts/
│   ├── styles/
│   └── tests/
├── docs/                   # product, architecture, and strategy documentation
│   ├── architecture/
│   └── vision/
├── infrastructure/         # docker, grafana, prometheus, redis, postgres configs
├── datasets/               # routes, scenarios, stations, verified and synthetic data
├── deploy/                 # deployment manifests and environment configs
├── spec/                   # tariff, scenario, config, and vector specification files
├── research/               # experiments, exports, notebooks
├── tools/                  # benchmarking, migration, scenario and vector tooling
├── README.md
├── CHANGELOG.md
├── CONTRIBUTING.md
├── CODE_OF_CONDUCT.md
├── SECURITY.md
├── DISCLAIMER.md
├── ROADMAP.md
└── LICENSE
```

---

## Key technologies

- Go for backend services and orchestration logic
- PostgreSQL for transactional data and operational records
- Redis for caching and realtime coordination
- Browser-based frontend using HTML, CSS, JavaScript, Canvas, SVG, Web Workers
- Simulation tooling for demand, fleet, and system testing
- Containerized infrastructure for deployable environments

---

## Getting started

### Backend

```bash
cd backend
go mod download
go run ./cmd/api
```

For worker and simulator flows:

```bash
cd backend
go run ./cmd/worker
go run ./cmd/simulator
```

### Frontend

The frontend is a static browser client connected to the Go API. Start the local backend and dependencies first:

```bash
docker compose -f infrastructure/docker/docker-compose.yml up --build -d
```

Then serve the UI:

```bash
cd frontend
python -m http.server 8000
```

Open http://localhost:8000. The backend allows `http://localhost:8000` and `http://127.0.0.1:8000` by default; configure `APP_CORS_ORIGINS` for another local origin. See [frontend/README.md](frontend/README.md) for connected workflows and prototype limitations.

### Data and simulation

The project includes scenario and station data under `datasets/`, plus simulation logic under `backend/internal/simulation` and `backend/cmd/simulator`.

---

## Documentation map

- [docs/vision/executive-summary.md](docs/vision/executive-summary.md) — strategic overview
- [docs/vision/problem-statement.md](docs/vision/problem-statement.md) — mobility pain points and opportunity
- [docs/vision/product-vision.md](docs/vision/product-vision.md) — product philosophy and feature intent
- [docs/vision/business-model.md](docs/vision/business-model.md) — revenue and unit economics
- [docs/vision/roadmap.md](docs/vision/roadmap.md) — milestones and delivery phases
- [docs/architecture/system-overview.md](docs/architecture/system-overview.md) — architecture framing
- [docs/architecture/backend-architecture.md](docs/architecture/backend-architecture.md) — Go service architecture
- [docs/architecture/frontend-architecture.md](docs/architecture/frontend-architecture.md) — UI architecture
- [docs/architecture/data-architecture.md](docs/architecture/data-architecture.md) — data model and storage patterns
- [docs/architecture/deployment-architecture.md](docs/architecture/deployment-architecture.md) — environment and delivery model
- [docs/architecture/payment-architecture.md](docs/architecture/payment-architecture.md) — wallet and settlement model
- [docs/architecture/safety-architecture.md](docs/architecture/safety-architecture.md) — trust and safety controls
- [docs/architecture/security-architecture.md](docs/architecture/security-architecture.md) — security model and trust boundaries
- [docs/architecture/monitoring-observability.md](docs/architecture/monitoring-observability.md) — observability and incident readiness
- [docs/architecture/simulation-architecture.md](docs/architecture/simulation-architecture.md) — simulation design
- [docs/engineering/standards.md](docs/engineering/standards.md) — engineering standards and implementation expectations
- [docs/quality/test-strategy.md](docs/quality/test-strategy.md) — quality, testing, and validation strategy
- [docs/release/release-process.md](docs/release/release-process.md) — release process and rollout discipline
- [docs/operations/runbook.md](docs/operations/runbook.md) — operational playbook and incident response

---

## Contribution guidance

Please read [CONTRIBUTING.md](CONTRIBUTING.md) and the project standards before opening pull requests.

---

## Project status

This repository is best understood as a full-stack mobility platform prototype and simulation environment. It is structured for product design, backend engineering, operational planning, and experimentation with transit-aware transport coordination.

---

## License

This project is licensed under the terms in [LICENSE](LICENSE).