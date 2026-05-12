CREATE TABLE IF NOT EXISTS checkpoints (
    created_at DATE,
    card_id TEXT NOT NULL,
    amount DECIMAL(10, 2)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_uniq_checkpoint ON checkpoints (created_at, card_id);

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
    resolved_category TEXT REFERENCES categories(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_trans_authed_at ON transactions(authed_at);
CREATE INDEX IF NOT EXISTS idx_trans_search_terms ON transactions(description, amount);

-- A set of rules to match against in order to automatically figure out a transaction name & category
CREATE TABLE IF NOT EXISTS mappings (
    id TEXT PRIMARY KEY,
    author_id TEXT NOT NULL REFERENCES users(id),

    name TEXT NOT NULL,
    -- transaction details 
    match_text           TEXT, -- regex <3
    match_amount         NUMERIC(8,2),
    match_amount_matcher CHAR,
    match_card_id        TEXT,
    -- resulting data
    res_name     TEXT,
    res_category TEXT REFERENCES categories(id) ON DELETE SET NULL,
    -- extra :3
    priority   INTEGER DEFAULT 0 NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_mappings_text ON mappings (trans_text);
CREATE INDEX IF NOT EXISTS idx_mappings_amount ON mappings (trans_amount);

CREATE TABLE IF NOT EXISTS mapped_transactions (
    trans_id TEXT NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    mapping_id TEXT NOT NULL REFERENCES mappings(id) ON DELETE CASCADE,
    updated_name BOOLEAN NOT NULL
);
