// Package emlauncher は画面のテンプレートと静的ファイルを埋め込む.
// プログラムの本体は internal/ 以下, コマンドは cmd/ 以下にある.
package emlauncher

import "embed"

// Templates は templates/ 以下の画面のテンプレート (html/template).
//
//go:embed templates
var Templates embed.FS

// Static は static/ 以下の静的ファイル (CSS, アイコン).
//
//go:embed static
var Static embed.FS
