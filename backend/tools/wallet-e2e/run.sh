#!/bin/sh
set -eu
cd "$(dirname "$0")/../.."
fixture="sub2api-wallet-e2e-$$"
cleanup() { docker rm -fv "$fixture" >/dev/null 2>&1 || true; }
trap cleanup EXIT HUP INT TERM
docker run -d --name "$fixture" -e POSTGRES_PASSWORD=wallet-e2e-fixture -e POSTGRES_DB=wallet_e2e -p 127.0.0.1::5432 postgres:18-alpine >/dev/null
ready=0
while [ "$ready" -lt 60 ]; do
  if docker exec "$fixture" pg_isready -U postgres -d wallet_e2e >/dev/null 2>&1; then break; fi
  ready=$((ready + 1))
  sleep 1
done
port=$(docker port "$fixture" 5432/tcp | sed 's/.*://')
export WALLET_E2E_DSN="postgres://postgres:wallet-e2e-fixture@127.0.0.1:$port/wallet_e2e?sslmode=disable"
go run ./tools/wallet-e2e
