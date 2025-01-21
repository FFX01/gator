-- name: CreateFeedFollow :one
WITH inserted_follow AS (
    INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
    VALUES (
        $1,
        $2,
        $3,
        $4,
        $5
    )
    RETURNING *
)
SELECT
    inserted_follow.*,
    users.name AS user_name,
    feeds.name AS feed_name
FROM inserted_follow 
JOIN users 
    ON users.id = inserted_follow.user_id
JOIN feeds 
    ON feeds.id = inserted_follow.feed_id;

-- name: GetFeedFollowsForUser :many
SELECT
    feed_follows.*,
    feeds.name AS feed_name,
    users.name AS user_name
FROM feed_follows
JOIN feeds
    ON feeds.id = feed_follows.feed_id
JOIN users
    ON users.id = feed_follows.user_id
WHERE
    feed_follows.user_id = $1;

-- name: DeleteFeedFollow :exec
DELETE FROM feed_follows
WHERE
    user_id = $1
    AND feed_id = $2;
