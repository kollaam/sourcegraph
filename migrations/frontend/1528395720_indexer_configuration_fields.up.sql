BEGIN;

DROP VIEW lsif_indexes_with_repository_name;

ALTER TABLE lsif_indexes
    ADD COLUMN install_image text,
    ADD COLUMN install_commands text[],
    ADD COLUMN root text,
    ADD COLUMN indexer text,
    ADD COLUMN arguments text[];

UPDATE lsif_indexes SET
    install_image = '',
    install_commands = '{}'::text[],
    root = '',
    indexer = 'sourcegraph/lsif-go:latest',
    arguments = '{}'::text[];

ALTER TABLE lsif_indexes
    ALTER COLUMN install_image SET NOT NULL,
    ALTER COLUMN install_commands SET NOT NULL,
    ALTER COLUMN root SET NOT NULL,
    ALTER COLUMN indexer SET NOT NULL,
    ALTER COLUMN arguments SET NOT NULL;

CREATE VIEW lsif_indexes_with_repository_name AS
    SELECT u.*, r.name as repository_name FROM lsif_indexes u
    JOIN repo r ON r.id = u.repository_id
    WHERE r.deleted_at IS NULL;

COMMIT;
