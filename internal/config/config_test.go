package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCompilePreg(t *testing.T) {
	for _, tc := range []struct {
		pattern string
		match   []string
		nomatch []string
	}{
		{`/@example\.com$/`, []string{"a@example.com"}, []string{"a@example.org", "a@EXAMPLE.com"}},
		{`/@example\.com$/i`, []string{"a@EXAMPLE.com"}, []string{"a@example.org"}},
		{`#^[a-z]+@(foo|bar)\.jp$#u`, []string{"x@foo.jp", "y@bar.jp"}, []string{"x@baz.jp"}},
		{`{@example\.(com|net)$}`, []string{"a@example.net"}, []string{"a@example.org"}},
		{`example\.com$`, []string{"a@example.com"}, []string{"a@example.org"}},
	} {
		re, err := CompilePreg(tc.pattern)
		if err != nil {
			t.Fatalf("%s: %v", tc.pattern, err)
		}
		for _, s := range tc.match {
			if !re.MatchString(s) {
				t.Errorf("%s does not match %q", tc.pattern, s)
			}
		}
		for _, s := range tc.nomatch {
			if re.MatchString(s) {
				t.Errorf("%s matches %q", tc.pattern, s)
			}
		}
	}
	for _, bad := range []string{"", "/abc", "/abc/x", "/(/"} {
		if _, err := CompilePreg(bad); err == nil {
			t.Errorf("%q: no error", bad)
		}
	}
}

const minimal = `
session:
  secret: 0123456789abcdef
database:
  user: u
storage_class: LocalFile
local_file:
  path: /tmp/files
`

func TestParseDefaults(t *testing.T) {
	c, err := Parse([]byte(minimal))
	if err != nil {
		t.Fatal(err)
	}
	if c.Listen != "127.0.0.1:8080" || c.Memcache.Addr() != "localhost:11211" || c.LocalFile.URLPrefix != "/files" || !c.LocalFile.Serve {
		t.Errorf("defaults not applied: %+v", c)
	}
	if c.Location != time.Local {
		t.Errorf("Location = %v, want Local", c.Location)
	}
	if c.MailBccRecipient() != "" {
		t.Errorf("MailBccRecipient = %q", c.MailBccRecipient())
	}
}

func TestParseRejectsUnknownKey(t *testing.T) {
	if _, err := Parse([]byte(minimal + "\nno_such_key: 1\n")); err == nil {
		t.Fatal("unknown key accepted")
	}
}

func TestParseErrors(t *testing.T) {
	for _, tc := range []struct{ name, yaml, want string }{
		{"short secret", strings.Replace(minimal, "0123456789abcdef", "short", 1), "session.secret"},
		{"bad storage", strings.Replace(minimal, "LocalFile", "FTP", 1), "storage_class"},
		{"bad tz", minimal + "timezone: Nowhere/City\n", "timezone"},
		{"bad pattern", minimal + "login:\n  allowed_mailaddr_pattern: '/a/x'\n", "allowed_mailaddr_pattern"},
		{"s3 without bucket", strings.Replace(minimal, "LocalFile", "S3", 1), "aws.bucket_name"},
		{"s3 without region", strings.Replace(minimal, "LocalFile", "S3", 1) + "aws:\n  bucket_name: b\n", "aws.region"},
	} {
		_, err := Parse([]byte(tc.yaml))
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want containing %q", tc.name, err, tc.want)
		}
	}
}

// tools/export_php_config.php の出力 (JSON) を読める.
func TestParseExportedJSON(t *testing.T) {
	dir := t.TempDir()
	auth := filepath.Join(dir, "dbauth")
	if err := os.WriteFile(auth, []byte("emlauncher:secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	js := `{
    "listen": "127.0.0.1:8080",
    "timezone": "Asia/Tokyo",
    "cache_prefix": "emlauncher_e2e_local_emlauncher_",
    "title_prefix": "[E2E] ",
    "enable_https": false,
    "mail_sender": "EMLauncher <no-reply@example.com>",
    "mail_bcc_to": "",
    "upload_lock_wait_timeout": 0,
    "mail": {"sendmail": "/usr/local/bin/capture-mail -t -i"},
    "login": {
        "enable_password": true, "enable_google_auth": false, "google_app_id": "", "google_app_secret": "",
        "allowed_mailaddr_pattern": "", "enable_azuread_auth": false, "azuread_app_id": "", "azuread_app_secret": ""
    },
    "package": {"file_size_warning_ios": 150, "file_size_warning_android": 100},
    "session": {"secret": "3f2a0c9d8e7b6a5f4e3d2c1b0a9f8e7d"},
    "database": {"host": "db", "port": 3306, "name": "emlauncher", "authfile": "` + filepath.ToSlash(auth) + `"},
    "memcache": {"host": "memcached", "port": 11211},
    "storage_class": "LocalFile",
    "aws": {"key": "test", "secret": "test", "region": "ap-northeast-1", "bucket_name": "b", "base_url": "", "external_url": ""},
    "local_file": {"path": "/var/www/emlauncher", "url_prefix": "/files", "serve": true},
    "apkfile": {"java": "java -Xmx1g", "bundletool": "/bundletool.jar", "keystore": "/k", "kspass": "pass:x", "keyalias": "a", "keypass": "pass:x", "aapt2": ""}
}`
	c, err := Parse([]byte(js))
	if err != nil {
		t.Fatal(err)
	}
	if c.Database.User != "emlauncher" || c.Database.Password != "secret" {
		t.Errorf("authfile: user=%q password=%q", c.Database.User, c.Database.Password)
	}
	if c.Location.String() != "Asia/Tokyo" {
		t.Errorf("Location = %v", c.Location)
	}
	if c.MailBccRecipient() != "EMLauncher <no-reply@example.com>" {
		t.Errorf("MailBccRecipient = %q", c.MailBccRecipient())
	}
	if c.FileSizeWarningMB("iOS") != 150 || c.FileSizeWarningMB("unknown") != 0 {
		t.Errorf("FileSizeWarningMB")
	}
}

// authfile は PHP 版と同じく, 前後の空白を除いて ":" で分ける.
func TestReadAuthFile(t *testing.T) {
	for content, want := range map[string][2]string{
		"user:pass\n":     {"user", "pass"},
		"  user:pass  \n": {"user", "pass"},
		"user":            {"user", ""},
		"user:pa:ss":      {"user", "pa"},
	} {
		p := filepath.Join(t.TempDir(), "auth")
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		u, pw, err := readAuthFile(p)
		if err != nil || [2]string{u, pw} != want {
			t.Errorf("readAuthFile(%q) = %q, %q, %v", content, u, pw, err)
		}
	}
}

func TestBaseURL(t *testing.T) {
	c, err := Parse([]byte(minimal + "base_url: https://emlauncher.example.com/\n"))
	if err != nil || c.BaseURL != "https://emlauncher.example.com" {
		t.Fatalf("base_url = %q, %v", c.BaseURL, err)
	}
	for _, bad := range []string{"emlauncher.example.com", "ftp://x", "https://x/path", "https://x?q=1", "http://x\nenable_https: true"} {
		if _, err := Parse([]byte(minimal + "base_url: " + bad + "\n")); err == nil {
			t.Errorf("base_url %q accepted", bad)
		}
	}
}
