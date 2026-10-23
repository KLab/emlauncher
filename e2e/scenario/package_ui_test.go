package scenario

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/KLabTools/emlauncher/e2e/internal/fixture"
	"github.com/KLabTools/emlauncher/e2e/internal/harness"
)

func TestPackageEditUI(t *testing.T) {
	e := harness.New(t)
	owner := e.NewUser("owner")
	other := e.NewUser("other")
	app := e.CreateApp(owner, harness.Unique("app"))
	id := itoa(e.MustAPIUpload(app.APIKey, "before", "a.apk", fixture.APK("com.example.a"), form("tags", "old")))

	page := other.Get("/package?id=" + id)
	harness.ExpectStatus(t, page, http.StatusOK)
	expectContains(t, "package page", page.String(), "before")
	expectErrorPage(t, e, other.Get("/package?id=99999999"), http.StatusNotFound, "Not Found")

	expectErrorPage(t, e, other.Get("/package/edit?id="+id), http.StatusForbidden, "Permission Denied")
	harness.ExpectStatus(t, owner.Get("/package/edit?id="+id), http.StatusOK)

	res := owner.PostForm("/package/edit_commit?id="+id, form(
		"title", "after", "description", "edited", "tags[]", "new1", "tags[]", "new2", "tags[]", "", "protect", "1"))
	owner.ExpectRedirect(res, "/package?id="+id)
	pkg := e.DB.Row("SELECT * FROM package WHERE id = ?", id)
	expectEq(t, "title", pkg.Str("title"), "after")
	expectEq(t, "description", pkg.Str("description"), "edited")
	expectEq(t, "protect", pkg.Str("protect"), "1")
	expectEq(t, "tags", strings.Join(sortedTagNames(e, pkg.Int("id")), ","), "new1,new2")

	// protect のチェックを外す
	owner.PostForm("/package/edit_commit?id="+id, form("title", "after"))
	expectEq(t, "protect", e.DB.Row("SELECT protect FROM package WHERE id = ?", id).Str("protect"), "0")
	expectEq(t, "tags", len(sortedTagNames(e, pkg.Int("id"))), 0)

	harness.ExpectStatus(t, owner.PostForm("/package/edit_commit?id="+id, form("title", "")), http.StatusBadRequest)
	expectErrorPage(t, e, other.PostForm("/package/edit_commit?id="+id, form("title", "hijack")), http.StatusForbidden, "Permission Denied")
	expectEq(t, "title", e.DB.Row("SELECT title FROM package WHERE id = ?", id).Str("title"), "after")
}

func TestPackageDeleteUI(t *testing.T) {
	e := harness.New(t)
	owner := e.NewUser("owner")
	other := e.NewUser("other")
	app := e.CreateApp(owner, harness.Unique("app"))
	p1 := e.MustAPIUpload(app.APIKey, "p1", "a.apk", fixture.APK("com.example.a"), nil)
	time.Sleep(1100 * time.Millisecond)
	p2 := e.MustAPIUpload(app.APIKey, "p2", "b.aab", fixture.AAB("com.example.b"), form("tags", "t"))
	pkg1, pkg2 := e.Package(p1), e.Package(p2)
	att := e.DB.Row("SELECT * FROM attached_file WHERE package_id = ?", p2)

	expectErrorPage(t, e, other.Get("/package/delete_confirm?id="+itoa(p2)), http.StatusForbidden, "Permission Denied")

	res := owner.Get("/package/delete_confirm?id=" + itoa(p2))
	harness.ExpectStatus(t, res, http.StatusOK)
	token := harness.HiddenToken(t, res)

	expectErrorPage(t, e, owner.PostForm("/package/delete", form("id", itoa(p2), "token", "wrong")), http.StatusBadRequest, "Bad Request")
	expectEq(t, "package rows", e.DB.Count("SELECT count(*) FROM package WHERE id = ?", p2), 1)

	// 間違ったトークンでもトークンは消えないので, もう一度確認画面から
	res = owner.Get("/package/delete_confirm?id=" + itoa(p2))
	token = harness.HiddenToken(t, res)
	res = owner.PostForm("/package/delete", form("id", itoa(p2), "token", token))
	owner.ExpectRedirect(res, "/app?id="+itoa(app.ID))

	expectEq(t, "package rows", e.DB.Count("SELECT count(*) FROM package WHERE id = ?", p2), 0)
	expectEq(t, "package_tag rows", e.DB.Count("SELECT count(*) FROM package_tag WHERE package_id = ?", p2), 0)
	expectEq(t, "attached rows", e.DB.Count("SELECT count(*) FROM attached_file WHERE package_id = ?", p2), 0)
	checkNotStored(t, e, harness.PackageKey(pkg2))
	checkNotStored(t, e, harness.AttachedKey(att))
	checkStored(t, e, harness.PackageKey(pkg1), nil, "", false)
	expectEq(t, "last_upload rolled back", e.Application(app.ID).Str("last_upload"), pkg1.Str("created"))

	// トークンは使い捨て
	expectErrorPage(t, e, owner.PostForm("/package/delete", form("id", itoa(p1), "token", token)), http.StatusBadRequest, "Bad Request")
}

func TestPackageAttachUI(t *testing.T) {
	e := harness.New(t)
	owner := e.NewUser("owner")
	other := e.NewUser("other")
	app := e.CreateApp(owner, harness.Unique("app"))
	id := itoa(e.MustAPIUpload(app.APIKey, "p", "a.ipa", fixture.IPA("A", "com.example.a"), nil))
	id2 := itoa(e.MustAPIUpload(app.APIKey, "p2", "b.ipa", fixture.IPA("B", "com.example.b"), nil))

	data := []byte("symbols")
	expectErrorPage(t, e, other.PostMultipart("/package/attach", form("id", id),
		harness.File{Field: "file", Name: "sym.zip", Data: data}), http.StatusForbidden, "Permission Denied")

	res := owner.PostMultipart("/package/attach", form("id", id),
		harness.File{Field: "file", Name: "sym.zip", Data: data, ContentType: "application/zip"})
	owner.ExpectRedirect(res, "/package?id="+id)
	att := e.DB.Row("SELECT * FROM attached_file WHERE package_id = ?", id)
	expectEq(t, "original_file_name", att.Str("original_file_name"), "sym.zip")
	expectEq(t, "file_size", att.Int("file_size"), int64(len(data)))
	expectEq(t, "file_type", att.Str("file_type"), "")
	checkStored(t, e, harness.AttachedKey(att), data, mimeStored, false)
	expectContains(t, "package page", owner.Get("/package?id="+id).String(), "sym.zip")

	// オーナー以外もダウンロードはできる
	res = other.Get("/package/attach_download?id=" + id + "&attached_id=" + att.Str("id"))
	harness.ExpectStatus(t, res, http.StatusFound)
	expectDownload(t, e, res.Location(), harness.AttachedKey(att), data, "sym.zip")
	expectErrorPage(t, e, other.Get("/package/attach_download?id="+id2+"&attached_id="+att.Str("id")), http.StatusNotFound, "Not Found")

	expectErrorPage(t, e, other.Get("/package/attach_delete_confirm?id="+id+"&attached_id="+att.Str("id")), http.StatusForbidden, "Permission Denied")
	res = owner.Get("/package/attach_delete_confirm?id=" + id + "&attached_id=" + att.Str("id"))
	harness.ExpectStatus(t, res, http.StatusOK)
	token := harness.HiddenToken(t, res)
	res = owner.PostForm("/package/attach_delete", form("id", id, "attached_id", att.Str("id"), "token", token))
	owner.ExpectRedirect(res, "/package?id="+id)
	expectEq(t, "attached rows", e.DB.Count("SELECT count(*) FROM attached_file WHERE id = ?", att.Str("id")), 0)
	checkNotStored(t, e, harness.AttachedKey(att))
}
