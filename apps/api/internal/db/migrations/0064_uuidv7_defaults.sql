-- +goose Up
-- Time-ordered primary keys (UUIDv7, RFC 9562, PostgreSQL 18): new rows append to the right edge of their
-- index. Existing rows keep their version 4 ids; only the defaults change.
-- +goose StatementBegin
DO $mig$
DECLARE c record;
BEGIN
    FOR c IN
        SELECT n.nspname AS sch, cl.relname AS tbl
        FROM pg_attrdef d
        JOIN pg_class cl ON cl.oid = d.adrelid
        JOIN pg_namespace n ON n.oid = cl.relnamespace
        JOIN pg_attribute a ON a.attrelid = d.adrelid AND a.attnum = d.adnum
        WHERE a.attname = 'id'
          AND pg_get_expr(d.adbin, d.adrelid) = 'gen_random_uuid()'
          AND cl.relkind IN ('r', 'p')
          AND n.nspname NOT IN ('pg_catalog', 'information_schema')
    LOOP
        EXECUTE format('ALTER TABLE %I.%I ALTER COLUMN id SET DEFAULT uuidv7()', c.sch, c.tbl);
    END LOOP;
END
$mig$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $mig$
DECLARE c record;
BEGIN
    FOR c IN
        SELECT n.nspname AS sch, cl.relname AS tbl
        FROM pg_attrdef d
        JOIN pg_class cl ON cl.oid = d.adrelid
        JOIN pg_namespace n ON n.oid = cl.relnamespace
        JOIN pg_attribute a ON a.attrelid = d.adrelid AND a.attnum = d.adnum
        WHERE a.attname = 'id'
          AND pg_get_expr(d.adbin, d.adrelid) = 'uuidv7()'
          AND cl.relkind IN ('r', 'p')
          AND n.nspname NOT IN ('pg_catalog', 'information_schema')
    LOOP
        EXECUTE format('ALTER TABLE %I.%I ALTER COLUMN id SET DEFAULT gen_random_uuid()', c.sch, c.tbl);
    END LOOP;
END
$mig$;
-- +goose StatementEnd
