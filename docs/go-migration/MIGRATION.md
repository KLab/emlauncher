# PHP 版から Go 版への移行手順（草案）

Go 版の実装に合わせて詳細化し、移行リハーサル（`e2e/migration/rehearse.sh <storage> go`）で検証する。リハーサルのスクリプトはこの手順書と同じ順序で実行すること。

## 0. 事前確認（環境ごと）

| 項目 | 確認方法 | Go 版の設定 |
|---|---|---|
| PHP の TZ | `php -i \| grep date.timezone`。Apache の設定や `.htaccess` で上書きしていないかも確認 | `timezone`（空なら実行環境の TZ） |
| storage_class | `config/emlauncher_config.php` | `storage_class`（S3 / LocalFile） |
| memcached のキー接頭辞 | `{application_identifier}_{MFW_ENV}_{PHP を置いたディレクトリ名}_`（mfwServerEnv::cachePrefix） | `cache_prefix` を同じ値にする。違うと移行前に発行したインストールトークンが使えなくなる |
| DB の接続情報 | `config/mfw_serverenv_config.php` と authfile | `database` |
| メール | `mail_sender`、`mail_bcc_to`、sendmail の設定 | `mail` |
| ログイン方法 | `login`（password / Google / AzureAD、`allowed_mailaddr_pattern`） | `login`（コールバック URL は変えない） |
| bundletool / keystore | `apkfile` | `apkfile`（Java は引き続き必要） |

## 1. 準備

1. Go 版のバイナリと設定ファイルを配置する。設定は 0. の表のとおり PHP の設定から作る。
2. LocalFile の場合、保存先のファイルの所有者を確認する。Go のサービスを Apache と同じユーザで動かすか、所有者を変更する。
3. Go 版を systemd サービスとして起動する（この時点ではまだリクエストは来ない）。`/health` を確認する。

## 2. 切替

1. Apache のリバースプロキシ先を Go 版に向ける（段階移行中は `/api/*` のみ、最終的に全体）。PHP 版は停止する。
2. LocalFile で Apache の Alias（`/files`）を使っていた場合は、Go 版の配信に切り替えるか Alias を残す（`local_file.serve`）。
3. cron の `batch/delete_old_packages.php` を Go 版のバッチに差し替える。

## 3. 切替後

- 利用者は再ログインする（セッションは引き継がない）。
- 移行前に発行したインストールトークン、ゲストパスの URL はそのまま使える。
- DB のスキーマ変更・データ移行は無い。切り戻しは、リバースプロキシ先を PHP 版に戻すだけでよい（Go 版で作ったデータも PHP 版でそのまま扱える）。
