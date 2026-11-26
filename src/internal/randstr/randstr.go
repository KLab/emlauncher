// Package randstr は英数字のランダム文字列を作る (ファイル名, API キー, トークン).
package randstr

import (
	"crypto/rand"
	"math/big"
)

const alphabet = "1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var alphabetLen = big.NewInt(int64(len(alphabet)))

// New は [0-9a-zA-Z] からなる n 文字の文字列を crypto/rand で作る.
func New(n int) string {
	b := make([]byte, n)
	for i := range b {
		k, err := rand.Int(rand.Reader, alphabetLen)
		if err != nil {
			// crypto/rand が失敗するのは OS の乱数源が使えないときだけで, 続行できない.
			panic("randstr: " + err.Error())
		}
		b[i] = alphabet[k.Int64()]
	}
	return string(b)
}
