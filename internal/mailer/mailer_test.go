package mailer

import (
	"bytes"
	"encoding/base64"
	"fmt"
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
		Subject: "New Comment to アプリ\r\nBcc: evil@example.com",
		Body:    strings.Repeat("本文 body\n", 20),
	}
	b, err := m.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	msg, err := mail.ReadMessage(bytes.NewReader(b))
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

// 宛先が多くても 1 行が長くならないよう折り返し, 読むと元のアドレスに戻る.
func TestMessageBytesFoldsAddresses(t *testing.T) {
	var bcc []string
	for i := range 100 {
		bcc = append(bcc, fmt.Sprintf("user%03d@example.com", i))
	}
	b, err := Message{From: "a@example.com", To: []string{"b@example.com"}, Bcc: bcc, Body: "x"}.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(b), "\r\n") {
		if len(line) > 78 {
			t.Fatalf("line too long (%d): %q", len(line), line)
		}
	}
	msg, err := mail.ReadMessage(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	addrs, err := msg.Header.AddressList("Bcc")
	if err != nil || len(addrs) != len(bcc) || addrs[99].Address != bcc[99] {
		t.Fatalf("Bcc = %v, %v", addrs, err)
	}
}

// アドレスに改行があればヘッダを差し込めるので送らない.
func TestMessageBytesRejectsControlChars(t *testing.T) {
	for _, m := range []Message{
		{From: "a@example.com", To: []string{"b@example.com\r\nBcc: evil@example.com"}},
		{From: "a@example.com", Bcc: []string{"b@example.com\nX: y"}},
		{From: "a@example.com\r\n", To: []string{"b@example.com"}},
	} {
		if _, err := m.Bytes(); err == nil {
			t.Errorf("accepted %+v", m)
		}
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
