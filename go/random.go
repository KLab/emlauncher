// API キー・ファイル名・トークンに使うランダム文字列を作る.
// 形式は PHP 版 (model/Random.php) と同じ [0-9a-zA-Z] だが, 暗号論的乱数を使う.

package emlauncher

import (
	"crypto/rand"
	"math/big"
)

const randomChars = "1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// String は長さ n のランダムな英数字列を返す.
func randomString(n int) string {
	b := make([]byte, n)
	max := big.NewInt(int64(len(randomChars)))
	for i := range b {
		v, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic(err)
		}
		b[i] = randomChars[v.Int64()]
	}
	return string(b)
}
