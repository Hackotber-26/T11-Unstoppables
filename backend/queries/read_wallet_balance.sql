SELECT COALESCE(SUM(amount_paise), 0)
FROM wallet_ledger_entries
WHERE user_id = $1;