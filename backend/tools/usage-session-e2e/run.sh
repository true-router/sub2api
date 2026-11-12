#!/bin/sh
set -eu
cd "$(dirname "$0")/../.."
fixture="sub2api-usage-session-e2e-$$"
cleanup() { docker rm -fv "$fixture" >/dev/null 2>&1 || true; }
trap cleanup EXIT HUP INT TERM
docker run -d --name "$fixture" -e POSTGRES_PASSWORD=session-e2e-only -e POSTGRES_DB=usage_session_e2e -p 127.0.0.1::5432 postgres:18-alpine >/dev/null
attempt=0
until docker exec "$fixture" pg_isready -U postgres -d usage_session_e2e >/dev/null 2>&1; do
  attempt=$((attempt + 1))
  [ "$attempt" -lt 60 ] || exit 1
  sleep 1
done
port=$(docker port "$fixture" 5432/tcp | sed 's/.*://')
export USAGE_SESSION_E2E_DSN="postgres://postgres:session-e2e-only@127.0.0.1:$port/usage_session_e2e?sslmode=disable"
go run ./tools/usage-session-e2e "$@"
