# EMLauncher Go 移植計画

PHP 版 EMLauncher を Go に移植する計画。挙動の完全一致は求めないが、**DB スキーマと S3（ストレージ）の構成は一切変えない**ことを絶対条件とする。

## 0. 決定事項

| 項目 | 決定 |
|---|---|
| DB / S3 構成 | 変更しない。Go 側はマイグレーションを持たず、既存スキーマに合わせる |
| タイムゾーン | 設定可能にする。未指定なら実行環境の TZ を使う（本番は JST が多い） |
| 本番構成 | EC2 + Apache、ロードバランサ（ALB）配下、非コンテナ。Go は systemd サービスとして Apache の裏で動かす |
| セッション | 互換を捨てる（切替時に全員が再ログイン）。インストールトークンは互換を保つ |
| 画面 | 作り直してよい。ただしボタンや機能の位置は大きく変えない |
| LocalFile ストレージ | Go 版にも残す。PHP 版で保存済みのファイルをそのまま扱えること |
| テスト | PHP 版に対してシナリオテストを先に作り、同じテストを Go 版に通す（[TESTING.md](TESTING.md)） |
| 移行手順の検証 | PHP 版で中間状態を作り、手順書どおりに Go 版へ移行できることをテストで確認する |

## 1. 現行構成

### 1.1 アプリケーション

- Apache + PHP 8.2。`.htaccess` で全リクエストを `web/main.php` に集め、自作フレームワーク mfw（submodule）が `/{module}/{action}` で振り分ける。
- actions 約 2,300 行、models 約 1,500 行、テンプレート約 3,200 行（PHP 直書き、Bootstrap 3 + jQuery）。
- 外部依存: MySQL、S3（または LocalFile）、memcached、sendmail、Google / AzureAD OAuth、Java（bundletool / keytool）。

### 1.2 DB（12 テーブル、utf8mb4）

`application`, `application_owner`, `tag`, `package`, `package_tag`, `attached_file`, `comment`, `install_log`, `app_install_user`, `guest_pass`, `guestpass_log`, `user_pass`（定義: `data/sql/tables.sql`）

移植で守るべき暗黙の仕様:

- **日時**: `date('Y-m-d H:i:s')`、つまり PHP の TZ のローカル時刻で書き込んでいる。Go も同じ TZ のローカル時刻で読み書きする。
- **パスワード**: `user_pass.passhash` は `"{stretch}:{salt}:{hash}"`。`hash = sha1(pass.salt)` の後、`hash = sha1(hash.salt)` を stretch 回繰り返す。stretch は 10〜20、salt は英数字 16 文字。
- **date_to_sort**: `application.date_to_sort` = max(created, last_upload, last_commented)。
- **last_upload の巻き戻し**: パッケージ削除時、`application.last_upload == package.created` なら、残っている最新パッケージの created（無ければ NULL）に戻す。
- **コメント番号**: `comment.number` はアプリごとの「最大値 + 1」。
- **notify**: `app_install_user.notify` はスキーマのコメントだと「0:送る」だが、コードは**真なら送る**。コードに従う。
- **ファイル名・API キー・トークン**: `[0-9a-zA-Z]` の 16 / 32 文字。Go では crypto/rand を使う。
- **package_tag**: パッケージとタグの対応。タグ削除時は package_tag から先に消す。

### 1.3 ストレージのキー構成（S3 / LocalFile 共通）

| 用途 | キー | S3 ACL / ContentType |
|---|---|---|
| アイコン | `app-icons/{app_id}/{rand16}.png`（144×144 PNG） | public-read / image/png。URL は署名なし |
| パッケージ | `package/{app_id}/{pkg_id}_{package.file_name}` | private / `application/octet-stream`（※） |
| 添付ファイル | `package/{app_id}/{pkg_id}_{attached_id}_{attached_file.file_name}` | private / `application/octet-stream`（※） |
| 一時ファイル | `temp-data/{rand16}.{ext}` | private / `application/octet-stream`（※）。確定時に copy + delete で正式キーへ移動 |

※ コード上は apk に `application/vnd.android.package-archive` などを指定しているが、aws-sdk-php の MultipartUploader がその設定を読まず、アップロード元の一時ファイルに拡張子も無いため、実際にはすべて `application/octet-stream` で保存されている（シナリオテストで確認）。Go 版もこれに合わせる。正しい MIME にするかは別途判断する。

- ダウンロードは 60 分有効の署名付き URL。`ResponseContentDisposition: attachment; filename*=UTF-8''{urlencode(original_file_name)}`
- `aws.base_url` 指定時は path-style。署名付き URL の `base_url` 部分を `external_url` に置換して返す。
- **LocalFile**: 保存先は `rtrim(local_file.path,'/') + '/' + trim(key,'/')`、URL は `rtrim(url_prefix,'/') + '/' + key`。有効期限なし。配信は Apache の Alias で行っている。

### 1.4 memcached

キーの先頭には `{application_identifier}_{env}_{branch}_` が付く（mfwServerEnv::cachePrefix）。`branch` は **PHP を置いたディレクトリ名**（`basename(APP_ROOT)`）。Go 版では設定 `cache_prefix` で明示し、移行時に PHP 版と同じ値にする。

| キー | 値 | 有効期限 | Go 版での扱い |
|---|---|---|---|
| `pkg_install_token_{token}` | JSON 文字列 `{"mail","package_id","expire"}` | 発行時に指定（API は任意時間） | **互換を保つ**（同じキー・同じ形式） |
| `ios_plist_token_{token}` | pkg_id | 60 秒 | 形式を合わせる（短命なので互換は必須ではない） |
| `{sha1}`（パスワードリセット） | PHP シリアライズされた配列 | 30 分 | 互換なし（Go 版は JSON で保存） |

### 1.5 セッション

PHP のファイルセッション（Cookie 名 `{prefix}seskey`）。内容はログイン中のメールアドレス、ログイン前の URL、削除確認用の CSRF トークン、OAuth 失敗時のメールアドレス。

### 1.6 エンドポイント

- **画面**: `top`, `app`（index / new / create / comment / comment_post / preferences* / upload / upload_post）, `package`（index / install / install_plist / create_token / create_guestpass / guestpass / expire_guestpass / guestpass_install / edit / edit_commit / delete_confirm / delete / attach / attach_download / attach_delete_confirm / attach_delete）, `myapps`（installed / own / delete）, `guestpass/list`, `login`（index / password* / google* / azuread*）, `logout`, `doc/*`, `qr/code`, `health`
- **ajax**（セッション認証）: `upload_package_temporary`, `notification_setting`
- **API**（`api_key` 認証、CI から利用）: `upload`, `package_list`, `delete`, `create_token`, `package_edit`, `package_attach`, `package_detach`
- **バッチ**: `batch/delete_old_packages.php`（`-e env -k keep -n`）

## 2. 移植方針

1. **DB スキーマとストレージのキー・ACL・ContentType は変えない。** 既存スキーマに合わせた手書き SQL で実装する。
2. **URL とリクエストパラメータ名は変えない。** API は外部（CI）互換のため。画面の form パラメータ名も、シナリオテストを PHP 版と共用するため変えない。
3. **API の JSON はキーを合わせる。** 数値フィールド（id、file_size など）の型は、API ドキュメントの Caveats で「文字列か数値か不定」とされており、PHP 版も実際にエンドポイントによって混在している（アップロード直後は数値、DB から読んだものは文字列）。Go 版は文字列に統一する。`protect` は bool。
4. **インストールトークン（memcached）は同じキーと形式で読み書きする。** 切替前に発行済みの URL を切替後も使えるようにするため。
5. **セッションは署名付き Cookie にする。** ALB 配下で複数台になってもスティッキーセッションが不要になる。
6. **OAuth のコールバック URL（`/login/google_callback`, `/login/azuread_callback`）は変えない。**

### 2.1 ライブラリの置き換え

| PHP | Go |
|---|---|
| PDO | `go-sql-driver/mysql` + `sqlx` |
| aws-sdk-php | `aws-sdk-go-v2`（s3 / `feature/s3/manager`） |
| Memcached | `bradfitz/gomemcache` |
| CFPropertyList | `howett.net/plist` |
| php-apk-parser | `shogo82148/androidbinary/apk` |
| bundletool / keytool | 引き続き `exec` で呼ぶ |
| Imagick | `image` + `golang.org/x/image/draw`（png / jpeg / gif / webp） |
| endroid/qr-code | `skip2/go-qrcode` |
| mb_send_mail | `sendmail -t` 互換コマンド、または SMTP（設定で選択） |
| mfw の OAuth / HTTP | `golang.org/x/oauth2` |
| PHP テンプレート | `html/template` + `embed` |

### 2.2 ディレクトリ構成

PHP 版の `vendor/`（composer）がリポジトリ直下にあるため、ルートに `go.mod` を置くと Go の vendoring と衝突する。Go のコードは `go/` 以下の独立モジュール（モジュール名 `emlauncher`）にする。リポジトリには github.com 以外のミラーもあるので、モジュール名にホスト名は含めない。

パッケージは基本的にフラットにし、独立させる理由があるものだけ分ける。

```
go/                          package emlauncher: 設定, DB (table*.go), 業務処理 (core*.go), HTTP ハンドラ, セッション, メール
  templates/, static/        画面のテンプレートと静的ファイル (バイナリに埋め込む)
  storage/                   S3 / LocalFile. 外部サービスとの境界. 同じテストを両方の実装に流す
  pkgfile/                   IPA / APK / AAB の解析. アプリ本体に依存しない. テストデータを持つ
  cmd/emlauncher/            HTTP サーバ
  cmd/delete-old-packages/   バッチ
  tools/                     移行用のスクリプト (PHP 版の設定の書き出し)
e2e/                         package e2e: テスト用のクライアント・フィクスチャ (モジュール名 emlauncher-e2e)
  scenario/, migration/      シナリオテストと移行リハーサル (実行の仕方が違うので分ける)
  env/                       テスト環境 (docker compose)
docs/go-migration/           本計画書・テスト・移行手順
```

## 3. 個別の設計

### 3.1 タイムゾーン

- 設定 `timezone`。空なら `time.Local`（環境変数 `TZ` → `/etc/localtime`）、`Asia/Tokyo` などを指定すればそれに固定。
- DB 接続は `parseTime=true&loc=<上記>`。DB の DATETIME は TZ を持たないため、PHP と同じ TZ にしないと時刻がずれる。
- 起動時に使用 TZ をログに出す。
- 移行手順: 各環境で PHP が使っている TZ（`php -i | grep date.timezone`、Apache の設定や `.htaccess` の上書きも確認）を調べ、Go の設定に写す。

### 3.2 EC2 + Apache + ALB

- Go は systemd サービスとして `127.0.0.1:8080` 等で待ち受ける。Apache は `mod_proxy` でリバースプロキシする。これで `/api/*` だけ先に Go に振り向ける段階的な切替が、Apache の設定だけでできる。最終的に Apache を外して ALB から直接 Go に向けてもよい。
- URL 生成は `X-Forwarded-Proto` / `Host` を見る（`enable_https`、リダイレクト、itms-services の plist URL）。
- ALB のヘルスチェックは `/health`。
- 大きなファイル（最大 7GB）のアップロードは `MultipartReader` で一時ファイルに流し、S3 のマルチパートアップロードで送る。ALB の idle timeout と Go の `ReadTimeout` は PHP の設定（300 秒）以上にする。
- デプロイ物: バイナリ、設定ファイル、Java（bundletool / keytool 用）。PHP と composer は不要になる。

### 3.3 LocalFile ストレージ

- 設定 `storage_class: LocalFile`、`local_file.path`、`local_file.url_prefix` は PHP 版と同じ意味。
- パスの組み立て、ディレクトリ作成（0755）、rename、削除、URL の作り方は PHP 版と同じにする。既存ファイルがそのまま見える。
- 配信: Go が `url_prefix` 以下をローカルディスクから返す（パストラバーサル対策あり）。`local_file.serve: false` にすれば従来どおり Apache の Alias に任せられる。
- 移行時の注意: 既存ファイルの所有者は Apache のユーザー。Go を同じユーザーで動かすか、所有者を変更する。

### 3.4 画面

- フェーズ 6 の最初に、現行の全画面のスクリーンショットとボタン・リンクの一覧を作る。
- ヘッダのナビゲーション、各画面のボタンの位置・ラベル、タブ構成（platform フィルタ、タグ、ページング、Install ボタンなど）は一覧に合わせる。CSS や Bootstrap のバージョンは自由。
- 移植後、新旧スクリーンショットを並べて確認してもらう。

### 3.5 PHP 版の既知の不具合（踏襲しない）

シナリオテストで確認したもの（[TESTING.md](TESTING.md)）:

- **`api/delete` で最新のパッケージを消すと、1 つ前のパッケージのファイルを消し、削除対象のファイルが残る**（`$pkg` 変数の再代入）。データ消失のバグで、本番にも「DB に行があるのにファイルが無い」パッケージと、行の無いファイルがありうる。Go 版はファイルが無いパッケージがあっても動くようにする。必要なら DB とストレージの突き合わせツールを用意する。
- `api/delete` で最後のパッケージを消すと 500 を返す（DB からは消えている）。
- app / package モジュールのエラーページがステータス 200 で返る（`build()` が headers を捨てる）。Go 版は 403 / 404 などを返す。

コードを読んで見つけたもの:

- `model/storage/S3.php` の catch 節が未定義の `$s3Client` を参照している。
- `api/delete` が添付ファイルの実体を消していない（ストレージにゴミが残る）。
- いくつかの更新処理で `$con` を渡し忘れている（共有 PDO のため偶然動いている）。
- Content-Disposition のファイル名を PHP の `urlencode` でエンコードしており、空白が `+` になる。Go 版は `%20` にする。

## 4. フェーズ

| # | 内容 | 状態 |
|---|---|---|
| 0 | テスト基盤: PHP 版のシナリオテストと移行リハーサル（[TESTING.md](TESTING.md)） | 完了 |
| 1 | 基盤: config（TZ）、DB、Storage（S3 / LocalFile）、memcached、ログ、ALB 対応の URL 生成 | 完了 |
| 2 | パッケージ解析（IPA / APK / AAB）、アイコン変換 | 完了 |
| 3 | API 7 本 + health + qr | 完了（シナリオテスト合格） |
| 4 | Apache の mod_proxy で `/api/*` だけ Go に切替 | 手順は [MIGRATION.md](MIGRATION.md) に記載。本番での実施は未 |
| 5 | 認証（既存パスワードとの互換、リセットメール、Google / AzureAD）、Cookie セッション | 完了。Google / AzureAD は実際の IdP では未確認 |
| 6 | 画面移植、ajax、インストールフロー、ゲストパス、メール通知 | 完了（PHP 版のテンプレートの構造をそのまま移植したので配置は同じ）。新旧画面の目視確認と iOS / Android 実機の確認は未 |
| 7 | バッチ移植、移行手順書、移行リハーサル、切り戻しの確認 | 完了。本番切替は未 |

### 残っていること

- 各本番環境で `go/tools/export_php_config.php` を実行し, 書き出された TZ・storage_class・cache_prefix を確認する
- 画面の新旧の目視確認（配置は同じだが, CDN の Bootstrap を 3.0.0 → 3.3.7 に上げたので細部の見た目が違いうる）
- iOS / Android 実機でのインストール, Google / AzureAD ログインの確認
- （任意）PHP 版の api/delete のバグで生じた, DB とストレージの不整合の調査

## 5. 移行手順

[MIGRATION.md](MIGRATION.md)（草案）。フェーズ 7 で詳細化し、移行リハーサル（`e2e/migration/rehearse.sh <storage> go`）で検証する。
