-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
-- +goose StatementEnd

CREATE INDEX IF NOT EXISTS movies_title_idx ON movies USING GIN(to_tsvector('simple', title));
CREATE INDEX IF NOT EXISTS movies_genre_idx ON movies USING GIN(genres);

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd

DROP INDEX IF EXISTS movies_title_idx;
DROP INDEX IF EXISTS movies_genre_idx;
