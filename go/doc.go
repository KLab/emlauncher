// Package emlauncher は EMLauncher (Go 版) の本体.
//
// DB スキーマとストレージの構成は PHP 版と同じ. URL とリクエストパラメータ名も PHP 版と同じ (/{module}/{action}?...).
// ストレージ (storage) とパッケージ解析 (pkgfile) 以外は, このパッケージにまとめている.
//
// 主なファイル:
//   - config.go, db.go, cache.go, mail.go, session.go, clock.go: 設定と外部との接続
//   - table*.go: 各テーブルの読み書き
//   - core*.go: API と画面で共通の業務処理
//   - server.go, ui*.go, api.go: HTTP ハンドラ. templates/ は画面のテンプレート
package emlauncher
