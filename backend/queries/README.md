# Backend Queries

Every SQL statement executed by `backend/internal/storage` is kept here and embedded into the Go module. Files use PostgreSQL syntax and positional `$N` parameters; the storage layer owns transactions, scan types, validation, and error mapping.

Run `go test ./queries ./internal/storage` from `backend/` to verify the query catalog. The SQL remains hand-authored rather than generated; update a query file and the storage call together, then cover behavior in a unit or integration test.