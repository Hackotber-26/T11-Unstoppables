ALTER TABLE trip_requests RENAME COLUMN fare_cents TO fare_paise;
ALTER TABLE wallet_ledger_entries RENAME COLUMN amount_cents TO amount_paise;

ALTER TABLE wallet_accounts
    ALTER COLUMN currency SET DEFAULT 'INR';

UPDATE wallet_accounts
SET currency = 'INR'
WHERE currency <> 'INR';

CREATE TABLE account_invitations (
    token_hash TEXT PRIMARY KEY,
    role TEXT NOT NULL CHECK (role IN ('driver', 'admin')),
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    redeemed_at TIMESTAMPTZ,
    CHECK (expires_at > created_at)
);

CREATE INDEX account_invitations_active_idx
    ON account_invitations (role, expires_at)
    WHERE redeemed_at IS NULL;
