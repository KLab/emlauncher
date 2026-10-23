package e2e

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime"
	"mime/quotedprintable"
	"net/mail"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"
)

// Mailbox は送信メールを置くディレクトリ (1 通 1 ファイル, sendmail -t 形式) を読む.
type Mailbox struct {
	t   testing.TB
	dir string
}

type Mail struct {
	File       string
	Header     mail.Header
	Subject    string
	Body       string
	Recipients []string // To / Cc / Bcc のアドレス
}

func (m *Mailbox) All() []Mail {
	m.t.Helper()
	ents, err := os.ReadDir(m.dir)
	if err != nil {
		m.t.Fatalf("read mailbox %s: %v", m.dir, err)
	}
	var names []string
	for _, e := range ents {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	var out []Mail
	for _, n := range names {
		raw, err := os.ReadFile(filepath.Join(m.dir, n))
		if err != nil {
			m.t.Fatal(err)
		}
		out = append(out, parseMail(m.t, n, raw))
	}
	return out
}

// To は address 宛て (To/Cc/Bcc のいずれか) のメールを返す.
func (m *Mailbox) To(address string) []Mail {
	m.t.Helper()
	var out []Mail
	for _, ml := range m.All() {
		if slices.Contains(ml.Recipients, address) {
			out = append(out, ml)
		}
	}
	return out
}

// WaitTo は address 宛てのメールが n 通以上届くまで待つ.
func (m *Mailbox) WaitTo(address string, n int) []Mail {
	m.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		got := m.To(address)
		if len(got) >= n {
			return got
		}
		if time.Now().After(deadline) {
			m.t.Fatalf("mail to %s: got %d, want %d", address, len(got), n)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func parseMail(t testing.TB, name string, raw []byte) Mail {
	t.Helper()
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("parse mail %s: %v\n%s", name, err, raw)
	}
	dec := new(mime.WordDecoder)
	subject, err := dec.DecodeHeader(msg.Header.Get("Subject"))
	if err != nil {
		subject = msg.Header.Get("Subject")
	}
	body, _ := io.ReadAll(msg.Body)
	switch strings.ToLower(msg.Header.Get("Content-Transfer-Encoding")) {
	case "base64":
		if b, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(string(body)), "")); err == nil {
			body = b
		}
	case "quoted-printable":
		if b, err := io.ReadAll(quotedprintable.NewReader(bytes.NewReader(body))); err == nil {
			body = b
		}
	}
	var rcpts []string
	for _, h := range []string{"To", "Cc", "Bcc"} {
		v := msg.Header.Get(h)
		if v == "" {
			continue
		}
		addrs, err := (&mail.AddressParser{WordDecoder: dec}).ParseList(v)
		if err != nil {
			t.Fatalf("parse %s %q in %s: %v", h, v, name, err)
		}
		for _, a := range addrs {
			rcpts = append(rcpts, a.Address)
		}
	}
	return Mail{File: name, Header: msg.Header, Subject: subject, Body: string(body), Recipients: rcpts}
}
