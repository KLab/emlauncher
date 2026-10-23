#!/bin/bash
# E2E テスト環境 (PHP 版) を作り直して起動する.
#   e2e/env/up.sh [local|s3]
# DB / ストレージ / メールボックスは毎回空の状態から始まる.
set -euo pipefail

cd "$(dirname "$0")"
storage="${1:-local}"
case "$storage" in
	local|s3) ;;
	*) echo "usage: $0 [local|s3]" >&2; exit 2 ;;
esac
export E2E_ENV="e2e_${storage}"

compose() { docker compose -f compose.yml "$@"; }

compose --profile go down -v --remove-orphans >/dev/null 2>&1 || true
rm -rf ../.work/files ../.work/mailbox
mkdir -p ../.work/files ../.work/mailbox
chmod 777 ../.work/files ../.work/mailbox

compose up -d --build

echo "waiting for s3..."
for _ in $(seq 60); do
	compose exec -T s3 awslocal s3 mb s3://emlauncher-e2e >/dev/null 2>&1 && break
	sleep 1
done
compose exec -T s3 awslocal s3 ls s3://emlauncher-e2e >/dev/null

echo "waiting for php..."
for _ in $(seq 60); do
	curl -sf "http://localhost:${E2E_PHP_PORT:-18081}/health" >/dev/null && break
	sleep 1
done
curl -sf "http://localhost:${E2E_PHP_PORT:-18081}/health" >/dev/null
echo "ready: storage=${storage} url=http://localhost:${E2E_PHP_PORT:-18081}"
