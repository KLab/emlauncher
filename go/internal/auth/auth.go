// Package auth はパスワードのハッシュとログインできるアカウントの判定.
package auth

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	random "github.com/KLabTools/emlauncher/go/internal/random"
)

// HashPassword は PHP 版 (model/UserPass.php) と同じ形式 "{stretch}:{salt}:{hash}" でハッシュを作る.
// hash = sha1(password + salt) の後, hash = sha1(hash + salt) を stretch 回 (10〜20) 繰り返す.
// 既存のユーザがそのままログインできるよう, 形式は変えない.
func HashPassword(password string) string {
	n, err := rand.Int(rand.Reader, big.NewInt(11))
	if err != nil {
		panic(err)
	}
	return hashWith(password, random.String(16), 10+int(n.Int64()))
}

func hashWith(password, salt string, stretch int) string {
	h := sha1hex(password + salt)
	for range stretch {
		h = sha1hex(h + salt)
	}
	return fmt.Sprintf("%d:%s:%s", stretch, salt, h)
}

// CheckPassword は passhash が password に一致するか.
func CheckPassword(passhash, password string) bool {
	if password == "" || passhash == "" {
		return false
	}
	parts := strings.SplitN(passhash, ":", 3)
	if len(parts) != 3 {
		return false
	}
	stretch, err := strconv.Atoi(parts[0])
	if err != nil || stretch < 0 || stretch > 1000 {
		return false
	}
	return hashWith(password, parts[1], stretch) == passhash
}

func sha1hex(s string) string {
	h := sha1.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

// CompilePHPRegexp は PHP の preg_match 形式 ("/pattern/flags") の正規表現を Go の regexp にする.
// 区切り文字が無ければそのままパターンとして扱う.
func CompilePHPRegexp(p string) (*regexp.Regexp, error) {
	if len(p) >= 2 {
		delim := p[0]
		if !(delim >= 'a' && delim <= 'z' || delim >= 'A' && delim <= 'Z' || delim >= '0' && delim <= '9' || delim == '\\' || delim == ' ') {
			end := delim
			switch delim {
			case '(':
				end = ')'
			case '{':
				end = '}'
			case '[':
				end = ']'
			case '<':
				end = '>'
			}
			if i := strings.LastIndexByte(p, end); i > 0 {
				pattern, flags := p[1:i], p[i+1:]
				prefix := ""
				for _, f := range flags {
					switch f {
					case 'i', 'm', 's', 'U':
						prefix += string(f)
					case 'u', 'D':
					default:
						return nil, fmt.Errorf("unsupported regexp flag %q", f)
					}
				}
				if prefix != "" {
					pattern = "(?" + prefix + ")" + pattern
				}
				return regexp.Compile(pattern)
			}
		}
	}
	return regexp.Compile(p)
}
