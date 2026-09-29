#!/bin/sh
# Builds postgres/data.sql/fixtures.sql from a database dump (e.g. a staging one): restores it into a throwaway
# database in the docker-compose Postgres, anonymises it (postgres/anonymize.sql), migrates it to the current schema and
# dumps its data. Nothing is printed from the dump; the throwaway database is dropped at the end.
# Every fixture account's password is "pyrhouse"; the first active admin is called "admin".
# Usage, from the repository root: sh postgres/build-fixtures.sh <dump.sql>
set -eu

dump=$1
db=pyrhouse_fixtures_build
out=postgres/data.sql/fixtures.sql
port=${DB_PORT:-15432}

docker-compose up -d postgres >/dev/null 2>&1 || docker compose up -d postgres >/dev/null
container=$(docker-compose ps -q postgres 2>/dev/null || docker compose ps -q postgres)
until docker exec "$container" pg_isready -U postgres >/dev/null 2>&1; do sleep 1; done
sql() { docker exec -i "$container" psql -U postgres -v ON_ERROR_STOP=1 -qtA "$@" 2>/dev/null; }

sql -c "DROP DATABASE IF EXISTS $db"
# template0: an older cluster's template1 may refuse to be copied (collation version mismatch).
sql -c "CREATE DATABASE $db TEMPLATE template0"
trap 'sql -c "DROP DATABASE IF EXISTS $db" >/dev/null || true' EXIT

# The dump names its own roles (OWNER TO …), which don't exist here; those statements fail and nothing else should.
docker exec -i "$container" psql -U postgres -d "$db" -q < "$dump" >/dev/null 2>&1 || true
[ "$(sql -d "$db" -c 'SELECT count(*) FROM users')" -gt 0 ] || { echo "The dump restored no users." >&2; exit 1; }

sql -d "$db" < postgres/anonymize.sql
DATABASE_URL="postgres://postgres@localhost:$port/$db?sslmode=disable" go run ./main.go -migrate -dir=./migrations >/dev/null

tables=$(sql -d "$db" -c "SELECT string_agg(format('%I', tablename), ', ' ORDER BY tablename) FROM pg_tables WHERE schemaname = 'public' AND tablename <> 'schema_migrations'")
{
  echo "-- Fixtures: anonymised data from $(basename "$dump"), built by postgres/build-fixtures.sh on $(date +%Y-%m-%d)."
  echo "-- Load into a migrated database; it replaces whatever the tables hold. Passwords: \"pyrhouse\"."
  echo "TRUNCATE $tables RESTART IDENTITY CASCADE;"
  docker exec "$container" pg_dump -U postgres --data-only --disable-triggers --no-owner --exclude-table=schema_migrations "$db"
} > "$out"
echo "Wrote $out ($(grep -c '^COPY ' "$out") tables)."
