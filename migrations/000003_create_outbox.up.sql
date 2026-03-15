CREATE TABLE outbox (
    id bigserial PRIMARY KEY,
    topic text NOT NULL,
    event_type text NOT NULL,
    payload jsonb NOT NULL,
    published boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_outbox_unpublished ON outbox (id) WHERE published = false;
