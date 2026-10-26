// sendmail 互換コマンドでメールを送る (PHP 版の mb_send_mail と同じく sendmail -t 形式).

package emlauncher

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime"
	"os/exec"
	"strings"
	"time"
)

type MailMessage struct {
	From    string
	To      []string
	Bcc     []string
	Subject string
	Body    string
}

type Mailer interface {
	Send(m MailMessage) error
}

// Sendmail はメッセージを標準入力に渡してコマンドを実行する.
type Sendmail struct {
	Command string // 例: "/usr/sbin/sendmail -t -i"
}

func (s *Sendmail) Send(m MailMessage) error {
	args := strings.Fields(s.Command)
	if len(args) == 0 {
		return fmt.Errorf("mail: sendmail command is not set")
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = bytes.NewReader(formatMail(m, time.Now()))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mail: %s: %v: %s", s.Command, err, out)
	}
	return nil
}

// Format は UTF-8 / base64 のテキストメールにする.
func formatMail(m MailMessage, date time.Time) []byte {
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
	header("Date", date.Format(time.RFC1123Z))
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
