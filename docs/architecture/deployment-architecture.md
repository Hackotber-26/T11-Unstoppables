# Deployment Architecture

The repository is structured for deployment across multiple environments, with environment-specific logic separated under `deploy/` and `infrastructure/`.

---

## Environment model

- development environment for local iteration
- staging environment for validation and operational rehearsal
- production environment for live rider and driver operations

---

## Infrastructure components

### Containerized services
Docker-based deployment patterns support consistent environments for application and platform services.

### PostgreSQL
Runs as a persistent transactional database for production and operational state.

### Redis
Supports caching and event coordination for realtime transport operations.

### Monitoring stack
Prometheus and Grafana provide metrics and dashboarding for service health and operational patterns.

### Edge and proxying
Caddy is included as part of the platform infrastructure for routing and service access.

---

## Delivery model

The deployment strategy emphasizes:

- environment parity between local and cloud workflows
- service segregation for API, worker, and simulation jobs
- operational observability through metrics and dashboards
- safe rollout of new matching, routing, and pricing logic

---

## Production concerns

A production deployment should include:

- secure secret management
- explicit access policies and authorization boundaries
- error and latency monitoring
- backup and recovery planning
- safety and incident escalation workflows