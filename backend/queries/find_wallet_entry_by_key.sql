SELECT user_id, amount_cents, entry_type, trip_id
FROM wallet_ledger_entries
WHERE idempotency_key = $1;