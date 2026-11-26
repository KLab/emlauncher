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
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, s.command[0], s.command[1:]...)
	cmd.Stdin = bytes.NewReader(m.Bytes())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mailer: %s: %w: %s", s.command[0], err, bytes.TrimSpace(out))
	}
	return nil
}

// Bytes は sendmail -t に渡す形式 (ヘッダと base64 の本文) にする.
func (m Message) Bytes() []byte {
	var b bytes.Buffer
	header := func(k, v string) {
		if v != "" {
			fmt.Fprintf(&b, "%s: %s\r\n", k, v)
		}
	}
	header("From", m.From)
	header("To", strings.Join(m.To, ", "))
	header("Bcc", strings.Join(m.Bcc, ", "))
	header("Subject", mime.BEncoding.Encode("UTF-8", m.Subject))
	header("MIME-Version", "1.0")
	header("Content-Type", "text/plain; charset=UTF-8")
	header("Content-Transfer-Encoding", "base64")
	b.WriteString("\r\n")

	enc := base64.StdEncoding.EncodeToString([]byte(m.Body))
	for len(enc) > 76 {
		b.WriteString(enc[:76] + "\r\n")
		enc = enc[76:]
	}
	b.WriteString(enc + "\r\n")
	return b.Bytes()
}
