#!/bin/bash
# E2E テスト用に PHP 版 (KLab/emlauncher) のソースを用意する.
#   e2e/env/php-src.sh [ref]
# ref の既定値は E2E_PHP_REF, それも無ければ PHP 版の基準の版 (KLab/emlauncher の 26ea3e5).
# このリポジトリの履歴に ref が無ければ, E2E_PHP_REPO (既定は KLab/emlauncher) から取得する.
# e2e/.work/php-src/ に, mfw (submodule) と composer の依存も含めて展開する. 同じ ref なら何もしない.
set -euo pipefail

cd "$(dirname "$0")"
ref="${1:-${E2E_PHP_REF:-26ea3e55b581272452e0ab14accfb4748d100a07}}"
dst="$(cd .. && pwd)/.work/php-src"
g() { git -c safe.directory="*" "$@"; }

repo="$(g rev-parse --show-toplevel)"
if ! commit="$(g -C "$repo" rev-parse --verify --quiet "${ref}^{commit}")"; then
	repo="$(cd .. && pwd)/.work/php-repo"
	if [ ! -d "$repo" ]; then
		g clone -q --bare --filter=blob:none "${E2E_PHP_REPO:-https://github.com/KLab/emlauncher.git}" "$repo"
	fi
	commit="$(g -C "$repo" rev-parse --verify "${ref}^{commit}")"
fi

if [ -f "$dst/.ref" ] && [ "$(cat "$dst/.ref")" = "$commit" ]; then
	echo "php-src: ${commit} (cached)"
	exit 0
fi

echo "php-src: extracting ${commit}"
rm -rf "$dst"
mkdir -p "$dst"
g -C "$repo" archive "$commit" | tar -x -C "$dst"

# mfw (submodule) はそのコミットが指している版を取得する
mfw_url="$(g -C "$repo" show "${commit}:.gitmodules" | sed -n 's/^[[:space:]]*url = //p' | head -1)"
mfw_commit="$(g -C "$repo" ls-tree "$commit" mfw | awk '{print $3}')"
rm -rf "$dst/mfw"
git clone -q "$mfw_url" "$dst/mfw"
git -C "$dst/mfw" checkout -q "$mfw_commit"

# composer の依存 (PHP の拡張はイメージ側にあるので, ここでは確認しない)
docker run --rm -u "$(id -u):$(id -g)" -v "$dst:/app" -w /app composer:2.8.3 \
	install --ignore-platform-reqs --no-interaction --no-progress --quiet

echo "$commit" > "$dst/.ref"
echo "php-src: ready (${commit})"
