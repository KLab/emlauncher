#!/bin/bash
# 移行リハーサル.
# 移行元 (PHP 版) で中間状態を作り, 移行手順どおりに移行先へ切り替えて, 状態が引き継がれていることを確認する.
#
#   e2e/migration/rehearse.sh [local|s3] [php|go]
#
# 移行先:
#   php : 移行手順の代わりに PHP 版のコンテナを作り直す (リハーサルの仕組み自体の確認用)
#   go  : docs/go-migration/MIGRATION.md の手順で Go 版に切り替え, 最後に PHP 版への切り戻しも確認する
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
		# docs/go-migration/MIGRATION.md の手順どおりに行う
		echo "--- 3-1. PHP 版の設定から Go 版の設定を作る"
		mkdir -p .work/go-config
		compose cp ../tools/export_php_config.php php:/tmp/export_php_config.php
		compose exec -T php php /tmp/export_php_config.php /srv/emlauncher "$E2E_ENV" > ".work/go-config/config.${E2E_ENV}.yml"
		# listen はリバースプロキシの構成に合わせる (コンテナなので全インタフェースで待ち受ける)
		sed -i 's/"127.0.0.1:8080"/"0.0.0.0:8080"/' ".work/go-config/config.${E2E_ENV}.yml"
		cat ".work/go-config/config.${E2E_ENV}.yml"
		echo "--- 3-2. PHP 版を止める"
		compose stop php
		echo "--- 3-3. Go 版を Apache と同じユーザで起動する"
		E2E_GO_CONFIG_DIR=../.work/go-config E2E_GO_USER=33:33 env/up-go.sh "$storage"
		;;
	*)
		echo "usage: $0 [local|s3] [php|go]" >&2
		exit 2
		;;
esac

echo "=== 4. 移行先で確認"
E2E_TARGET="$target" E2E_MIGRATION_PHASE=verify go test ./migration -run '^TestVerify$' -count=1 -v

if [ "$target" = go ]; then
	echo "=== 5. 切り戻し (Go 版を止めて PHP 版を起動する)"
	compose --profile go stop go
	compose start php
	for _ in $(seq 60); do
		curl -sf "http://localhost:${E2E_PHP_PORT:-18081}/health" >/dev/null && break
		sleep 1
	done
	echo "=== 6. 切り戻した移行元で, 移行先で作ったデータを確認"
	E2E_TARGET=php E2E_MIGRATION_PHASE=rollback go test ./migration -run '^TestRollback$' -count=1 -v
fi
