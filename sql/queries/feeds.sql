-- name: CreateFeed :one
INSERT INTO feeds (id, created_at, updated_at, name, url, user_id)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING *;

-- name: GetFeedsWithUsers :many
SELECT f.*, u.name as user_name FROM feeds f
JOIN users u ON f.user_id = u.id;

-- name: GetFeedByURL :one
SELECT * FROM feeds WHERE url = $1;

-- name: MarkFeedAsFetched :exec
UPDATE feeds
SET last_fetched_at = @at,
    updated_at = @at
WHERE id = @id;

-- name: GetNextFeedToFetch :one
SELECT * FROM feeds
ORDER BY last_fetched_at ASC NULLS FIRST
LIMIT 1;
