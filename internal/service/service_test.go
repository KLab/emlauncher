package service

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"image"
	"image/png"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/KLab/emlauncher-go/internal/cache"
	"github.com/KLab/emlauncher-go/internal/config"
	"github.com/KLab/emlauncher-go/internal/db"
	"github.com/KLab/emlauncher-go/internal/mailer"
	"github.com/KLab/emlauncher-go/internal/randstr"
	"github.com/KLab/emlauncher-go/internal/storage"

	"github.com/go-sql-driver/mysql"
)

// DB を使うテストは, E2E テストの環境 (e2e/env/up.sh) を起動して EML_TEST_DB_HOST を指定したときだけ実行する.
//
//	EML_TEST_DB_HOST=127.0.0.1 [EML_TEST_DB_PORT=13306] [EML_TEST_MEMCACHED=127.0.0.1:21211] go test ./...

type fakeMailer struct {
	mu   sync.Mutex
	sent []mailer.Message
}

func (m *fakeMailer) Send(_ context.Context, msg mailer.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, msg)
	return nil
}

type testEnv struct {
	*Service
	mail *fakeMailer
	dir  string
}

func newTestService(t *testing.T) *testEnv {
	t.Helper()
	host := os.Getenv("EML_TEST_DB_HOST")
	if host == "" {
		t.Skip("set EML_TEST_DB_HOST to run")
	}
	port := os.Getenv("EML_TEST_DB_PORT")
	if port == "" {
		port = "13306"
	}
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}
	mc := mysql.NewConfig()
	mc.Net, mc.Addr = "tcp", host+":"+port
	mc.User, mc.Passwd, mc.DBName = "emlauncher", "emlauncher", "emlauncher"
	mc.ParseTime, mc.Loc = true, loc
	conn, err := mysql.NewConnector(mc)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB := sql.OpenDB(conn)
	t.Cleanup(func() { sqlDB.Close() })
	if err := sqlDB.Ping(); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	var c *cache.Cache
	if addr := os.Getenv("EML_TEST_MEMCACHED"); addr != "" {
		c = cache.New(addr, "emlauncher_unittest_"+randstr.New(8)+"_")
	}
	m := &fakeMailer{}
	cfg := &config.Config{Location: loc, MailSender: "EMLauncher <no-reply@example.com>"}
	return &testEnv{
		Service: New(Deps{
			DB: sqlDB, Storage: storage.NewLocal(dir, "/files"), Cache: c, Mailer: m, Config: cfg,
			Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		}),
		mail: m,
		dir:  dir,
	}
}

func pngData(t *testing.T) []byte {
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 10, 10))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func (e *testEnv) createApp(t *testing.T, owner string) db.Application {
	t.Helper()
	ctx := context.Background()
	id, err := e.CreateApp(ctx, owner, AppInfo{Title: "app-" + randstr.New(8), Icon: pngData(t)})
	if err != nil {
		t.Fatal(err)
	}
	app, err := e.GetApp(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	return app
}

// upload は中身が name のテキストのファイル (platform unknown) をアップロードする.
func (e *testEnv) upload(t *testing.T, appID int64, title string, tags ...string) PackageDetail {
	t.Helper()
	path := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(path, []byte(title), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := e.Upload(context.Background(), appID, LocalFile{Path: path, Name: title + ".txt", Size: int64(len(title))},
		UploadOptions{}, NewPackage{Title: title, Tags: tags})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func (e *testEnv) exists(key string) bool {
	_, err := os.Stat(filepath.Join(e.dir, filepath.FromSlash(key)))
	return err == nil
}

func TestPackageLifecycle(t *testing.T) {
	e := newTestService(t)
	ctx := context.Background()
	owner := randstr.New(8) + "@example.com"
	app := e.createApp(t, owner)
	if !e.exists(app.IconKey.String) {
		t.Fatalf("icon %s is not stored", app.IconKey.String)
	}

	p1 := e.upload(t, app.ID, "p1", "a", "b", "a", "")
	time.Sleep(1100 * time.Millisecond) // created を別の秒にする
	p2 := e.upload(t, app.ID, "p2", "b", "c")
	if !e.exists(PackageKey(p1.Package)) || !e.exists(PackageKey(p2.Package)) {
		t.Fatal("package files are not stored")
	}
	if p1.Platform != "unknown" || p1.Identifier.Valid || p1.Description.Valid {
		t.Errorf("p1 = %+v", p1.Package)
	}
	tags, err := e.q.ListTagsByApp(ctx, app.ID)
	if err != nil || len(tags) != 3 {
		t.Fatalf("tags = %v, %v (want a, b, c)", tags, err)
	}

	ids, ok, err := e.TagIDsByName(ctx, app.ID, []string{"b"})
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	list, err := e.ListPackages(ctx, PackageQuery{AppID: app.ID, TagIDs: ids, Limit: 10})
	if err != nil || len(list) != 2 || list[0].ID != p2.ID {
		t.Fatalf("list by tag b = %v, %v", list, err)
	}
	if _, ok, _ := e.TagIDsByName(ctx, app.ID, []string{"nosuch"}); ok {
		t.Fatal("unknown tag found")
	}

	title, desc, protect := "edited", "", true
	newTags := []string{"c", "d"}
	if _, err := e.EditPackage(ctx, p1.ID, PackageEdit{Title: &title, Description: &desc, Protect: &protect, Tags: &newTags}); err != nil {
		t.Fatal(err)
	}
	ds, err := e.Details(ctx, mustGet(t, e, p1.ID))
	if err != nil {
		t.Fatal(err)
	}
	if d := ds[0]; d.Title != "edited" || !d.Description.Valid || d.Protect != 1 || len(d.Tags) != 2 {
		t.Errorf("edited = %+v", d)
	}

	// 最新を消すと last_upload が 1 つ前に戻る
	if _, err := e.DeletePackage(ctx, p2.ID); err != nil {
		t.Fatal(err)
	}
	if e.exists(PackageKey(p2.Package)) {
		t.Fatal("deleted package file remains")
	}
	a, _ := e.GetApp(ctx, app.ID)
	if !a.LastUpload.Time.Equal(p1.Created) {
		t.Errorf("last_upload = %v, want %v", a.LastUpload, p1.Created)
	}
	if _, err := e.DeletePackage(ctx, p1.ID); err != nil {
		t.Fatal(err)
	}
	a, _ = e.GetApp(ctx, app.ID)
	if a.LastUpload.Valid || !a.DateToSort.Equal(a.Created) {
		t.Errorf("after deleting all: last_upload = %v, date_to_sort = %v", a.LastUpload, a.DateToSort)
	}
	if _, err := e.DeletePackage(ctx, p1.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete twice: %v", err)
	}
}

func mustGet(t *testing.T, e *testEnv, id int64) db.Package {
	p, err := e.GetPackage(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCommitUploadValidatesTempName(t *testing.T) {
	e := newTestService(t)
	app := e.createApp(t, randstr.New(8)+"@example.com")
	for _, name := range []string{"../package/1/1_x.apk", "short.apk", randstr.New(16) + ".apk"} {
		_, err := e.CommitUpload(context.Background(), app.ID, StagedUpload{TempName: name, FileName: "a.apk", Platform: "Android"}, NewPackage{Title: "t"})
		var ie *InvalidError
		if !errors.As(err, &ie) {
			t.Errorf("temp_name %q: err = %v, want InvalidError", name, err)
		}
	}
}

func TestCommentsAndNotification(t *testing.T) {
	e := newTestService(t)
	ctx := context.Background()
	owner := randstr.New(8) + "@example.com"
	app := e.createApp(t, owner)
	if err := e.SetOwners(ctx, app.ID, owner, []string{"co-" + owner, "", owner}); err != nil {
		t.Fatal(err)
	}
	p := e.upload(t, app.ID, "p")
	c1, err := e.PostComment(ctx, app.ID, "user@example.com", p.ID, "hello", "http://example.com")
	if err != nil {
		t.Fatal(err)
	}
	c2, err := e.PostComment(ctx, app.ID, "user@example.com", 0, "again", "http://example.com")
	if err != nil {
		t.Fatal(err)
	}
	if c1.Number != 1 || c2.Number != 2 || c2.PackageID.Valid {
		t.Errorf("comments = %+v, %+v", c1, c2)
	}
	if len(e.mail.sent) != 2 || len(e.mail.sent[0].To) != 2 || !strings.Contains(e.mail.sent[0].Body, "Target Package: p") {
		t.Errorf("mails = %+v", e.mail.sent)
	}
	a, _ := e.GetApp(ctx, app.ID)
	if !a.DateToSort.Equal(c2.Created) {
		t.Errorf("date_to_sort = %v, want %v", a.DateToSort, c2.Created)
	}
}

func TestInstallToken(t *testing.T) {
	e := newTestService(t)
	if e.cache == nil {
		t.Skip("set EML_TEST_MEMCACHED to run")
	}
	ctx := context.Background()
	app := e.createApp(t, randstr.New(8)+"@example.com")
	p := e.upload(t, app.ID, "p")
	tok, err := e.CreateInstallToken(ctx, p.Package, "a@example.com", 31*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	got, err := e.InstallTokenByToken(ctx, tok.Token)
	if err != nil || got.PackageID != p.ID || got.Mail != "a@example.com" {
		t.Fatalf("token = %+v, %v", got, err)
	}
	// PHP 版が書いた形式 (package_id が数値) も読める
	phpToken := randstr.New(32)
	if err := e.cache.Set(installTokenPrefix+phpToken, []byte(`{"mail":"b@example.com","package_id":12,"expire":"2099-01-01 00:00:00"}`), time.Minute); err != nil {
		t.Fatal(err)
	}
	if got, err := e.InstallTokenByToken(ctx, phpToken); err != nil || got.PackageID != 12 {
		t.Fatalf("php token = %+v, %v", got, err)
	}
	// インストールトークンをパスワードリセットのキーとして使えない
	if _, err := e.PasswordResetMail(ctx, installTokenPrefix+tok.Token); !errors.Is(err, ErrNotFound) {
		t.Fatalf("install token accepted as a reset key: %v", err)
	}
	if err := e.ResetPassword(ctx, installTokenPrefix+tok.Token, "x"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ResetPassword with install token: %v", err)
	}
	// 形式の違うトークン (memcached のキーにできない値を含む) は見つからない扱い
	for _, bad := range []string{"nosuch", "a b", strings.Repeat("a", 300), "x\r\n" + randstr.New(28)} {
		if _, err := e.InstallTokenByToken(ctx, bad); !errors.Is(err, ErrNotFound) {
			t.Fatalf("token %q: %v", bad, err)
		}
		if ok, err := e.CheckPlistToken(ctx, bad, 1); ok || err != nil {
			t.Fatalf("plist token %q: %v %v", bad, ok, err)
		}
	}
	if _, err := e.InstallTokenByToken(ctx, "nosuch"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown token: %v", err)
	}
}
