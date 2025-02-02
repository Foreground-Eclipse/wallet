BEGIN;
CREATE TABLE wallets (
    wallet_id UUID PRIMARY KEY,
    balance INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE operations (
    id UUID PRIMARY KEY,
    wallet_id UUID NOT NULL,
    type TEXT NOT NULL,
    amount INTEGER NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    FOREIGN KEY (wallet_id) REFERENCES wallets(wallet_id)
);
COMMIT;