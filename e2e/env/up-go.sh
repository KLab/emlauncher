#!/bin/bash
# Go 版を (再) ビルドして起動する. 先に e2e/env/up.sh で DB などを起動しておくこと.
#   e2e/env/up-go.sh [local|s3]
set -euo pipefail

cd "$(dirname "$0")"
storage="${1:-local}"
export E2E_ENV="e2e_${storage}"

docker compose -f compose.yml --profile go up -d --build --force-recreate --no-deps go

echo "waiting for go..."
for _ in $(seq 60); do
	curl -sf "http://localhost:${E2E_GO_PORT:-18082}/health" >/dev/null && break
	sleep 1
done
curl -sf "http://localhost:${E2E_GO_PORT:-18082}/health" >/dev/null
echo "ready: go storage=${storage} url=http://localhost:${E2E_GO_PORT:-18082}"
