#!/usr/bin/env sh
# Isolated Docker deployment smoke test; never uses the production database/key.
set -eu
cd "$(dirname "$0")/.."
project=miva-deploy-check
if docker container inspect "$project-db" >/dev/null 2>&1; then
  echo 'A deployment smoke test is already running.' >&2
  exit 1
fi
work_dir=$(mktemp -d /tmp/miva-deploy-check.XXXXXX)
cleanup() {
  docker compose --project-name "$project" --env-file "$work_dir/.env" -f compose.vps.yaml -f deploy/compose.database-network.yaml down --remove-orphans >/dev/null 2>&1 || true
  docker rm -fv "$project-db" >/dev/null 2>&1 || true
  docker network rm "$project-database" >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM
cat > "$work_dir/.env" <<'ENV'
APP_ORIGIN=http://127.0.0.1:5191
COOKIE_SECURE=false
ALLOW_REGISTRATION=true
DB_HOST=db
DB_PORT=5432
DB_NAME=miva_test
DB_USER=english_app
DB_PASSWORD=smoke-test-only-password
DB_NETWORK=miva-deploy-check-database
DB_SSLMODE=disable
FRONTEND_BIND=127.0.0.1
FRONTEND_PORT=5191
POSTGRES_TOOLS_IMAGE=postgres:18-alpine
AI_BASE_URL=https://member.vpsindonesia.co.id/api/v1
AI_API_KEY=
AI_MODEL=sonnet-5
AI_API_FORMAT=chat_completions
AI_JSON_MODE=prompt
ENV
chmod 600 "$work_dir/.env"
python3 -m unittest discover -s scripts/tests -v
docker network create "$project-database" >/dev/null
docker run -d --name "$project-db" --network "$project-database" --network-alias db --memory 256m -e POSTGRES_USER=qa_admin -e POSTGRES_DB=miva_test -e POSTGRES_PASSWORD=smoke-test-admin-password postgres:18-alpine -c shared_buffers=16MB -c max_connections=15 >/dev/null
for attempt in $(seq 1 60); do
  if docker exec "$project-db" pg_isready -U qa_admin -d miva_test >/dev/null 2>&1; then break; fi
  sleep 1
done
docker exec "$project-db" psql -U qa_admin -d miva_test -v ON_ERROR_STOP=1 -c "CREATE ROLE english_app LOGIN PASSWORD 'smoke-test-only-password' CONNECTION LIMIT 5; CREATE SCHEMA english_app AUTHORIZATION english_app;" >/dev/null
python3 scripts/deploy.py --env-file "$work_dir/.env" --project-name "$project" --release smoke-test --backups-dir "$work_dir/backups" --lock-file "$work_dir/deploy.lock"
# A second deploy must skip applied migrations and create a usable backup.
python3 scripts/deploy.py --env-file "$work_dir/.env" --project-name "$project" --release smoke-test-repeat --backups-dir "$work_dir/backups" --lock-file "$work_dir/deploy.lock"
docker exec "$project-db" psql -U qa_admin -d miva_test -v ON_ERROR_STOP=1 -tAc 'SELECT count(*) FROM english_app.schema_migrations' | tr -d ' ' | rg '^2$'
python3 - "$work_dir" <<'PY'
import json, sys, urllib.request, urllib.error
from pathlib import Path
root = Path(sys.argv[1])
assert (root / 'backups/smoke-test-repeat.dump').read_bytes().startswith(b'PGDMP')
base = 'http://127.0.0.1:5191'
assert json.load(urllib.request.urlopen(base + '/healthz'))['status'] == 'ok'
assert b'<title>Miva' in urllib.request.urlopen(base + '/').read()
assert b'<title>Miva' in urllib.request.urlopen(base + '/study').read()
try:
    urllib.request.urlopen(base + '/assets/missing.js')
    raise AssertionError('missing asset returned a SPA page')
except urllib.error.HTTPError as error:
    assert error.code == 404
request = urllib.request.Request(base + '/api/auth/register', data=json.dumps({'email':'smoke@example.test','password':'Smoke-test-password-123!','name':'Smoke Test'}).encode(), headers={'Content-Type':'application/json','Origin':base}, method='POST')
with urllib.request.urlopen(request) as response:
    assert json.load(response)['user']['name'] == 'Smoke Test'
    assert 'HttpOnly' in response.headers['Set-Cookie']
print('PASS: split frontend/backend, fresh migrations, repeated deploy, backup, API proxy, cookie and SPA routing.')
PY
