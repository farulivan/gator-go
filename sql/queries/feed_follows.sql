-- name: CreateFeedFollow :one
WITH inserted_feed_folow AS (
    INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
    VALUES ($1, $2, $3, $4, $5)
    RETURNING *
)
SELECT 
    inserted_feed_folow.*,
    feeds.name as feed_name,
    users.name as user_name
FROM inserted_feed_folow
JOIN feeds ON inserted_feed_folow.feed_id = feeds.id
JOIN users ON inserted_feed_folow.user_id = users.id;


-- name: GetFeedFollowsByUserID :many
SELECT 
    feed_follows.*,
    feeds.name AS feed_name,
    users.name AS user_name
FROM feed_follows
JOIN feeds ON feed_follows.feed_id = feeds.id
JOIN users ON feed_follows.user_id = users.id
WHERE feed_follows.user_id = $1;