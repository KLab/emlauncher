# 設計メモ

コードを読むだけでは分かりにくい前提と, DB・ストレージ・memcached の暗黙の仕様をまとめる.

## 前提

- **DB スキーマとストレージの構成（キー, S3 の ACL と ContentType）は PHP 版から変えない.** Go 版はマイグレーションを持たず, 既存のスキーマ（`sql/tables.sql`）に合わせた手書きの SQL で読み書きする. PHP 版と同じ DB・ストレージを共有でき, 切り戻しもできる.
- **URL とリクエストパラメータ名は PHP 版と同じ**（`/{module}/{action}?...`）. API は CI などの外部から使われるため. 画面の form のパラメータ名も, E2E テストを PHP 版と共用するため変えない.
- **OAuth のコールバック URL**（`/login/google_callback`, `/login/azuread_callback`）は変えない.
- PHP 版との互換は 1 年程度（2027 年 11 月ごろまで）保つ想定. それまでは上記を変えない. PHP 版の最終版はコミット `4124ca6`.

## ディレクトリ構成

パッケージは基本的にフラットにし, 独立させる理由があるものだけ分ける.

```
src/                         package emlauncher: 設定, DB (table*.go), 業務処理 (core*.go), HTTP ハンドラ, セッション, メール
  config.sample.yml          設定例
  templates/, static/        画面のテンプレートと静的ファイル (バイナリに埋め込む)
  storage/                   S3 / LocalFile. 外部サービスとの境界. 同じテストを両方の実装に流す
  pkgfile/                   IPA / APK / AAB の解析. アプリ本体に依存しない. テストデータを持つ
  cmd/emlauncher/            HTTP サーバ
  cmd/delete-old-packages/   バッチ
sql/                         DB スキーマ (PHP 版から変えていない)
tools/                       移行用のスクリプト (PHP 版の設定の書き出し)
e2e/                         E2E テストと移行リハーサル. テスト専用の依存 (AWS SDK など) を本体に
                             持ち込まないよう別モジュール (emlauncher-e2e)
docs/                        設計メモ, PHP 版からの移行手順
```

モジュール名は `emlauncher`. リポジトリには github.com 以外のミラーもあるので, モジュール名にホスト名は含めない.

## DB

テーブルの定義は `sql/tables.sql`（12 テーブル, utf8mb4）. スキーマに表れない仕様:

- **日時**: DATETIME は TZ を持たないので, 設定 `timezone` のローカル時刻で読み書きする（PHP 版は `date('Y-m-d H:i:s')`）. PHP 版と同じ DB を使う間は, PHP 版と同じ TZ にしないと時刻がずれる. `timezone` が空なら `time.Local`（環境変数 `TZ` → `/etc/localtime`）. 起動時に使用する TZ をログに出す.
- **パスワード**: `user_pass.passhash` は `"{stretch}:{salt}:{hash}"`. `hash = sha1(pass.salt)` の後, `hash = sha1(hash.salt)` を stretch 回繰り返す. stretch は 10〜20, salt は英数字 16 文字.
- **date_to_sort**: `application.date_to_sort` = max(created, last_upload, last_commented).
- **last_upload の巻き戻し**: パッケージ削除時, `application.last_upload == package.created` なら, 残っている最新のパッケージの created（無ければ NULL）に戻す.
- **コメント番号**: `comment.number` はアプリごとの「最大値 + 1」.
- **notify**: `app_install_user.notify` はスキーマのコメントだと「0: 送る」だが, PHP 版のコードは**真なら送る**. コードに従う.
- **ファイル名・API キー・トークン**: `[0-9a-zA-Z]` の 16 / 32 文字. crypto/rand で作る.
- **package_tag**: パッケージとタグの対応. タグ削除時は package_tag から先に消す.

## ストレージ

キーの構成は S3 / LocalFile 共通.

| 用途 | キー | S3 ACL / ContentType |
|---|---|---|
| アイコン | `app-icons/{app_id}/{rand16}.png`（144×144 PNG） | public-read / `image/png`. URL は署名なし |
| パッケージ | `package/{app_id}/{pkg_id}_{package.file_name}` | private / `application/octet-stream`（※） |
| 添付ファイル | `package/{app_id}/{pkg_id}_{attached_id}_{attached_file.file_name}` | private / `application/octet-stream`（※） |
| 一時ファイル | `temp-data/{rand16}.{ext}` | private / `application/octet-stream`（※）. 確定時に copy + delete で正式なキーへ移す |

※ PHP 版はコード上は apk に `application/vnd.android.package-archive` などを指定していたが, aws-sdk-php の MultipartUploader がその設定を読まず, 実際にはすべて `application/octet-stream` で保存されていた. 既存のオブジェクトと揃えるため, Go 版もこれに合わせている.

- ダウンロードは 60 分有効の署名付き URL. `ResponseContentDisposition: attachment; filename*=UTF-8''{元のファイル名をパーセントエンコード}`.
- `aws.base_url` を指定したときは path-style. 署名付き URL の `base_url` の部分を `external_url` に置き換えて返す（サーバから見たエンドポイントとブラウザから見たエンドポイントが違う S3 互換ストレージ向け）.
- **LocalFile**: 保存先は `rtrim(local_file.path, '/') + '/' + trim(key, '/')`, URL は `rtrim(url_prefix, '/') + '/' + key`. 有効期限なし. ディレクトリは 0755 で作る. Go 版が `url_prefix` 以下をローカルディスクから配信する（パストラバーサル対策あり）. `local_file.serve: false` なら Apache の Alias などに任せる.

## memcached

キーの先頭には設定 `cache_prefix` が付く. PHP 版の `{application_identifier}_{env}_{PHP を置いたディレクトリ名}_` と同じにすると, インストールトークンを PHP 版と共有できる.

| キー | 値 | 有効期限 | PHP 版との互換 |
|---|---|---|---|
| `pkg_install_token_{token}` | JSON 文字列 `{"mail","package_id","expire"}` | 発行時に指定（API は任意の時間） | あり（同じキー・同じ形式） |
| `ios_plist_token_{token}` | pkg_id | 60 秒 | 形式は同じ（短命なので互換は必須ではない） |
| `{sha1}`（パスワードリセット） | JSON | 30 分 | なし（PHP 版は PHP のシリアライズ形式） |

値は文字列（バイト列）として読み書きする. 30 日を超える有効期限は memcached の仕様に合わせて絶対時刻で渡す.

## セッション

署名付き Cookie に, ログイン中のメールアドレス, ログイン前の URL, 削除確認用の CSRF トークン, OAuth の state と失敗時のメールアドレスを入れる. サーバ側に状態を持たないので, ALB の裏で複数台にしてもスティッキーセッションは要らない（全台で `session.secret` を揃える）. PHP 版のファイルセッションとの互換は無い.

## API

JSON のキーは PHP 版と同じ. 数値フィールド（id, file_size など）は文字列で返す. API ドキュメントの Caveats で「文字列か数値か不定」としており, PHP 版も DB から読んだ値は文字列で返していた. `protect` は bool.

## HTTP

- URL は `X-Forwarded-Proto` / `X-Forwarded-Host` を見て組み立てる（`enable_https`, リダイレクト, itms-services の plist URL）.
- 大きなファイル（最大 7GB）のアップロードは, リクエストを一時ファイル（`TMPDIR`）に流し, S3 にはマルチパートアップロードで送る.
- テンプレートと静的ファイルはバイナリに埋め込む. 更新は無停止の再起動で行う.

## 無停止の再起動

`cloudflare/tableflip` を使う. SIGHUP（`systemctl reload`）を受けると, 新しいプロセスに待ち受けソケットを引き継ぎ, 古いプロセスは処理中のリクエストを終えてから終了する. 新しいプロセスは設定を読み直す. 起動に失敗したら古いプロセスが動き続ける. E2E テスト（`TestGracefulRestart*`, `TestSystemdStopStart`）で, Docker と systemd の両方で確認している.

- systemd では, 古いプロセスが処理中のリクエストを終えるまでは `MainPID` は古いプロセスのまま. 古いプロセスが終了した時点で PID ファイルから新しいプロセスに移る. その後に新しいプロセスが異常終了しても `Restart=always` で再起動される. `systemctl stop` / `start` も効く.
- `systemctl reload` はシグナルを送った時点で成功を返す. 新しいプロセスの起動に失敗したかはログ（`upgrade failed`）で確認する.
- 古いプロセスが処理中のリクエストを終えるまでは, 次の reload は拒否される（tableflip の仕様. ログに `parent hasn't exited`）.
- ユニットの `ExecReload` は `/bin/sh -c 'kill -HUP $MAINPID'` にしている（`/bin/kill` が無い環境があるため）.
