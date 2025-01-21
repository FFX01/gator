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

-- name: GetAllFeeds :many
SELECT
    feeds.*,
    users.name AS user_name
FROM feeds
JOIN users
    ON users.id = feeds.user_id;

-- name: GetFeedByUrl :one
SELECT
    *
FROM feeds
WHERE
    url = $1;

-- name: MarkFeedFetched :one
UPDATE feeds
SET 
    last_fetched_at = $2,
    updated_at = $3
WHERE
    id = $1
RETURNING *;

-- name: GetNextFeedToFetch :one
SELECT
    *
FROM feeds
ORDER BY
    last_fetched_at ASC NULLS FIRST
LIMIT 1;
