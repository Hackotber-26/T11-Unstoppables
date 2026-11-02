INSERT INTO audit_events (id, actor_id, action, resource, resource_id, metadata, occurred_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);