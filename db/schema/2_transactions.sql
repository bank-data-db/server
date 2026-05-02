CREATE TABLE IF NOT EXISTS checkpoints (
    created_at DATE UNIQUE,
    card_id TEXT NOT NULL,
    amount DECIMAL(10, 2)
);

CREATE TABLE IF NOT EXISTS categories (
    id TEXT PRIMARY KEY,
    author_id TEXT NOT NULL REFERENCES users(id),

    name TEXT NOT NULL,
    color TEXT NOT NULL,
    -- Icon is 1 character,
    -- BUT can be multiple unicode segments
    icon TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS transactions (
    id TEXT PRIMARY KEY,
    author_id TEXT NOT NULL REFERENCES users(id),
    card_id TEXT NOT NULL REFERENCES users(id),

    settled_at TIMESTAMPTZ NOT NULL,
    authed_at TIMESTAMPTZ NOT NULL,

    description TEXT NOT NULL,
    -- I fucking hate the money type... no support for it in pgx or sqlc AT ALL WTF
    amount NUMERIC(8,2) NOT NULL,

    resolved_name TEXT,
    resolved_category TEXT REFERENCES categories(id)
);

CREATE INDEX IF NOT EXISTS idx_trans_authed_at ON transactions(authed_at);
CREATE INDEX IF NOT EXISTS idx_trans_search_terms ON transactions(description, amount);

-- A set of rules to match against in order to automatically figure out a transaction name & category
CREATE TABLE IF NOT EXISTS mappings (
    id TEXT PRIMARY KEY,
    author_id TEXT NOT NULL REFERENCES users(id),

    name TEXT NOT NULL,
    -- transaction details 
    trans_text   TEXT, -- regex <3
    trans_amount NUMERIC(8,2),
    -- resulting data
    res_name     TEXT,
    res_category TEXT REFERENCES categories(id),
    -- extra :3
    priority   INTEGER DEFAULT 0 NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_mappings_text ON mappings (trans_text);
CREATE INDEX IF NOT EXISTS idx_mappings_amount ON mappings (trans_amount);

CREATE TABLE IF NOT EXISTS mapped_transactions (
    trans_id TEXT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    mapping_id TEXT NOT NULL REFERENCES mappings(id) ON DELETE CASCADE,
    updated_name BOOLEAN NOT NULL
)
