package storage

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func TestContentDisposition(t *testing.T) {
	got := ContentDisposition("my app (1).apk")
	want := "attachment; filename*=UTF-8''my%20app%20%281%29.apk"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := ContentDisposition("日本語.ipa"); got != "attachment; filename*=UTF-8''%E6%97%A5%E6%9C%AC%E8%AA%9E.ipa" {
		t.Errorf("got %q", got)
	}
}

// PHP 版の LocalFile と同じパス・URL になること.
func TestLocalLayout(t *testing.T) {
	l := NewLocal(LocalConfig{Path: "/var/www/emlauncher/", URLPrefix: "/files/"})
	if got := l.Path("package/1/2_abc.apk"); got != filepath.FromSlash("/var/www/emlauncher/package/1/2_abc.apk") {
		t.Errorf("Path = %q", got)
	}
	if got := l.Path("/app-icons/1/x.png/"); got != filepath.FromSlash("/var/www/emlauncher/app-icons/1/x.png") {
		t.Errorf("Path = %q", got)
	}
	if got := l.Path("../../etc/passwd"); got != filepath.FromSlash("/var/www/emlauncher/etc/passwd") {
		t.Errorf("Path escapes the directory: %q", got)
	}
	if got := l.PublicURL("package/1/2_abc.apk"); got != "/files/package/1/2_abc.apk" {
		t.Errorf("PublicURL = %q", got)
	}
}

func TestLocal(t *testing.T) {
	dir := t.TempDir()
	l := NewLocal(LocalConfig{Path: dir, URLPrefix: "/files"})
	testStorage(t, l, func(key string) ([]byte, bool) {
		b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(key)))
		if errors.Is(err, os.ErrNotExist) {
			return nil, false
		}
		if err != nil {
			t.Fatal(err)
		}
		return b, true
	}, func(rawURL string) *http.Response {
		rec := httptest.NewRecorder()
		l.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, rawURL, nil))
		return rec.Result()
	}, nil)

	// ディレクトリの一覧は返さない
	rec := httptest.NewRecorder()
	l.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/files/package/", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("directory listing: status %d", rec.Code)
	}
}

// E2E 環境の LocalStack (e2e/env/up.sh) を使う.
func TestS3(t *testing.T) {
	endpoint := os.Getenv("EML_TEST_S3_ENDPOINT")
	if endpoint == "" {
		t.Skip("EML_TEST_S3_ENDPOINT is not set")
	}
	ctx := context.Background()
	s, err := NewS3(ctx, S3Config{
		Key: "test", Secret: "test", Region: "ap-northeast-1", BucketName: "emlauncher-e2e",
		BaseURL: endpoint,
	})
	if err != nil {
		t.Fatal(err)
	}
	get := func(key string) ([]byte, bool) {
		out, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: &s.bucket, Key: &key})
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return nil, false
		}
		if err != nil {
			t.Fatal(err)
		}
		defer out.Body.Close()
		b, _ := io.ReadAll(out.Body)
		return b, true
	}
	meta := func(key string) (string, bool) {
		head, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: &s.bucket, Key: &key})
		if err != nil {
			t.Fatal(err)
		}
		acl, err := s.client.GetObjectAcl(ctx, &s3.GetObjectAclInput{Bucket: &s.bucket, Key: &key})
		if err != nil {
			t.Fatal(err)
		}
		public := false
		for _, g := range acl.Grants {
			if g.Grantee != nil && aws.ToString(g.Grantee.URI) == "http://acs.amazonaws.com/groups/global/AllUsers" && g.Permission == types.PermissionRead {
				public = true
			}
		}
		return aws.ToString(head.ContentType), public
	}
	testStorage(t, s, get, func(rawURL string) *http.Response {
		res, err := http.Get(rawURL)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}, func(icon, file string) {
		// ACL と ContentType は PHP 版と同じ
		if ct, pub := meta(icon); ct != "image/png" || !pub {
			t.Errorf("icon: ContentType=%q public=%v", ct, pub)
		}
		if ct, pub := meta(file); ct != FileContentType || pub {
			t.Errorf("file: ContentType=%q public=%v", ct, pub)
		}
		if !strings.HasPrefix(s.PublicURL(icon), endpoint+"/emlauncher-e2e/") {
			t.Errorf("PublicURL = %q", s.PublicURL(icon))
		}
	})
}

// 大きなファイルの Rename はマルチパートコピーになる (CopyObject は 5GB まで).
func TestS3RenameMultipart(t *testing.T) {
	endpoint := os.Getenv("EML_TEST_S3_ENDPOINT")
	if endpoint == "" {
		t.Skip("EML_TEST_S3_ENDPOINT is not set")
	}
	ctx := context.Background()
	s, err := NewS3(ctx, S3Config{
		Key: "test", Secret: "test", Region: "ap-northeast-1", BucketName: "emlauncher-e2e",
		BaseURL: endpoint,
	})
	if err != nil {
		t.Fatal(err)
	}
	// 5MB のパート 2 つと端数 1 つ (パートの最小は 5MB)
	s.copyThreshold = 1
	s.copyPartSize = 5 << 20
	data := make([]byte, 11<<20)
	for i := range data {
		data[i] = byte(i % 251)
	}
	src := filepath.Join(t.TempDir(), "upload")
	os.WriteFile(src, data, 0o600)
	prefix := "go-test/" + randomName() + "/"
	temp := prefix + "temp-data/" + randomName() + ".apk"
	file := prefix + "package/1/2_" + randomName() + ".apk"
	if err := s.SaveFile(ctx, temp, src); err != nil {
		t.Fatal(err)
	}
	if err := s.Rename(ctx, temp, file); err != nil {
		t.Fatal(err)
	}
	defer s.Delete(ctx, file)
	if _, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: &s.bucket, Key: &temp}); err == nil {
		t.Error("temp file remains after rename")
	}
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: &s.bucket, Key: &file})
	if err != nil {
		t.Fatal(err)
	}
	defer out.Body.Close()
	got, _ := io.ReadAll(out.Body)
	if string(got) != string(data) {
		t.Errorf("copied data differs: %d bytes, want %d", len(got), len(data))
	}
	if ct := aws.ToString(out.ContentType); ct != FileContentType {
		t.Errorf("ContentType = %q", ct)
	}
}

// testStorage は Storage の実装に共通の動作を確認する. check は削除前に呼ばれる.
func testStorage(t *testing.T, s Storage, get func(key string) ([]byte, bool), fetch func(rawURL string) *http.Response, check func(icon, file string)) {
	t.Helper()
	ctx := context.Background()
	prefix := "go-test/" + randomName() + "/"

	icon := prefix + "app-icons/1/" + randomName() + ".png"
	if err := s.SaveIcon(ctx, icon, []byte("png data")); err != nil {
		t.Fatal(err)
	}
	if b, ok := get(icon); !ok || string(b) != "png data" {
		t.Fatalf("icon: %q %v", b, ok)
	}
	res := fetch(s.PublicURL(icon))
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK || string(body) != "png data" {
		t.Errorf("fetch icon: %d %q", res.StatusCode, body)
	}

	src := filepath.Join(t.TempDir(), "upload")
	os.WriteFile(src, []byte("package data"), 0o600)
	temp := prefix + "temp-data/" + randomName() + ".apk"
	if err := s.SaveFile(ctx, temp, src); err != nil {
		t.Fatal(err)
	}
	file := prefix + "package/1/2_" + randomName() + ".apk"
	if err := s.Rename(ctx, temp, file); err != nil {
		t.Fatal(err)
	}
	if _, ok := get(temp); ok {
		t.Error("temp file remains after rename")
	}
	if b, ok := get(file); !ok || string(b) != "package data" {
		t.Fatalf("file: %q %v", b, ok)
	}

	u, err := s.DownloadURL(ctx, file, time.Hour, "my app.apk")
	if err != nil {
		t.Fatal(err)
	}
	res = fetch(u)
	body, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK || string(body) != "package data" {
		t.Errorf("download: %d %q", res.StatusCode, body)
	}
	if _, isS3 := s.(*S3); isS3 {
		if cd := res.Header.Get("Content-Disposition"); cd != "attachment; filename*=UTF-8''my%20app.apk" {
			t.Errorf("Content-Disposition = %q", cd)
		}
		if !strings.Contains(u, "X-Amz-Expires=3600") {
			t.Errorf("download url has no expiry: %s", u)
		}
	}

	if check != nil {
		check(icon, file)
	}
	if err := s.Delete(ctx, file); err != nil {
		t.Fatal(err)
	}
	if _, ok := get(file); ok {
		t.Error("file remains after delete")
	}
	if err := s.Delete(ctx, file); err != nil {
		t.Errorf("deleting a missing key: %v", err)
	}
}
