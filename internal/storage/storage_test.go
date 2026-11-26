package storage

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/KLab/emlauncher-go/internal/randstr"
)

func TestRFC5987(t *testing.T) {
	for in, want := range map[string]string{
		"my app.apk":  "my%20app.apk",
		"a+b~c_d-e.f": "a%2Bb~c_d-e.f",
		"日本.ipa":      "%E6%97%A5%E6%9C%AC.ipa",
	} {
		if got := rfc5987(in); got != want {
			t.Errorf("rfc5987(%q) = %q, want %q", in, got, want)
		}
	}
}

// testStorage は Local と S3 の共通の振る舞いを確かめる.
// download はダウンロード URL から中身を取得する.
func testStorage(t *testing.T, s Storage, download func(t *testing.T, url string) []byte) {
	ctx := context.Background()
	prefix := "storage-test/" + randstr.New(8) + "/"

	src := filepath.Join(t.TempDir(), "src")
	data := []byte("package data " + randstr.New(16))
	if err := os.WriteFile(src, data, 0o644); err != nil {
		t.Fatal(err)
	}

	tmp := prefix + "temp-data/" + randstr.New(16) + ".apk"
	if err := s.PutFile(ctx, tmp, src); err != nil {
		t.Fatal(err)
	}
	if n, err := s.Size(ctx, tmp); err != nil || n != int64(len(data)) {
		t.Fatalf("Size = %d, %v", n, err)
	}

	key := prefix + "package/1/2_x.apk"
	if err := s.Move(ctx, tmp, key); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Size(ctx, tmp); !errors.Is(err, ErrNotFound) {
		t.Fatalf("source remains after Move: %v", err)
	}
	u, err := s.DownloadURL(ctx, key, time.Hour, "my app.apk")
	if err != nil {
		t.Fatal(err)
	}
	if got := download(t, u); !bytes.Equal(got, data) {
		t.Fatalf("downloaded %q, want %q", got, data)
	}

	icon := prefix + "app-icons/1/" + randstr.New(16) + ".png"
	if err := s.PutIcon(ctx, icon, []byte("png")); err != nil {
		t.Fatal(err)
	}
	if got := download(t, s.PublicURL(icon)); string(got) != "png" {
		t.Fatalf("icon = %q", got)
	}

	for _, k := range []string{key, icon} {
		if err := s.Delete(ctx, k); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Size(ctx, k); !errors.Is(err, ErrNotFound) {
			t.Fatalf("%s remains after Delete: %v", k, err)
		}
	}
	if err := s.Delete(ctx, key); err != nil {
		t.Fatalf("Delete of missing key: %v", err)
	}
	if err := s.Move(ctx, tmp, key); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Move of missing key: %v", err)
	}
}

func TestLocal(t *testing.T) {
	dir := t.TempDir()
	l := NewLocal(dir+"/", "/files/")
	srv := httptest.NewServer(l.Handler())
	defer srv.Close()
	testStorage(t, l, func(t *testing.T, u string) []byte {
		if !strings.HasPrefix(u, "/files/") {
			t.Fatalf("url = %q", u)
		}
		return get(t, srv.URL+u)
	})

	for _, bad := range []string{"../x", "a/../../x", ""} {
		if err := l.PutIcon(context.Background(), bad, nil); err == nil {
			t.Errorf("PutIcon(%q) accepted", bad)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "secret"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"/files/../secret", "/files/d", "/files/d/", "/other/secret", "/files//secret"} {
		res, err := http.Get(srv.URL + p)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if p == "/files//secret" {
			continue
		}
		if res.StatusCode != http.StatusNotFound {
			t.Errorf("GET %s: status %d", p, res.StatusCode)
		}
	}
}

// S3 のテストは S3 互換ストレージ (E2E テストの LocalStack など) を指定したときだけ実行する.
//
//	EML_TEST_S3_ENDPOINT=http://localhost:14566 EML_TEST_S3_BUCKET=emlauncher-e2e
func newTestS3(t *testing.T) *S3 {
	endpoint := os.Getenv("EML_TEST_S3_ENDPOINT")
	if endpoint == "" {
		t.Skip("set EML_TEST_S3_ENDPOINT to run")
	}
	bucket := os.Getenv("EML_TEST_S3_BUCKET")
	if bucket == "" {
		bucket = "emlauncher-e2e"
	}
	s, err := NewS3(context.Background(), S3Config{
		Key: "test", Secret: "test", Region: "ap-northeast-1", Bucket: bucket,
		BaseURL: endpoint, ExternalURL: endpoint,
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestS3(t *testing.T) {
	s := newTestS3(t)
	testStorage(t, s, func(t *testing.T, u string) []byte {
		if strings.Contains(u, "X-Amz-Signature") && !strings.Contains(u, "X-Amz-Expires=3600") {
			t.Errorf("url = %s", u)
		}
		return get(t, u)
	})
}

// 5GB を超えるものと同じ手順 (UploadPartCopy) で移動できる.
func TestS3MultipartMove(t *testing.T) {
	s := newTestS3(t)
	s.maxCopySize = 1 << 20
	s.copyPartSize = 5 << 20
	ctx := context.Background()

	data := make([]byte, 11<<20+123)
	if _, err := io.ReadFull(rand.Reader, data); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(t.TempDir(), "big")
	if err := os.WriteFile(src, data, 0o644); err != nil {
		t.Fatal(err)
	}
	tmp := "storage-test/" + randstr.New(16) + ".bin"
	dst := "storage-test/" + randstr.New(16) + ".bin"
	if err := s.PutFile(ctx, tmp, src); err != nil {
		t.Fatal(err)
	}
	if err := s.Move(ctx, tmp, dst); err != nil {
		t.Fatal(err)
	}
	defer s.Delete(ctx, dst)
	u, err := s.DownloadURL(ctx, dst, time.Minute, "big.bin")
	if err != nil {
		t.Fatal(err)
	}
	if got := get(t, u); !bytes.Equal(got, data) {
		t.Fatalf("content mismatch (%d bytes)", len(got))
	}
}

func get(t *testing.T, u string) []byte {
	t.Helper()
	res, err := http.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: status %d: %s", u, res.StatusCode, b)
	}
	return b
}
