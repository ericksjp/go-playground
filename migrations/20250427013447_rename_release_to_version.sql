-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
-- +goose StatementEnd

ALTER TABLE movies RENAME COLUMN "release" TO "version";

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';

ALTER TABLE movies RENAME COLUMN "version" TO "release";

-- +goose StatementEnd
