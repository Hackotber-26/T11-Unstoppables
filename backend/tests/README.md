# Backend Test Tiers

- `internal/**`: focused unit tests for domain rules, auth, API middleware, and storage URL parsing.
- `queries/`: embedded SQL catalog completeness and storage-reference checks.
- `tests/integration/`: PostgreSQL query preparation and persisted rider/trip/wallet/matching workflows. Tests skip without `TEST_DATABASE_URL`; use a disposable test database only.
- `tests/property/`: generated fare and pooling invariants using Go's built-in `testing/quick`.
- `tests/vectors/`: synthetic graph contract checks for provenance, endpoints, and route shape.
- `tests/load/`: Go benchmarks for in-memory pool formation. Run explicitly with `go test ./tests/load -bench=. -benchmem`.

Run the deterministic suite with `go test ./...` from `backend/`. This compiles the benchmark package but does not execute benchmarks. Integration checks run only when the test database URL is present.