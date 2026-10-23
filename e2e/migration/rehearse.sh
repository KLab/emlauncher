#!/bin/bash
# 移行リハーサル.
# 移行元 (PHP 版) で中間状態を作り, 移行手順どおりに移行先へ切り替えて, 状態が引き継がれていることを確認する.
#
#   e2e/migration/rehearse.sh [local|s3] [php|go]
#
# 移行先:
#   php : 移行手順の代わりに PHP 版のコンテナを作り直す (リハーサルの仕組み自体の確認用)
#   go  : docs/go-migration/MIGRATION.md の手順で Go 版に切り替える (Go 版の実装後に有効)
set -euo pipefail

storage="${1:-local}"
target="${2:-php}"
cd "$(dirname "$0")/.."

export E2E_STORAGE="$storage"
export E2E_ENV="e2e_${storage}"
compose() { docker compose -f env/compose.yml "$@"; }

echo "=== 1. 移行元 (PHP 版) を空の状態で起動"
env/up.sh "$storage"

echo "=== 2. 移行元で中間状態を作る"
E2E_MIGRATION_PHASE=seed go test ./migration -run '^TestSeed$' -count=1 -v

echo "=== 3. 移行 (${target})"
case "$target" in
	php)
		compose up -d --force-recreate --no-deps php
		for _ in $(seq 60); do
			curl -sf "http://localhost:${E2E_PHP_PORT:-18081}/health" >/dev/null && break
			sleep 1
		done
		;;
	go)
		echo "Go 版への移行手順は未実装 (docs/go-migration/MIGRATION.md 参照)" >&2
		exit 1
		;;
	*)
		echo "usage: $0 [local|s3] [php|go]" >&2
		exit 2
		;;
esac

echo "=== 4. 移行先で確認"
E2E_TARGET="$target" E2E_MIGRATION_PHASE=verify go test ./migration -run '^TestVerify$' -count=1 -v
