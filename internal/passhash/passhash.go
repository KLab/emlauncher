// Package passhash は user_pass.passhash の形式 (PHP 版と互換) のパスワードハッシュを扱う.
//
// 形式は "{stretch}:{salt}:{hash}". hash = sha1(password + salt) の後,
// hash = sha1(hash + salt) を stretch 回繰り返した 16 進文字列.
package passhash

import (
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/KLab/emlauncher-go/internal/randstr"
)

const (
	minStretch = 10
	maxStretch = 20
	saltLength = 16
	// 不正な値で極端に重い計算をしないための上限
	stretchLimit = 100000
)

// New は password のハッシュを作る. stretch は 10〜20, salt は英数字 16 文字.
func New(password string) string {
	n, err := rand.Int(rand.Reader, big.NewInt(maxStretch-minStretch+1))
	if err != nil {
		panic("passhash: " + err.Error())
	}
	stretch := minStretch + int(n.Int64())
	salt := randstr.New(saltLength)
	return fmt.Sprintf("%d:%s:%s", stretch, salt, calc(password, salt, stretch))
}

// Check は password が passhash に一致するか確かめる. 空のパスワードや形式の誤りは不一致.
func Check(passhash, password string) bool {
	if password == "" || passhash == "" {
		return false
	}
	parts := strings.SplitN(passhash, ":", 3)
	if len(parts) != 3 {
		return false
	}
	stretch, err := strconv.Atoi(parts[0])
	if err != nil || stretch < 0 || stretch > stretchLimit {
		return false
	}
	want := calc(password, parts[1], stretch)
	return subtle.ConstantTimeCompare([]byte(want), []byte(parts[2])) == 1
}

func calc(password, salt string, stretch int) string {
	h := sha1hex(password + salt)
	for range stretch {
		h = sha1hex(h + salt)
	}
	return h
}

func sha1hex(s string) string {
	sum := sha1.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}
