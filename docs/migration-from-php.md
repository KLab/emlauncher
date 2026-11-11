# PHP 版から Go 版への移行手順

EC2 + Apache + ALB の構成で, PHP 版を Go 版に切り替える手順. DB とストレージは変更しない.

切替後は Apache を使わない. Go 版が Apache の代わりに 80 番で待ち受け, ALB から直接受ける（ALB とターゲットグループの設定は変えない）. Apache を止めてから Go 版が起動するまでの数秒間はサービスが止まる.

この手順は移行リハーサル（`e2e/migration/rehearse.sh <local|s3> go`）で検証している. リハーサルは PHP 版で中間状態を作り, 下記 1〜2 の手順で Go 版に切り替え, 状態が引き継がれていること, さらに PHP 版へ切り戻せることを確認する（[e2e/README.md](../e2e/README.md)）. ただしリハーサルは Docker で動かすため, 2-3 の Apache と systemd の操作はコンテナの入れ替えで代えている.

## 0. 準備（ビルド）

```sh
cd src && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ../dist/ ./cmd/emlauncher ./cmd/delete-old-packages
```

配置するもの:

- `emlauncher`（HTTP サーバ）, `delete-old-packages`（バッチ）
- 設定ファイル（1. で作る）
- Java（bundletool / keytool 用. PHP 版で使っていたものをそのまま使う）

PHP, composer, memcached の PHP 拡張は Go 版には不要.

## 1. 設定を作る（PHP 版のサーバで行う）

### 1-1. PHP 版の設定から書き出す

このリポジトリの `tools/export_php_config.php` を PHP 版のサーバにコピーして実行する.

```sh
php export_php_config.php /path/to/emlauncher <MFW_ENV> > /etc/emlauncher/config.yml
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

- `listen`: `":80"` にする（Apache の代わりに ALB から受ける. ターゲットグループのポートに合わせる）. 書き出した値は `127.0.0.1:8080`.
- `session.secret`: 書き出すたびにランダムに作られる. **複数台構成では全台で同じ値にする**（違うとサーバをまたぐたびにログアウトされる）.
- `local_file.serve`: `true` のままにする（Apache を使わないので, Go 版が `/files` を配信する）.

## 2. 切替

### 2-1. ファイルの権限を確認する

Go 版は **Apache と同じユーザ**（例: `apache`, `www-data`）で動かす. リハーサルも Apache と同じユーザで動かして確認している.

- `database.authfile` が, そのユーザで読めること.
- LocalFile の場合, 保存先（`local_file.path`）以下に, そのユーザで書き込めること（既存のファイルは Apache が作ったもの）.
- 一時ディレクトリ（`TMPDIR`, 既定は `/tmp`）に, 最大のパッケージ（PHP 版の設定では 7GB）を置ける空きがあること. アップロードされたファイルはいったんここに書かれる.

### 2-2. Go 版のユニットを置く

ここではまだ起動しない（80 番は Apache が使っている）. systemd のユニットの例（`/etc/systemd/system/emlauncher.service`）:

```ini
[Unit]
Description=EMLauncher
After=network.target

[Service]
User=apache
RuntimeDirectory=emlauncher
PIDFile=/run/emlauncher/emlauncher.pid
ExecStart=/opt/emlauncher/emlauncher -config /etc/emlauncher/config.yml -pid-file /run/emlauncher/emlauncher.pid
ExecReload=/bin/sh -c 'kill -HUP $MAINPID'
# apache ユーザのまま 80 番で待ち受けるため
AmbientCapabilities=CAP_NET_BIND_SERVICE
# 停止時は処理中のリクエスト (大きなアップロード) を待つ. emlauncher の -drain-timeout (既定 1h) に合わせる
TimeoutStopSec=1h
Restart=always
# timezone を空にした場合は, ここで TZ を指定してもよい
# Environment=TZ=Asia/Tokyo

[Install]
WantedBy=multi-user.target
```

```sh
systemctl daemon-reload
```

### 2-3. Apache を止めて Go 版を起動する（ここでサービスが止まる）

```sh
# Apache は処理中のリクエストを終えてから終了する. 待ち受けはすぐに閉じるので, Go 版を起動できる
apachectl graceful-stop
systemctl disable httpd
systemctl enable --now emlauncher
curl -s http://127.0.0.1/health        # OK
journalctl -u emlauncher | head        # 起動時に timezone / storage / cache_prefix を出力する
```

- ALB, ターゲットグループ（80 番, ヘルスチェックは `/health`）は変えない.
- ALB が付ける `X-Forwarded-Proto` / `X-Forwarded-Host` から URL を組み立てる.
- PHP 版の `DocumentRoot`（`web/`）, `.htaccess`, Apache の設定は使わなくなる. アクセスログは Apache ではなく Go 版のログ（journald）に出る.
- 以後の更新（バイナリや設定の差し替え）は `systemctl reload emlauncher` で行う（無停止. README 参照）.

### 2-4. バッチを差し替える

cron の `batch/delete_old_packages.php -e <env> -k <keep>` を次に置き換える.

```sh
/opt/emlauncher/delete-old-packages -config /etc/emlauncher/config.yml -k <keep>
```

## 3. 切替後

- 利用者は再ログインする（セッションは引き継がない）.
- 移行前に発行したインストールトークン, ゲストパスの URL はそのまま使える.
- DB のスキーマ変更・データ移行は無い.
- Apache と PHP 版は, 切り戻しに備えて止めたまま残しておき, 不要になったら削除する.

### 切替で変わる挙動

PHP 版の不具合を直したもの:

- API `delete` で最新のパッケージを消すと, PHP 版は**1 つ前のパッケージのファイルを消し, 削除対象のファイルを残していた**（レスポンスも 1 つ前のパッケージになる）. このため, PHP 版で運用していた環境には「DB に行があるのにファイルが無い」パッケージと, DB に行の無いファイルがありうる. Go 版はファイルが無いパッケージがあっても動く.
- API `delete` で最後のパッケージを消すと, PHP 版は 500 を返していた（DB からは消えている）.
- API `delete` で, PHP 版は添付ファイルの実体を消していなかった（ストレージに残る）. Go 版は消す.
- app / package モジュールのエラーページ（Not Found, Permission Denied など）が, PHP 版はステータス 200 で返っていた. Go 版は 403 / 404 などを返す.
- ダウンロード時のファイル名で, PHP 版は空白を `+` にしていた. Go 版は `%20` にする.
- 30 日を超える有効期限のインストールトークンが, PHP 版では発行直後に失効していた.
- 画面のアップロードで, PHP 版は一時ファイル名を検証しておらず, 任意のキーのファイルを移動できた. Go 版は一時ファイルとして作った名前だけを受け付ける.

その他:

- ファイル名・API キー・トークンのランダム文字列は, 形式は同じで, 暗号論的乱数で作るようにした.
- 設定の正規表現（`login.allowed_mailaddr_pattern`）は, PHP 版と同じく preg 形式（`/pattern/flags`）のまま書ける.

### Go 版で確認できていないこと

- 実際の Google / AzureAD でのログイン
- iOS / Android 実機でのインストール

## 4. 切り戻し

Go 版を止めて Apache を起動し, cron も戻す. 切替と同じく数秒間止まる.

```sh
# Go 版は待ち受けをすぐに閉じ, 処理中のリクエストを終えてから終了する.
# 終了を待つと大きなアップロードの間ずっと止まるので, 待たずに Apache を起動する
systemctl disable emlauncher
systemctl stop --no-block emlauncher
systemctl enable httpd
until systemctl start httpd; do sleep 1; done   # 80 番が空くまで再試行する
curl -s http://127.0.0.1/health
```

Go 版で作ったデータ（パッケージ, インストールトークン, パスワード）はそのまま PHP 版で使える（リハーサルで確認している）. 利用者は再ログインが必要.
