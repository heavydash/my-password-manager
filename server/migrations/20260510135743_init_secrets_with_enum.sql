-- +goose Up
CREATE TYPE secret_type AS ENUM (
    'password',
    'note',
    'card',
    'ssh_key',
    'custom'
);

ALTER TABLE secrets ADD COLUMN type_new secret_type;

UPDATE secrets SET type_new = type::text::secret_type;

ALTER TABLE secrets DROP COLUMN type;
ALTER TABLE secrets RENAME COLUMN type_new TO type;


-- +goose Down
ALTER TABLE secrets ADD COLUMN type_old TEXT;
UPDATE secrets SET type_old = type::text;
ALTER TABLE secrets DROP COLUMN type;
ALTER TABLE secrets RENAME COLUMN type_old TO type;

DROP TYPE IF EXISTS secret_type;