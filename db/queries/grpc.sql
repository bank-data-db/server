-- name: CardsDelete :execrows
DELETE FROM cards WHERE user_id = $1 AND id = $2;

-- name: CardsUpdate :execrows
UPDATE cards SET name = $3 WHERE user_id = $1 AND id = $2;

-- name: CategoriesDelete :execrows
DELETE FROM categories WHERE author_id = $1 AND id = $2;

-- name: TransactionsDelete :execrows
DELETE FROM transactions WHERE author_id = $1 AND id = $2;

-- name: CategoriesExists :one
SELECT EXISTS (
    SELECT 1 FROM categories WHERE id = $1 AND author_id = $2
);
