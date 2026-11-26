package config

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// CompilePreg は PHP の preg_match 形式 ("/pattern/flags") の正規表現を Go の regexp にする.
// 区切り文字が無いもの (英数字や \ で始まるもの) はそのまま Go の正規表現として扱う.
//
// フラグは i, m, s, U を Go の同名のフラグに, u と D は無視する (Go は常に UTF-8.
// D は $ を末尾だけに一致させるもので, Go の $ は元からそう). それ以外はエラーにする.
func CompilePreg(pattern string) (*regexp.Regexp, error) {
	if pattern == "" {
		return nil, errors.New("empty pattern")
	}
	open := pattern[0]
	if isAlnum(open) || open == '\\' || open == ' ' {
		return regexp.Compile(pattern)
	}
	closeDelim := open
	switch open {
	case '(':
		closeDelim = ')'
	case '{':
		closeDelim = '}'
	case '[':
		closeDelim = ']'
	case '<':
		closeDelim = '>'
	}
	end := strings.LastIndexByte(pattern, closeDelim)
	if end <= 0 {
		return nil, fmt.Errorf("no ending delimiter %q in %q", closeDelim, pattern)
	}
	body, flags := pattern[1:end], pattern[end+1:]

	var goFlags strings.Builder
	for _, f := range flags {
		switch f {
		case 'i', 'm', 's', 'U':
			goFlags.WriteRune(f)
		case 'u', 'D', '\n', '\r', ' ':
		default:
			return nil, fmt.Errorf("unsupported flag %q in %q", f, pattern)
		}
	}
	if goFlags.Len() > 0 {
		body = "(?" + goFlags.String() + ")" + body
	}
	return regexp.Compile(body)
}

func isAlnum(c byte) bool {
	return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9'
}
