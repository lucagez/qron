-- +goose Up
-- +goose StatementBegin
alter table tiny.job add column deduplication_key text;
alter table tiny.job add constraint job_deduplication_key_constraint unique (deduplication_key);

create index job_deduplication_key_idx on tiny.job (deduplication_key);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
alter table tiny.job drop column deduplication_key;
-- +goose StatementEnd
