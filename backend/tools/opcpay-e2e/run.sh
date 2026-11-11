#!/bin/sh
set -eu
cd "$(dirname "$0")/../.."
fixture="sub2api-opcpay-e2e-$$"
cleanup() { docker rm -fv "$fixture" >/dev/null 2>&1 || true; }
trap cleanup EXIT HUP INT TERM
docker run -d --name "$fixture" -e POSTGRES_PASSWORD=opcpay-e2e-fixture -e POSTGRES_DB=opcpay_e2e -p 127.0.0.1::5432 postgres:18-alpine >/dev/null
ready=0
while [ "$ready" -lt 60 ]; do
  if docker exec "$fixture" pg_isready -U postgres -d opcpay_e2e >/dev/null 2>&1; then break; fi
  ready=$((ready + 1))
  sleep 1
done
port=$(docker port "$fixture" 5432/tcp | sed 's/.*://')
export OPCPAY_E2E_DSN="postgres://postgres:opcpay-e2e-fixture@127.0.0.1:$port/opcpay_e2e?sslmode=disable"
go run ./tools/opcpay-e2e
