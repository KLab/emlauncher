#!/bin/bash
# Go 版を systemd の下で (再) ビルドして起動する. 先に e2e/env/up.sh で DB などを起動しておくこと.
#   e2e/env/up-systemd.sh [local|s3]
# テストは E2E_TARGET=systemd で実行する.
set -euo pipefail

cd "$(dirname "$0")"
storage="${1:-local}"
export E2E_ENV="e2e_${storage}"

if [ -z "${E2E_GO_CONFIG_DIR:-}" ]; then
	# ディレクトリは作り直さない (他のコンテナがマウントしているため). ファイルだけ上書きする
	mkdir -p ../.work/go-runtime-config
	cp go/config.*.yml ../.work/go-runtime-config/
fi

docker compose -f compose.yml --profile systemd up -d --build --force-recreate --no-deps systemd

echo "waiting for systemd..."
for _ in $(seq 60); do
	curl -sf "http://localhost:${E2E_SYSTEMD_PORT:-18083}/health" >/dev/null && break
	sleep 1
done
curl -sf "http://localhost:${E2E_SYSTEMD_PORT:-18083}/health" >/dev/null
docker compose -f compose.yml --profile systemd exec -T systemd systemctl status emlauncher --no-pager | head -5
echo "ready: systemd storage=${storage} url=http://localhost:${E2E_SYSTEMD_PORT:-18083}"
