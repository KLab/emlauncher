# Go 移植のテスト

PHP 版に対して作ったシナリオテストを、そのまま Go 版にも流す。テストは HTTP で EMLauncher を操作し、DB・ストレージ（LocalFile / S3）・memcached・送信メールを直接確認する。URL、リクエストパラメータ名、DB の内容、ストレージのキーを「仕様」として固定するためのもの。

```
e2e/
  env/                テスト環境 (docker compose: MySQL / memcached / LocalStack / PHP 版)
  *.go                テスト用のクライアント, DB・ストレージ・メールの確認, 高レベル操作,
                      IPA / APK / AAB / PNG の生成 (package e2e)
  scenario/           シナリオテスト
  migration/          移行リハーサル (seed → 移行 → verify)
  cmd/genfixture      フィクスチャを書き出す (手動確認用)
  .work/              作業領域 (git 管理外): LocalFile の保存先, メールボックス, 移行状態
```

`e2e/` は独立した Go モジュール（`emlauncher-e2e`）。テスト専用の依存（AWS SDK など）を本体に持ち込まないため。

### PHP 版のソース

PHP 版はリポジトリから削除したので, `e2e/env/up.sh` が `e2e/env/php-src.sh` で過去のコミットからソースを取り出して `e2e/.work/php-src/` に置く（`git archive`, mfw は当時の submodule のコミットを clone, composer の依存はコンテナで install）。既定は PHP 版の最終版（`4124ca6`）。別のコミットを使うときは `E2E_PHP_REF=<commit>` を指定する。

## 必要なもの

- Docker（WSL の Ubuntu 上の Docker で動作確認済み。Docker Desktop は使わない）
- Go 1.23 以上
- git と, インターネットへの接続（PHP 版の mfw と composer の依存の取得. 初回のみ）

WSL の Go が古い場合は、公式の方法でホーム以下に追加できる。

```sh
go install golang.org/dl/go1.23.4@latest && ~/go/bin/go1.23.4 download
export PATH=$HOME/sdk/go1.23.4/bin:$PATH
```

## シナリオテスト

```sh
# PHP 版 (LocalFile) を起動. DB・ストレージ・メールボックスは毎回空になる
e2e/env/up.sh local
cd e2e && go test ./scenario/ -count=1

# S3 (LocalStack) で
e2e/env/up.sh s3
cd e2e && E2E_STORAGE=s3 go test ./scenario/ -count=1
```

### Go 版に対して

Go 版は PHP 版と同じ DB・memcached・ストレージ・メールボックスに向けて起動する（`e2e/env/go/` の設定を使う）。`E2E_TARGET=go` にすると, 接続先（`http://localhost:18082`）とバッチのコマンドが Go 版に切り替わる。

```sh
e2e/env/up.sh local && e2e/env/up-go.sh local
cd e2e && E2E_TARGET=go go test ./scenario/ -count=1

e2e/env/up.sh s3 && e2e/env/up-go.sh s3
cd e2e && E2E_TARGET=go E2E_STORAGE=s3 go test ./scenario/ -count=1
```

2026-10-23 時点で, PHP 版・Go 版とも LocalFile・S3 の両方で全 36 シナリオが通る。

Go 版だけのシナリオとして, 無停止の再起動（`TestGracefulRestart`, `TestGracefulRestartFailure`）がある（PHP 版ではスキップ）。Go 版のコンテナで `kill -HUP` を送り, 再起動中に送り続けたリクエストが失敗しないこと, 途中まで送っていたアップロードが古いプロセスで完了すること, 古いプロセスが終了すること, 設定を壊したときは古いプロセスが動き続けることを確認する。Go 版の設定は `e2e/.work/go-runtime-config/` のコピーを使い, テストが一時的に書き換える。

### 主な環境変数

| 変数 | 既定値 | 内容 |
|---|---|---|
| `E2E_TARGET` | `php` | `php` / `go`。PHP 版の既知バグの扱いを切り替える |
| `E2E_BASE_URL` | `http://localhost:18081` | テスト対象の URL |
| `E2E_DB_DSN` | `emlauncher:emlauncher@tcp(localhost:13306)/emlauncher?charset=utf8mb4` | DB |
| `E2E_STORAGE` | `local` | `local` / `s3` |
| `E2E_LOCAL_DIR` | `e2e/.work/files` | LocalFile の保存先（ホスト側のパス） |
| `E2E_S3_ENDPOINT` / `E2E_S3_BUCKET` | `http://localhost:14566` / `emlauncher-e2e` | S3 |
| `E2E_MAILBOX` | `e2e/.work/mailbox` | 送信メールの置き場所（1 通 1 ファイル、sendmail -t 形式） |
| `E2E_MEMCACHED` / `E2E_CACHE_PREFIX` | `localhost:21211` / `emlauncher_e2e_{storage}_emlauncher_` | memcached とキーの接頭辞 |
| `E2E_TZ` | `Asia/Tokyo` | テスト対象の TZ。DB の日時がこの TZ で「今」になっているかを確認する |
| `E2E_BATCH_CMD` | PHP 版のバッチを docker で実行 | 古いパッケージ削除バッチ。`{keep}` が保持数に置き換わる |

`E2E_TARGET=go` のとき, `E2E_BASE_URL` と `E2E_BATCH_CMD` の既定値は Go 版のもの（`e2e/env/up-go.sh` で起動したもの）になる。メールは sendmail 互換コマンドで `E2E_MAILBOX` にファイルとして書き出す。

### 仕様として固定したもの

- URL とリクエストパラメータ名（API、画面の form、ajax）
- API の JSON のキー。数値フィールド（id、file_size など）は文字列・数値のどちらでもよい（API ドキュメントの Caveats のとおり。PHP 版はエンドポイントによって混在している）
- DB の行の内容（日時は TZ のローカル時刻、ランダム文字列の形式、タグの再利用、コメント番号、date_to_sort、last_upload の巻き戻しなど）
- ストレージのキー、S3 の ContentType と ACL
- memcached のインストールトークンの形式（`pkg_install_token_{token}` に JSON `{"mail","package_id","expire"}`）
- メール（宛先、Bcc、件名、本文の主な内容）
- エラーページの本文のメッセージ（"Not Found"、"Permission Denied" など）

### PHP 版の既知バグ

`E2E_TARGET=php` のときは現状の（誤った）挙動を確認し、`go` のときは正しい挙動を要求する（`Env.PHPBug`）。

| バグ | 影響 |
|---|---|
| API `delete` で最新のパッケージを消すと、**1 つ前のパッケージのファイルを消し、削除対象のファイルが残る**。レスポンスも 1 つ前のパッケージになる | データ消失。本番にも「DB に行があるのにファイルが無い」パッケージと、行の無いファイルがありうる |
| API `delete` で最後のパッケージを消すと 500（DB からは消えている） | クライアントがエラー扱いする |
| app / package モジュールのエラーページ（Not Found、Permission Denied など）がステータス 200 で返る | 表示上は問題ない |

### PHP 版の実際の挙動に合わせたもの

- **S3 の ContentType**: パッケージ・添付ファイル・一時ファイルはすべて `application/octet-stream`（アイコンだけ `image/png`）。PHP 版はコード上は MIME を指定しているが、aws-sdk-php の MultipartUploader がその設定を読まず、アップロード元の一時ファイルに拡張子も無いため。S3 の構成を変えないため、これを仕様とした。
- `description` を省略してアップロードすると NULL になる。

## 画面の比較

PHP 版と Go 版を同じ DB に向けて起動した状態で, 同じ画面を PC 幅・スマホ幅で撮影して並べる（Playwright のコンテナを使う）。

```sh
cd e2e
docker run --rm --network host -v $PWD:/e2e -w /e2e mcr.microsoft.com/playwright/python:v1.48.0-jammy   sh -c "pip install -q playwright==1.48.0; python3 screenshots.py --mail <mail> --password <password> --app <app id> --package <package id>"
# e2e/.work/screenshots/index.html (左が PHP 版, 右が Go 版)
```

2026-10-23 時点で, 15 画面 × 2 幅のすべてでページの大きさが一致し, 画素の差は次のものだけ（画素の差分で確認）。

- テスト環境のタイトルの接頭辞（`[E2E]` と `[E2E Go]`）
- QR コードの模様（生成ライブラリが違う. 読み取れる URL は同じ）
- ドキュメント内の URL（ポート番号）

## 移行リハーサル

PHP 版で運用中にありそうな「中間状態」を作り、移行手順どおりに切り替えた後、状態がそのまま使えることを確認する。

```sh
cd e2e
migration/rehearse.sh local go    # LocalFile, PHP 版 → Go 版 → PHP 版へ切り戻し
migration/rehearse.sh s3 go       # S3
migration/rehearse.sh local php   # 移行先を PHP 版にする (リハーサルの仕組み自体の確認用)
```

2026-10-23 時点で, LocalFile・S3 の両方で全フェーズが通る。

1. PHP 版を空の状態で起動する
2. **seed**（`E2E_MIGRATION_PHASE=seed`）: 画面と API だけで中間状態を作り、`e2e/.work/migration-state.json` に記録する
   - パスワードリセットで登録したユーザと、passhash を直接持つユーザ
   - 共同オーナーのアプリ、API キー更新済みのアプリ
   - IPA（保護）、APK（画面で編集・添付）、AAB（生成 APK・dSYM 付き）、画面からのアップロード、不明形式、削除済み（id の欠番）
   - インストール履歴（Android / iOS / PC）、通知を切ったユーザ、コメント
   - 有効なゲストパスと失効したゲストパス、API と画面で発行したインストールトークン（memcached）
   - upload_post されずに残った一時ファイル
3. **移行**: 移行先が `go` なら [MIGRATION.md](MIGRATION.md) の手順どおりに行う
   - PHP 版のコンテナで `tools/export_php_config.php` を実行して Go 版の設定を書き出す（TZ, memcached の接頭辞, DB, ストレージなど）
   - `listen` だけ環境に合わせて書き換える
   - PHP 版を止め, Go 版を **Apache と同じユーザ（www-data）** で起動する。authfile は同じホストにある前提でマウントする
   - 移行先が `php` ならコンテナを作り直すだけ
4. **verify**（`E2E_MIGRATION_PHASE=verify`）
   - 移行手順の前後で DB とストレージが変わっていない
   - 既存のパスワードで再ログインできる。画面・API の応答が移行前と同じ
   - 移行前にアップロードされたファイルをダウンロードでき、中身が同じ（Android では AAB の生成 APK）
   - iOS のインストール（plist）、移行前に発行したインストールトークンとゲストパス
   - 通知設定が引き継がれる、既存タグが再利用される、コメント番号が続く
   - 移行前のパッケージを編集・削除できる（ファイルも消える）、パスワードリセット、古いパッケージ削除バッチ

5. **切り戻し**（移行先が `go` のとき）: Go 版を止めて PHP 版を起動する
6. **rollback**（`E2E_MIGRATION_PHASE=rollback`）: verify で Go 版が作ったデータが PHP 版で使える
   - Go 版で変更したパスワード（Go 版が作ったハッシュ）でログインできる
   - Go 版がアップロードしたパッケージを表示・ダウンロードできる
   - Go 版が発行したインストールトークン（memcached）が使える

verify は状態を変更するため、一度流したら seed からやり直す。

### リハーサルで見つかったこと

- 書き出した設定の `database.authfile` は PHP 版のサーバ上のパス。Go 版を同じサーバで, そのファイルを読めるユーザで動かす必要がある（手順書に記載）。
- Go 版を Apache と同じユーザで動かすと, LocalFile の保存で chmod が失敗した（所有者の違うディレクトリ）。chmod を使わないよう修正した。
