-- +goose UP
CREATE TABLE feed_follows (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    user_id UUID NOT NULL references users(id) ON DELETE CASCADE,
    feed_id UUID NOT NULL references feeds(id) ON DELETE CASCADE,

    unique(user_id, feed_id)
);

-- +goose Down
DROP TABLE feed_follows;
