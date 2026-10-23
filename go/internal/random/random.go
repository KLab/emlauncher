// Package random は API キー・ファイル名・トークンに使うランダム文字列を作る.
// 形式は PHP 版 (model/Random.php) と同じ [0-9a-zA-Z] だが, 暗号論的乱数を使う.
package random

import (
	"crypto/rand"
	"math/big"
)

const chars = "1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// String は長さ n のランダムな英数字列を返す.
func String(n int) string {
	b := make([]byte, n)
	max := big.NewInt(int64(len(chars)))
	for i := range b {
		v, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic(err)
		}
		b[i] = chars[v.Int64()]
	}
	return string(b)
}
