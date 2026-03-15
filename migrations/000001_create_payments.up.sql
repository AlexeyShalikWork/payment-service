CREATE TABLE payments (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL,
    amount bigint NOT NULL,
    currency text NOT NULL,
    status text NOT NULL,
    created_at timestamptz DEFAULT now()
);
