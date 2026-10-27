EMLauncher
==========

社内向けのアプリ配布サービスです. iOS (ipa) / Android (apk, aab) のパッケージをアップロードし, 端末からインストールできます.

Go で実装しています. PHP 版はリポジトリから削除しました（最終版はコミット `4124ca6`）. PHP 版からの移行手順は [docs/go-migration/MIGRATION.md](docs/go-migration/MIGRATION.md) を参照してください.

## 必要なもの

- MySQL 5.7 以降 / MariaDB
- memcached
- 保存先: Amazon S3（または S3 互換ストレージ）, もしくはローカルディスク
- Java（Android App Bundle (aab) から apk を作る bundletool と keytool に使う）
- sendmail 互換のコマンド（更新通知やパスワードリセットのメール）
- ビルドに Go 1.23 以降

## Setup

### 1. ビルド

```sh
cd src && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ../dist/ ./cmd/emlauncher ./cmd/delete-old-packages
```

`dist/emlauncher`（HTTP サーバ）と `dist/delete-old-packages`（古いパッケージの削除バッチ）をサーバに配置します. テンプレートや CSS はバイナリに含まれています.

### 2. Database

[sql/database.sql](sql/database.sql) のパスワードを修正してから流し, テーブルを作ります.

```sh
mysql -uroot < sql/database.sql
mysql -uroot emlauncher < sql/tables.sql
```

### 3. bundletool（Android App Bundle を扱う場合）

```sh
curl -sLO https://github.com/google/bundletool/releases/download/1.17.2/bundletool-all-1.17.2.jar
keytool -genkeypair -keystore emlauncher-keystore.jks -keyalg RSA -keysize 2048 -validity 10000 -alias emlauncherkey
```

キーストアのパスワード, ファイル名, エイリアス名は設定ファイルの `apkfile` に書きます.
ARM64 のサーバで動かす場合は, bundletool に含まれる aapt2 が動かないので, ARM64 向けの aapt2 を用意して `apkfile.aapt2` に指定します.
bundletool は内蔵の aapt2 を Java の一時ディレクトリ（既定は `/tmp`）に展開して実行します. `/tmp` が noexec でマウントされている場合は, `apkfile.java` に `-Djava.io.tmpdir=<実行可能なディレクトリ>` を付けてください（例: `java -Xmx2048m -Djava.io.tmpdir=/var/tmp`）.

### 4. 設定

[src/config.sample.yml](src/config.sample.yml) をコピーして環境に合わせて書き換えます（例: `/etc/emlauncher/config.yml`）. 主な項目:

- `timezone`: DB の日時の TZ（空なら実行環境の TZ）
- `database`, `memcache`
- `storage_class`: `S3` / `LocalFile`. S3 のバケットは予め作成しておきます
- `session.secret`: 長いランダムな文字列. 複数台で動かす場合は全台で同じ値にします
- `login`: パスワードログイン, Google / AzureAD ログイン

### 5. 起動

systemd のユニットの例:

```ini
[Unit]
Description=EMLauncher
After=network.target

[Service]
User=emlauncher
RuntimeDirectory=emlauncher
PIDFile=/run/emlauncher/emlauncher.pid
ExecStart=/opt/emlauncher/emlauncher -config /etc/emlauncher/config.yml -pid-file /run/emlauncher/emlauncher.pid
ExecReload=/bin/sh -c 'kill -HUP $MAINPID'
# 停止時は処理中のリクエスト (大きなアップロード) を待つ. emlauncher の -drain-timeout (既定 1h) に合わせる
TimeoutStopSec=1h
Restart=always

[Install]
WantedBy=multi-user.target
```

`listen`（既定 `127.0.0.1:8080`）で待ち受けます. Apache / nginx のリバースプロキシや ALB の裏に置いてください. `X-Forwarded-Proto` / `X-Forwarded-Host` を見て URL を組み立てます. ヘルスチェックは `/health` です.

#### 更新（無停止の再起動）

バイナリや設定ファイルを置き換えて `systemctl reload emlauncher` すると, 処理中のリクエストを止めずに切り替わります. 新しいプロセスが待ち受けソケットを引き継ぎ, 古いプロセスは処理中のリクエスト（長いアップロードを含む）を終えてから終了します. 新しいプロセスが起動できなかった場合（設定の誤りなど）は, 古いプロセスがそのまま動き続けます（ログに `upgrade failed` が出ます）.

`systemctl restart` は一度停止するので, 処理中のアップロードが終わるまで待ち, その間はサービスが止まります. 更新には `reload` を使ってください.

アップロードされたファイルは一時ディレクトリ（`TMPDIR`, 既定は `/tmp`）にいったん書かれるので, 最大のパッケージを置ける空きを用意してください.

### 6. ユーザの追加

パスワードでログインするユーザは, `user_pass` テーブルにメールアドレスだけを登録し, ログイン画面の「forgot password」からパスワードを設定してもらいます.

```sql
INSERT INTO user_pass (mail) VALUES ('your-name@example.com');
```

Google / AzureAD ログインでは, `login.allowed_mailaddr_pattern` に一致するか `user_pass` に登録されたアカウントがログインできます.

### 7. 古いパッケージの削除（任意）

保護されていないパッケージを, アプリごとに新しいものから指定数だけ残して削除します. cron などで実行します.

```sh
/opt/emlauncher/delete-old-packages -config /etc/emlauncher/config.yml -k 100
```

## 開発

- テスト: `cd src && go test ./...`. DB や S3 を使うテストは, E2E テストの環境を起動して `EML_TEST_DB_HOST` などを指定したときだけ実行されます.
- E2E テスト・移行リハーサル: [docs/go-migration/TESTING.md](docs/go-migration/TESTING.md)
- 設計と移行の経緯: [docs/go-migration/PLAN.md](docs/go-migration/PLAN.md)
