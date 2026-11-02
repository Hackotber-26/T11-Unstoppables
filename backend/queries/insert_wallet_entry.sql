INSERT INTO wallet_ledger_entries (user_id, trip_id, idempotency_key, amount_paise, entry_type, description)
VALUES ($1, $2, $3, $4, $5, $6);