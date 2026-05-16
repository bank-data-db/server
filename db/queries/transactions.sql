-- name: TransactionsInsert :copyfrom
INSERT INTO transactions (
    id, author_id, card_id,
    authed_at, settled_at,
    description, amount,
    resolved_name, resolved_category
) VALUES (
    $1, $2, $3,
    $4, $5,
    $6, $7,
    $8, $9
);

-- name: MappedTransactionsInsert :copyfrom
INSERT INTO mapped_transactions (
    trans_id, mapping_id, updated_name
) VALUES ($1, $2, $3);

-- name: TransactionsExistsNoID :one
SELECT EXISTS(
    SELECT 1 FROM transactions WHERE card_id = $1 AND authed_at = $2 AND settled_at = $3 AND description = $4 AND amount = $5
);

-- name: TransactionsExists :one
SELECT EXISTS(
    SELECT 1 FROM transactions WHERE id = $1 AND author_id = $2
);