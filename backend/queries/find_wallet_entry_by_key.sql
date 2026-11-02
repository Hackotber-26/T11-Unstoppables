SELECT user_id, amount_paise, entry_type, trip_id
FROM wallet_ledger_entries
WHERE idempotency_key = $1;