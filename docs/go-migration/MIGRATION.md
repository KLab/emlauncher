# PHP 版から Go 版への移行手順

EC2 + Apache + ALB の構成で, PHP 版を Go 版に切り替える手順. DB とストレージは変更しない.

この手順は移行リハーサル（`e2e/migration/rehearse.sh <local|s3> go`）で検証している. リハーサルは PHP 版で中間状態を作り, 下記 1〜2 の手順で Go 版に切り替え, 状態が引き継がれていること, さらに PHP 版へ切り戻せることを確認する（[TESTING.md](TESTING.md)）.

## 0. 準備（ビルド）

```sh
cd go
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o dist/ ./cmd/emlauncher ./cmd/delete-old-packages
```

配置するもの:

- `emlauncher`（HTTP サーバ）, `delete-old-packages`（バッチ）
- 設定ファイル（1. で作る）
- Java（bundletool / keytool 用. PHP 版で使っていたものをそのまま使う）

PHP, composer, memcached の PHP 拡張は Go 版には不要.

## 1. 設定を作る（PHP 版のサーバで行う）

### 1-1. PHP 版の設定から書き出す

```sh
php go/tools/export_php_config.php /path/to/emlauncher <MFW_ENV> > /etc/emlauncher/config.yml
```

`/path/to/emlauncher` は PHP 版を置いたディレクトリ（`initialize.php` があるところ）. 次の値を PHP 版と同じにして書き出す.

| 項目 | 書き出し元 | 注意 |
|---|---|---|
| `timezone` | PHP の実行時の TZ（`date_default_timezone_get()`） | CLI と Apache で php.ini が違う場合がある. Apache 側の `date.timezone`（php.ini, Apache の設定, `.htaccess` の `php_value`）と一致しているか確認する. 違うと日時が数時間ずれて記録される |
| `cache_prefix` | `{application_identifier}_{MFW_ENV}_{ディレクトリ名}_` | 違うと移行前に発行したインストールトークンが使えなくなる |
| `database` | `mfw_serverenv_config.php` の DSN と authfile | unix_socket の DSN は未対応（警告が出るので host / port を手で設定する） |
| `mail.sendmail` | php.ini の `sendmail_path` | |
| その他 | `emlauncher_config.php` | キー名は同じ |

### 1-2. 環境に合わせて直す

- `listen`: 既定は `127.0.0.1:8080`（Apache のリバースプロキシから受ける）.
- `session.secret`: 書き出すたびにランダムに作られる. **複数台構成では全台で同じ値にする**（違うとサーバをまたぐたびにログアウトされる）.
- `local_file.serve`: 既定は `true`（Go 版が `/files` を配信する）. Apache の Alias で配信し続けるなら `false`.

## 2. 切替

### 2-1. ファイルの権限を確認する

Go 版は **Apache と同じユーザ**（例: `apache`, `www-data`）で動かす. リハーサルも Apache と同じユーザで動かして確認している.

- `database.authfile` が, そのユーザで読めること.
- LocalFile の場合, 保存先（`local_file.path`）以下に, そのユーザで書き込めること（既存のファイルは Apache が作ったもの）.
- 一時ディレクトリ（`TMPDIR`, 既定は `/tmp`）に, 最大のパッケージ（PHP 版の設定では 7GB）を置ける空きがあること. アップロードされたファイルはいったんここに書かれる.

### 2-2. Go 版を起動する

systemd のユニットの例（`/etc/systemd/system/emlauncher.service`）:

```ini
[Unit]
Description=EMLauncher
After=network.target

[Service]
User=apache
ExecStart=/opt/emlauncher/emlauncher -config /etc/emlauncher/config.yml
Restart=always
# timezone を空にした場合は, ここで TZ を指定してもよい
# Environment=TZ=Asia/Tokyo

[Install]
WantedBy=multi-user.target
```

```sh
systemctl daemon-reload
systemctl enable --now emlauncher
curl -s http://127.0.0.1:8080/health   # OK
journalctl -u emlauncher | head        # 起動時に timezone / storage / cache_prefix を出力する
```

### 2-3. Apache のリバースプロキシを Go 版に向ける

```apache
# mod_proxy, mod_proxy_http を有効にする
ProxyPreserveHost On
ProxyTimeout 600
ProxyPass        / http://127.0.0.1:8080/
ProxyPassReverse / http://127.0.0.1:8080/
```

- ALB が付ける `X-Forwarded-Proto` はそのまま Go 版に渡る. Go 版は `X-Forwarded-Proto` / `X-Forwarded-Host` から URL を組み立てる.
- PHP 版の `DocumentRoot`（`web/`）と `.htaccess` の設定は使わなくなる.
- 段階的に移行する場合は, 先に `ProxyPass /api/ http://127.0.0.1:8080/api/` だけ設定する（API はセッションを使わないので PHP 版と並行して動かせる）.
- ALB のヘルスチェックは `/health` のまま.

### 2-4. バッチを差し替える

cron の `batch/delete_old_packages.php -e <env> -k <keep>` を次に置き換える.

```sh
/opt/emlauncher/delete-old-packages -config /etc/emlauncher/config.yml -k <keep>
```

## 3. 切替後

- 利用者は再ログインする（セッションは引き継がない）.
- 移行前に発行したインストールトークン, ゲストパスの URL はそのまま使える.
- DB のスキーマ変更・データ移行は無い.

## 4. 切り戻し

Apache のリバースプロキシの設定を外して PHP 版に戻し, cron も戻す. Go 版で作ったデータ（パッケージ, インストールトークン, パスワード）はそのまま PHP 版で使える（リハーサルで確認している）. 利用者は再ログインが必要.
