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
| パッケージ | `package/{app_id}/{pkg_id}_{package.file_name}` | private / apk: `application/vnd.android.package-archive`、ipa・aab: `application/octet-stream` |
| 添付ファイル | `package/{app_id}/{pkg_id}_{attached_id}_{attached_file.file_name}` | private / apk は上と同じ、それ以外はアップロード時の type |
| 一時ファイル | `temp-data/{rand16}.{ext}` | private。確定時に copy + delete で正式キーへ移動 |

- ダウンロードは 60 分有効の署名付き URL。`ResponseContentDisposition: attachment; filename*=UTF-8''{urlencode(original_file_name)}`
- `aws.base_url` 指定時は path-style。署名付き URL の `base_url` 部分を `external_url` に置換して返す。
- **LocalFile**: 保存先は `rtrim(local_file.path,'/') + '/' + trim(key,'/')`、URL は `rtrim(url_prefix,'/') + '/' + key`。有効期限なし。配信は Apache の Alias で行っている。

### 1.4 memcached

キーの先頭には `{application_identifier}_{env}_{branch}_` が付く（mfwServerEnv::cachePrefix）。

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
3. **API の JSON は形と型を合わせる。** `PDO::ATTR_STRINGIFY_FETCHES` 有効のため、PHP 版は数値カラムを文字列で返す（`"id":"12"`）。`protect` は bool。シナリオテストで PHP 版の型を固定し、Go 版も合わせる。
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

PHP 版の `vendor/`（composer）がリポジトリ直下にあるため、ルートに `go.mod` を置くと Go の vendoring と衝突する。Go のコードは `go/` 以下の独立モジュールにする。

```
go/
  cmd/emlauncher/            HTTP サーバ
  cmd/delete-old-packages/   バッチ
  internal/config            設定（環境ごとの切替、TZ）
  internal/db, internal/model
  internal/storage           S3 / LocalFile（共通インタフェース）
  internal/pkgfile           IPA / APK / AAB 解析
  internal/session, internal/auth
  internal/mail
  internal/web               ルーティング・ハンドラ・テンプレート
e2e/                         シナリオテスト（PHP 版・Go 版共用、独立モジュール）
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

- `model/storage/S3.php` の catch 節が未定義の `$s3Client` を参照している。
- `api/delete` が添付ファイルの実体を消していない（ストレージにゴミが残る）。
- いくつかの更新処理で `$con` を渡し忘れている（共有 PDO のため偶然動いている）。
- Content-Disposition のファイル名を PHP の `urlencode` でエンコードしており、空白が `+` になる。Go 版は `%20` にする。

## 4. フェーズ

| # | 内容 | 完了の条件 |
|---|---|---|
| 0 | テスト基盤: PHP 版のシナリオテストと中間状態の作成・検証テスト（[TESTING.md](TESTING.md)）。各環境の TZ と storage_class を確認 | シナリオテストが PHP 版で全て通る |
| 1 | 基盤: config（TZ）、DB、Storage（S3 / LocalFile）、memcached、ログ、ALB 対応の URL 生成 | Storage のテストが S3（エミュレータ）と LocalFile の両方で通る |
| 2 | パッケージ解析（IPA / APK / AAB）、アイコン変換 | テスト用フィクスチャで identifier が PHP 版と一致 |
| 3 | API 7 本 + health + qr | API のシナリオテストが Go 版で通る |
| 4 | Apache の mod_proxy で `/api/*` だけ Go に切替 | CI からのアップロードが本番で動く |
| 5 | 認証（既存パスワードとの互換、リセットメール、Google / AzureAD）、Cookie セッション | ログインのシナリオテストが通る |
| 6 | 画面の一覧作成 → 画面移植 → 新旧スクリーンショット確認、ajax、インストールフロー、ゲストパス、メール通知 | 全シナリオテストが Go 版で通る。iOS / Android 実機でインストールできる |
| 7 | バッチ移植、移行手順書の作成、**移行リハーサルテスト**、全体切替（systemd 化）、PHP 撤去 | 移行リハーサルテストが通る。本番運用 |

## 5. 移行手順（概要。フェーズ 7 で詳細化し、テストで検証する）

1. Go 版の設定ファイルを PHP の設定（`config/emlauncher_config.php`, `config/mfw_serverenv_config.php`）から作る。TZ を PHP と揃える。
2. LocalFile 環境ではファイルの所有者・権限を確認する。
3. Go を起動し、Apache のリバースプロキシ先を Go に切り替える（PHP は停止）。
4. 利用者は再ログインする。発行済みのインストールトークン URL、ゲストパス URL はそのまま使える。
5. cron の `delete_old_packages.php` を Go 版のバッチに差し替える。
