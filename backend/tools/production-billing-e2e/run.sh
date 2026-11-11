#!/bin/sh
set -eu
cd "$(dirname "$0")/../.."
exec go run ./tools/production-billing-e2e "$@"
