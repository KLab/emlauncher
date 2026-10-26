#!/bin/bash
# PHP 版のソースを過去のコミットから取り出す (PHP 版はリポジトリから削除済み).
#   e2e/env/php-src.sh [ref]
# ref の既定値は E2E_PHP_REF, それも無ければ PHP 版の最終版 (master の 4124ca6).
# e2e/.work/php-src/ に, mfw (submodule) と composer の依存も含めて展開する. 同じ ref なら何もしない.
set -euo pipefail

cd "$(dirname "$0")"
ref="${1:-${E2E_PHP_REF:-4124ca6d5245572f13e96fce75b15bc17d30bb0b}}"
repo="$(git -c safe.directory="*" rev-parse --show-toplevel)"
dst="$(cd .. && pwd)/.work/php-src"
commit="$(git -c safe.directory="*" -C "$repo" rev-parse --verify "${ref}^{commit}")"

if [ -f "$dst/.ref" ] && [ "$(cat "$dst/.ref")" = "$commit" ]; then
	echo "php-src: ${commit} (cached)"
	exit 0
fi

echo "php-src: extracting ${commit}"
rm -rf "$dst"
mkdir -p "$dst"
git -c safe.directory="*" -C "$repo" archive "$commit" | tar -x -C "$dst"

# mfw (submodule) はそのコミットが指している版を取得する
mfw_url="$(git -c safe.directory="*" -C "$repo" show "${commit}:.gitmodules" | sed -n 's/^[[:space:]]*url = //p' | head -1)"
mfw_commit="$(git -c safe.directory="*" -C "$repo" ls-tree "$commit" mfw | awk '{print $3}')"
rm -rf "$dst/mfw"
git clone -q "$mfw_url" "$dst/mfw"
git -C "$dst/mfw" checkout -q "$mfw_commit"

# composer の依存 (PHP の拡張はイメージ側にあるので, ここでは確認しない)
docker run --rm -u "$(id -u):$(id -g)" -v "$dst:/app" -w /app composer:2.8.3 \
	install --ignore-platform-reqs --no-interaction --no-progress --quiet

echo "$commit" > "$dst/.ref"
echo "php-src: ready (${commit})"
