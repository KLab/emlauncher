package emlauncher

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFormat(t *testing.T) {
	raw := formatMail(MailMessage{
		From:    "EMLauncher <no-reply@example.com>",
		To:      []string{"EMLauncher <bcc@example.com>"},
		Bcc:     []string{"a@example.com", "b@example.com"},
		Subject: "New Comment to アプリ",
		Body:    strings.Repeat("本文 body\n", 20),
	}, time.Unix(0, 0))
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	subject, _ := new(mime.WordDecoder).DecodeHeader(msg.Header.Get("Subject"))
	if subject != "New Comment to アプリ" {
		t.Errorf("subject = %q", subject)
	}
	bcc, _ := msg.Header.AddressList("Bcc")
	if len(bcc) != 2 || bcc[1].Address != "b@example.com" {
		t.Errorf("bcc = %v", bcc)
	}
	enc, _ := io.ReadAll(msg.Body)
	body, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(string(enc)), ""))
	if err != nil || string(body) != strings.Repeat("本文 body\n", 20) {
		t.Errorf("body = %q, %v", body, err)
	}
}

func TestSendmail(t *testing.T) {
	out := filepath.Join(t.TempDir(), "mail.eml")
	script := filepath.Join(t.TempDir(), "sendmail.sh")
	os.WriteFile(script, []byte("#!/bin/sh\ncat > "+out+"\n"), 0o755)
	s := &Sendmail{Command: script + " -t -i"}
	if err := s.Send(MailMessage{From: "a@example.com", To: []string{"b@example.com"}, Subject: "s", Body: "b"}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(out)
	if !bytes.Contains(b, []byte("To: b@example.com")) {
		t.Errorf("mail = %q", b)
	}
	if err := (&Sendmail{Command: "/nonexistent -t"}).Send(MailMessage{}); err == nil {
		t.Error("no error for missing command")
	}
}
