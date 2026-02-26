CREATE UNLOGGED TABLE IF NOT EXISTS stg_users (
    job_id UUID NOT NULL,
    row_index BIGINT NOT NULL,
    external_id TEXT,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    phone_number TEXT NOT NULL
);

CREATE UNLOGGED TABLE IF NOT EXISTS stg_addresses (
    job_id UUID NOT NULL,
    row_index BIGINT NOT NULL,
    user_external_id TEXT,
    user_email TEXT NOT NULL,
    street TEXT NOT NULL,
    city TEXT NOT NULL,
    state TEXT NOT NULL,
    zip_code TEXT NOT NULL,
    country TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_stg_users_job_id ON stg_users (job_id);
CREATE INDEX IF NOT EXISTS idx_stg_addresses_job_id ON stg_addresses (job_id);
