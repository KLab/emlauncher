package mailer

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime"
	"net/mail"
	"strings"
	"testing"
)

func TestMessageBytes(t *testing.T) {
	m := Message{
		From:    "EMLauncher <no-reply@example.com>",
		To:      []string{"EMLauncher <bcc@example.com>"},
		Bcc:     []string{"a@example.com", "b@example.com"},
		Subject: "New Comment to アプリ",
		Body:    strings.Repeat("本文 body\n", 20),
	}
	msg, err := mail.ReadMessage(bytes.NewReader(m.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	subject, err := new(mime.WordDecoder).DecodeHeader(msg.Header.Get("Subject"))
	if err != nil || subject != m.Subject {
		t.Errorf("Subject = %q, %v", subject, err)
	}
	if got := msg.Header.Get("Bcc"); got != "a@example.com, b@example.com" {
		t.Errorf("Bcc = %q", got)
	}
	raw, _ := io.ReadAll(msg.Body)
	body, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(string(raw)), ""))
	if err != nil || string(body) != m.Body {
		t.Errorf("body = %q, %v", body, err)
	}
}

func TestNewSendmail(t *testing.T) {
	if _, err := NewSendmail("  "); err == nil {
		t.Fatal("empty command accepted")
	}
	s, err := NewSendmail("/usr/sbin/sendmail -t -i")
	if err != nil || len(s.command) != 3 {
		t.Fatalf("NewSendmail: %v %v", s, err)
	}
}
