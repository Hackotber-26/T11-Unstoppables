# Documentation Index

This directory contains the platform, product, architecture, operational, and algorithmic documentation for TrainSync.

---

## Documentation map

### Product and strategy
- [vision/executive-summary.md](vision/executive-summary.md) — executive overview and strategic rationale
- [vision/problem-statement.md](vision/problem-statement.md) — the transit-to-last-mile gap and opportunity
- [vision/product-vision.md](vision/product-vision.md) — product principles and feature intent
- [vision/business-model.md](vision/business-model.md) — revenue, cost, and unit economics
- [vision/roadmap.md](vision/roadmap.md) — phased product rollout plan

### Architecture
- [architecture/system-overview.md](architecture/system-overview.md) — system decomposition and platform boundaries
- [architecture/backend-architecture.md](architecture/backend-architecture.md) — Go backend and domain services
- [architecture/frontend-architecture.md](architecture/frontend-architecture.md) — rider, driver, admin, and public web experience
- [architecture/data-architecture.md](architecture/data-architecture.md) — persistent and derived data patterns
- [architecture/deployment-architecture.md](architecture/deployment-architecture.md) — environment, infrastructure, and release model
- [architecture/payment-architecture.md](architecture/payment-architecture.md) — wallet, pricing, and settlement architecture
- [architecture/safety-architecture.md](architecture/safety-architecture.md) — safety controls and trust-by-design
- [architecture/simulation-architecture.md](architecture/simulation-architecture.md) — simulation-first operational validation

### APIs and algorithms
- [api/README.md](api/README.md) — API documentation overview
- [algorithms/README.md](algorithms/README.md) — algorithmic design summary
- [algorithms/matching-pooling.md](algorithms/matching-pooling.md) — rider clustering and pool formation
- [algorithms/routing-eta.md](algorithms/routing-eta.md) — route optimization and arrival-time estimation

### Decisions and operating model
- [adr/README.md](adr/README.md) — architecture decision record index
- [adr/0001-simulation-first-architecture.md](adr/0001-simulation-first-architecture.md) — why simulation governs product decisions
- [simulation/README.md](simulation/README.md) — demand modeling and scenario framework
- [simulation/scenario-driven-operations.md](simulation/scenario-driven-operations.md) — operational validation workflow

### Developer and operations
- [developer/getting-started.md](developer/getting-started.md) — repository setup and developer workflow
- [operations/runbook.md](operations/runbook.md) — service operations, incidents, and monitoring
- [engineering/standards.md](engineering/standards.md) — engineering expectations and review standards
- [quality/test-strategy.md](quality/test-strategy.md) — QA and validation model
- [release/release-process.md](release/release-process.md) — staged rollout and release discipline

### Security and reliability
- [architecture/security-architecture.md](architecture/security-architecture.md) — identity, access, and trust boundaries
- [architecture/monitoring-observability.md](architecture/monitoring-observability.md) — metrics, dashboards, and alerting

---

## How to use this documentation

Use the documents in this order when onboarding:

1. Review the product vision and business model.
2. Understand the system overview and architecture layers.
3. Study the matching, routing, and simulation algorithms.
4. Use the developer and operational guides when implementing or running the system.
5. Refer to the ADRs when making architecture trade-offs.

---

## Philosophy

TrainSync is a transit-aware mobility system. The documentation reflects a system that treats transportation as an operational orchestration problem: predict demand, match journeys, assign supply, and optimize outcomes while preserving trust and safety.
