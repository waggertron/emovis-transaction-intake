CREATE TABLE IF NOT EXISTS transactions (
    id TEXT NOT NULL,
    source VARCHAR(64) NOT NULL,
    source_reference VARCHAR(128) NOT NULL,
    fingerprint VARCHAR(64) NOT NULL,
    payload JSONB NOT NULL,
    event_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE (source, source_reference),
    UNIQUE (event_id)
);

CREATE TABLE IF NOT EXISTS outbox_events (
    event_id TEXT PRIMARY KEY,
    event_payload JSONB NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'published', 'failed')),
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    retry_at TIMESTAMPTZ,
	lease_until TIMESTAMPTZ,
	claim_token TEXT,
    published_at TIMESTAMPTZ,
    last_error TEXT,
    CONSTRAINT outbox_transaction_event_fk FOREIGN KEY (event_id) REFERENCES transactions(event_id)
);

ALTER TABLE outbox_events ADD COLUMN IF NOT EXISTS claim_token TEXT;

CREATE INDEX IF NOT EXISTS outbox_dispatch_idx
    ON outbox_events (status, retry_at, lease_until, occurred_at)
    WHERE status = 'pending';
