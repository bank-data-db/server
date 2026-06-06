-- name: MappingsExists :one
SELECT EXISTS(SELECT 1 FROM mappings WHERE author_id = $1 AND id = $2);

-- name: MappingsDeleteKeepingOrphans :execrows
DELETE FROM mappings WHERE author_id = $1 AND id = $2;

-- name: MappingsUnmapTransactions :many
WITH deleted AS (
    -- run the deletion
    DELETE FROM mapped_transactions
    WHERE mapping_id = $1
    RETURNING trans_id, updated_name
), flattened AS (
    -- flatten the fuckers into a single 'trans_id', 'did i update the name' 'did i update the category' table
    -- we COULD run a complex delete function here, but 
    SELECT
        trans_id,
        BOOL_OR(updated_name IS TRUE) AS up_name,
        BOOL_OR(updated_name IS FALSE) AS up_cat
    FROM deleted
    GROUP BY trans_id
)
SELECT t.id, card_id, description, amount, up_name, up_cat FROM transactions t JOIN flattened ON t.id = trans_id;

-- name: MappingsDeleteForCategoryDelete :exec
DELETE FROM mappings WHERE res_category = $1 AND res_name IS NULL;

-- name: MappingsTransactionCount :one
SELECT COUNT(DISTINCT trans_id) FROM mapped_transactions WHERE mapping_id = $1;

-- name: MappingsRemapExistingName :exec
UPDATE transactions SET resolved_name = $2 WHERE id = (
    SELECT trans_id FROM mapped_transactions
    WHERE mapping_id = $1 AND updated_name IS TRUE
);

-- name: MappingsRemapExistingCategoryID :exec
UPDATE transactions SET resolved_category = $2 WHERE id = (
    SELECT trans_id FROM mapped_transactions
    WHERE mapping_id = $1 AND updated_name IS FALSE
);
