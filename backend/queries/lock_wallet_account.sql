SELECT user_id
FROM wallet_accounts
WHERE user_id = $1
FOR UPDATE;