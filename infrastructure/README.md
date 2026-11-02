# Infrastructure Overview

The infrastructure layer defines the runtime environment and operational foundation for the TrainSync platform.

---

## Included infrastructure

- Caddy for edge routing and proxying
- Docker-based environment packaging
- PostgreSQL deployment configuration
- Redis coordination and cache configuration
- Prometheus for metrics collection
- Grafana for dashboards and observability
- monitoring and support tooling for production-grade operations

---

## Responsibilities

- environment parity across development, staging, and production
- persistent storage and cache configuration
- policy-aware service execution
- platform observability and operational control
- deployment orchestration support for app services and workers

---

## Operational view

The infrastructure layer is the backbone that enables:

- service uptime and health checks
- structured telemetry collection
- environment reproducibility
- safe scaling and rollout management

---

## Design principle

Infrastructure should support the product rather than hide operational complexity. The platform should remain observable, traceable, and configurable across environments.

## Local stack

Run from `infrastructure/docker/`:

```powershell
Copy-Item .env.example .env
docker compose config
docker compose up --build -d
```

The stack binds host ports to loopback only. Defaults are API `8080`, Prometheus `9090`, Grafana `3001`, Caddy HTTP `8081`, Caddy HTTPS `8444`, PostgreSQL `5432`, and Redis `6379`; override them in `.env` if occupied. Redis requires a password even in local Compose. The API waits for PostgreSQL migrations and Redis health before starting.

Grafana is available at `http://localhost:3001`; its dashboard is provisioned from the repository. Local Grafana and database passwords are development defaults only. Set unique secrets and a real hostname/TLS policy before any shared or production deployment; do not commit `.env` files.

## Backup

With the stack running, create a custom-format PostgreSQL dump from the repository root:

```powershell
./infrastructure/postgres/backup.ps1 -OutputDirectory ./infrastructure/postgres/backups
```

The PostgreSQL schema is owned by `backend/migrations/` and applied by the one-shot `migrate` service; there is no parallel init schema. With no override, backups go to the ignored `infrastructure/postgres/backups/` directory.

Restore is destructive and requires PowerShell confirmation:

```powershell
./infrastructure/postgres/restore.ps1 -BackupPath ./infrastructure/postgres/backups/trainsync-YYYYMMDD-HHMMSS.dump
```

Backups contain sensitive operational data. Store them encrypted with access controls and test restoration before relying on them. No hosted or paid APIs are required by this stack.
