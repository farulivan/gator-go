-- name: CreateFeedAndFollow :one
WITH new_feed AS (
    INSERT INTO feeds (id, created_at, updated_at, name, url, user_id)
    VALUES (
        @feed_id,
        @created_at,
        @updated_at,
        @name,
        @url,
        @user_id
    )
    RETURNING *
),
new_follow AS (
    INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
    SELECT @follow_id, new_feed.created_at, new_feed.updated_at, new_feed.user_id, new_feed.id
    FROM new_feed
)
SELECT * FROM new_feed;

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
