SELECT id, user_id, COALESCE(trip_id::text, ''), idempotency_key,
    amount_paise, entry_type, description, created_at
FROM wallet_ledger_entries
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2;