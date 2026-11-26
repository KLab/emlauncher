// Package mailer は sendmail 互換のコマンドでメールを送る.
package mailer

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"os/exec"
	"strings"
	"time"
)

// Message は送るメール. 本文は UTF-8 のプレーンテキスト.
type Message struct {
	From    string
	To      []string
	Bcc     []string
	Subject string
	Body    string
}

// Sendmail は "sendmail -t -i" のようにメッセージを標準入力で受け取るコマンドで送る.
// 宛先はヘッダ (To / Bcc) から読ませる.
type Sendmail struct {
	command []string
}

// NewSendmail は空白で区切ったコマンドラインから Sendmail を作る.
func NewSendmail(commandLine string) (*Sendmail, error) {
	cmd := strings.Fields(commandLine)
	if len(cmd) == 0 {
		return nil, errors.New("mailer: empty sendmail command")
	}
	return &Sendmail{command: cmd}, nil
}

func (s *Sendmail) Send(ctx context.Context, m Message) error {
	msg, err := m.Bytes()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, s.command[0], s.command[1:]...)
	cmd.Stdin = bytes.NewReader(msg)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mailer: %s: %w: %s", s.command[0], err, bytes.TrimSpace(out))
	}
	return nil
}

const (
	crlf = "\r\n"
	// maxLineLength はヘッダの行を折り返す長さ (RFC 5322 の推奨は 78 文字以内, 上限は 998 文字).
	maxLineLength = 76
)

// Bytes は sendmail -t に渡す形式 (ヘッダと base64 の本文) にする.
// アドレスに改行などの制御文字があればエラーにする (ヘッダを差し込ませないため).
// 件名は常に MIME の encoded-word にするので, 改行があってもヘッダは壊れない.
func (m Message) Bytes() ([]byte, error) {
	for _, a := range append(append([]string{m.From}, m.To...), m.Bcc...) {
		if strings.ContainsFunc(a, isControl) {
			return nil, fmt.Errorf("mailer: invalid address %q", a)
		}
	}
	var b bytes.Buffer
	header := func(k, v string) {
		if v != "" {
			b.WriteString(k + ": " + v + crlf)
		}
	}
	header("From", m.From)
	header("To", foldAddresses("To", m.To))
	header("Bcc", foldAddresses("Bcc", m.Bcc))
	header("Subject", mime.BEncoding.Encode("UTF-8", m.Subject))
	header("MIME-Version", "1.0")
	header("Content-Type", "text/plain; charset=UTF-8")
	header("Content-Transfer-Encoding", "base64")
	b.WriteString(crlf)

	enc := base64.StdEncoding.EncodeToString([]byte(m.Body))
	for len(enc) > 76 {
		b.WriteString(enc[:76] + crlf)
		enc = enc[76:]
	}
	b.WriteString(enc + crlf)
	return b.Bytes(), nil
}

// foldAddresses はアドレスを ", " で区切って並べ, 行が長くなるところで折り返す
// (宛先が多い Bcc でも 1 行が 998 文字を超えないように).
func foldAddresses(name string, addrs []string) string {
	var b strings.Builder
	lineLen := len(name) + 2
	for i, a := range addrs {
		if i > 0 {
			b.WriteString(",")
			lineLen++
			if lineLen+1+len(a) > maxLineLength {
				b.WriteString(crlf)
				lineLen = 0
			}
			b.WriteString(" ")
			lineLen++
		}
		b.WriteString(a)
		lineLen += len(a)
	}
	return b.String()
}

func isControl(r rune) bool { return r < 0x20 || r == 0x7f }
