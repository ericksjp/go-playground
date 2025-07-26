-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';

CREATE TABLE IF NOT EXISTS movies (
    id bigserial PRIMARY KEY,
    created_at timestamp (0) with time zone NOT NULL DEFAULT NOW(),
    title text NOT NULL,
    year integer NOT NULL,
    runtime integer NOT NULL,
    genres text [] NOT NULL,
    release integer NOT NULL DEFAULT 1

    CHECK (runtime >= 0),
    CHECK (year BETWEEN 1888 AND date_part('year', now())),
    CHECK (array_length(genres, 1) BETWEEN 1 AND 5)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';

DROP TABLE IF EXISTS movies;

-- +goose StatementEnd
