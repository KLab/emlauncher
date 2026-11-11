package scenario

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	e2e "emlauncher-e2e"
)

func TestAPIUploadIPA(t *testing.T) {
	e := e2e.New(t)
	owner := e.NewUser("owner")
	app := e.CreateApp(owner, e2e.Unique("app"))

	ipa := e2e.IPA("Sample", "com.example.e2e.ios")
	res := e.APIUpload(app.APIKey, form(
		"title", "ipa build 1",
		"description", "説明 multi\nline",
		"tags", "dev,ios,",
		"protect", "true",
	), e2e.File{Field: "file", Name: "sample.ipa", Data: ipa})
	e2e.ExpectStatus(t, res, http.StatusOK)
	m := res.JSONObject(t)

	pkgID := e2e.QueryInt(t, jsonStr(t, m, "package_url"), "id")
	pkg := e.Package(pkgID)
	checkPackageJSON(t, e, m, pkg)

	expectEq(t, "app_id", pkg.Int("app_id"), app.ID)
	expectEq(t, "platform", pkg.Str("platform"), "iOS")
	expectMatch(t, "file_name", regexp.MustCompile(`^[0-9A-Za-z]{16}\.ipa$`), pkg.Str("file_name"))
	expectEq(t, "title", pkg.Str("title"), "ipa build 1")
	expectEq(t, "description", pkg.Str("description"), "説明 multi\nline")
	expectEq(t, "identifier", pkg.Str("identifier"), "com.example.e2e.ios")
	expectEq(t, "original_file_name", pkg.Str("original_file_name"), "sample.ipa")
	expectEq(t, "file_size", pkg.Int("file_size"), int64(len(ipa)))
	expectEq(t, "protect", pkg.Str("protect"), "1")
	e.ExpectRecent("package.created", pkg.Str("created"))
	expectEq(t, "tags", strings.Join(sortedTagNames(e, pkgID), ","), "dev,ios")
	expectEq(t, "install_count", jsonNum(t, m, "install_count"), "0")

	checkStored(t, e, e2e.PackageKey(pkg), ipa, mimeStored, false)

	a := e.Application(app.ID)
	expectEq(t, "last_upload", a.Str("last_upload"), pkg.Str("created"))
	expectEq(t, "date_to_sort", a.Str("date_to_sort"), pkg.Str("created"))
}

func TestAPIUploadAPK(t *testing.T) {
	e := e2e.New(t)
	owner := e.NewUser("owner")
	app := e.CreateApp(owner, e2e.Unique("app"))

	apk := e2e.APK("com.example.e2e.android")
	res := e.APIUpload(app.APIKey, form("title", "apk build"),
		e2e.File{Field: "file", Name: "sample.apk", Data: apk})
	e2e.ExpectStatus(t, res, http.StatusOK)
	m := res.JSONObject(t)
	pkg := e.Package(e2e.QueryInt(t, jsonStr(t, m, "package_url"), "id"))
	checkPackageJSON(t, e, m, pkg)

	expectEq(t, "platform", pkg.Str("platform"), "Android")
	expectEq(t, "identifier", pkg.Str("identifier"), "com.example.e2e.android")
	expectEq(t, "protect", pkg.Str("protect"), "0")
	expectEq(t, "description null", pkg.Null("description"), true)
	expectEq(t, "tags", len(sortedTagNames(e, pkg.Int("id"))), 0)
	checkStored(t, e, e2e.PackageKey(pkg), apk, mimeStored, false)
}

func TestAPIUploadAABAndDSYM(t *testing.T) {
	e := e2e.New(t)
	owner := e.NewUser("owner")
	app := e.CreateApp(owner, e2e.Unique("app"))

	aab := e2e.AAB("com.example.e2e.bundle")
	dsym := []byte("dummy dSYM archive")
	res := e.APIUpload(app.APIKey, form("title", "aab build"),
		e2e.File{Field: "file", Name: "sample.aab", Data: aab},
		e2e.File{Field: "dsym", Name: "sample.dSYM.zip", Data: dsym, ContentType: "application/zip"})
	e2e.ExpectStatus(t, res, http.StatusOK)
	m := res.JSONObject(t)
	pkg := e.Package(e2e.QueryInt(t, jsonStr(t, m, "package_url"), "id"))
	checkPackageJSON(t, e, m, pkg)

	expectEq(t, "platform", pkg.Str("platform"), "Android")
	expectEq(t, "identifier", pkg.Str("identifier"), "com.example.e2e.bundle")
	expectMatch(t, "file_name", regexp.MustCompile(`^[0-9A-Za-z]{16}\.aab$`), pkg.Str("file_name"))
	checkStored(t, e, e2e.PackageKey(pkg), aab, mimeStored, false)

	afs := e.DB.Rows("SELECT * FROM attached_file WHERE package_id = ? ORDER BY id", pkg.Str("id"))
	expectEq(t, "attached files", len(afs), 2)

	// AAB から生成した universal APK
	apk := afs[0]
	expectEq(t, "apk.app_id", apk.Int("app_id"), app.ID)
	expectEq(t, "apk.original_file_name", apk.Str("original_file_name"), "sample.apk")
	expectEq(t, "apk.file_type", apk.Str("file_type"), "apk")
	expectMatch(t, "apk.file_name", regexp.MustCompile(`^[0-9A-Za-z]{16}\.apk$`), apk.Str("file_name"))
	e.ExpectRecent("attached.created", apk.Str("created"))
	checkStored(t, e, e2e.AttachedKey(apk), nil, mimeStored, false)
	apkData := e.Storage.Read(e2e.AttachedKey(apk))
	expectEq(t, "apk.file_size", apk.Int("file_size"), int64(len(apkData)))
	if _, err := zip.NewReader(bytes.NewReader(apkData), int64(len(apkData))); err != nil {
		t.Fatalf("generated apk is not zip: %v", err)
	}

	// dSYM
	ds := afs[1]
	expectEq(t, "dsym.original_file_name", ds.Str("original_file_name"), "sample.dSYM.zip")
	expectEq(t, "dsym.file_type", ds.Str("file_type"), "dSYM")
	expectMatch(t, "dsym.file_name", regexp.MustCompile(`^[0-9A-Za-z]{16}\.zip$`), ds.Str("file_name"))
	expectEq(t, "dsym.file_size", ds.Int("file_size"), int64(len(dsym)))
	checkStored(t, e, e2e.AttachedKey(ds), dsym, mimeStored, false)
}

// AAB と一緒に APK を送ると, AAB から生成せずにその APK を添付する.
func TestAPIUploadAABWithAPK(t *testing.T) {
	e := e2e.New(t)
	if e.Config.Target == "php" {
		t.Skip("PHP 版 (4124ca6) には無い機能")
	}
	owner := e.NewUser("owner")
	app := e.CreateApp(owner, e2e.Unique("app"))

	aab := e2e.AAB("com.example.e2e.bundle")
	// 識別子は送った APK から読む
	apk := e2e.APK("com.example.e2e.prebuilt")
	res := e.APIUpload(app.APIKey, form("title", "aab with apk"),
		e2e.File{Field: "file", Name: "sample.aab", Data: aab},
		e2e.File{Field: "apk", Name: "prebuilt.apk", Data: apk})
	e2e.ExpectStatus(t, res, http.StatusOK)
	m := res.JSONObject(t)
	pkg := e.Package(e2e.QueryInt(t, jsonStr(t, m, "package_url"), "id"))
	checkPackageJSON(t, e, m, pkg)

	expectEq(t, "platform", pkg.Str("platform"), "Android")
	expectEq(t, "identifier", pkg.Str("identifier"), "com.example.e2e.prebuilt")
	checkStored(t, e, e2e.PackageKey(pkg), aab, mimeStored, false)

	afs := e.DB.Rows("SELECT * FROM attached_file WHERE package_id = ? ORDER BY id", pkg.Str("id"))
	expectEq(t, "attached files", len(afs), 1)
	af := afs[0]
	expectEq(t, "apk.original_file_name", af.Str("original_file_name"), "prebuilt.apk")
	expectEq(t, "apk.file_type", af.Str("file_type"), "apk")
	expectEq(t, "apk.file_size", af.Int("file_size"), int64(len(apk)))
	expectMatch(t, "apk.file_name", regexp.MustCompile(`^[0-9A-Za-z]{16}\.apk$`), af.Str("file_name"))
	checkStored(t, e, e2e.AttachedKey(af), apk, mimeStored, false)
}

func TestAPIUploadUnknownFile(t *testing.T) {
	e := e2e.New(t)
	owner := e.NewUser("owner")
	app := e.CreateApp(owner, e2e.Unique("app"))

	data := []byte("plain text, not a package")
	res := e.APIUpload(app.APIKey, form("title", "unknown"),
		e2e.File{Field: "file", Name: "readme.txt", Data: data, ContentType: "text/plain"})
	e2e.ExpectStatus(t, res, http.StatusOK)
	m := res.JSONObject(t)
	pkg := e.Package(e2e.QueryInt(t, jsonStr(t, m, "package_url"), "id"))
	checkPackageJSON(t, e, m, pkg)
	expectEq(t, "platform", pkg.Str("platform"), "unknown")
	expectEq(t, "identifier null", pkg.Null("identifier"), true)
	checkStored(t, e, e2e.PackageKey(pkg), data, mimeStored, false)
}

func TestAPIUploadReusesTags(t *testing.T) {
	e := e2e.New(t)
	owner := e.NewUser("owner")
	app := e.CreateApp(owner, e2e.Unique("app"))
	apk := e2e.APK("com.example.e2e.android")

	p1 := e.MustAPIUpload(app.APIKey, "p1", "a.apk", apk, form("tags", "x,y"))
	p2 := e.MustAPIUpload(app.APIKey, "p2", "a.apk", apk, form("tags", "y,z"))
	expectEq(t, "tags of p1", strings.Join(sortedTagNames(e, p1), ","), "x,y")
	expectEq(t, "tags of p2", strings.Join(sortedTagNames(e, p2), ","), "y,z")
	expectEq(t, "tag rows", e.DB.Count("SELECT count(*) FROM tag WHERE app_id = ?", app.ID), 3)
}

func TestAPIErrors(t *testing.T) {
	e := e2e.New(t)
	owner := e.NewUser("owner")
	app := e.CreateApp(owner, e2e.Unique("app"))
	c := e.NewClient()

	res := e.APIUpload("invalid-key", form("title", "x"), e2e.File{Field: "file", Name: "a.apk", Data: []byte("x")})
	e2e.ExpectStatus(t, res, http.StatusBadRequest)
	expectEq(t, "error", jsonStr(t, res.JSONObject(t), "error"), "Invalid api_key")

	res = c.Get("/api/upload?api_key=" + app.APIKey)
	e2e.ExpectStatus(t, res, http.StatusMethodNotAllowed)
	expectEq(t, "error", jsonStr(t, res.JSONObject(t), "error"), "Method Not Allowed")

	res = e.APIUpload(app.APIKey, url.Values{})
	e2e.ExpectStatus(t, res, http.StatusBadRequest)
	expectEq(t, "error", jsonStr(t, res.JSONObject(t), "error"), "A required field (file,title) is not present.")

	res = e.APIUpload(app.APIKey, url.Values{}, e2e.File{Field: "file", Name: "a.apk", Data: []byte("x")})
	e2e.ExpectStatus(t, res, http.StatusBadRequest)
	expectEq(t, "error", jsonStr(t, res.JSONObject(t), "error"), "A required field (title) is not present.")

	res = c.Get("/api/no_such_action?api_key=" + app.APIKey)
	e2e.ExpectStatus(t, res, http.StatusNotFound)

	expectEq(t, "packages", e.DB.Count("SELECT count(*) FROM package WHERE app_id = ?", app.ID), 0)
}

func TestAPIPackageList(t *testing.T) {
	e := e2e.New(t)
	owner := e.NewUser("owner")
	app := e.CreateApp(owner, e2e.Unique("app"))
	other := e.CreateApp(owner, e2e.Unique("other"))

	ipa := e.MustAPIUpload(app.APIKey, "ipa", "a.ipa", e2e.IPA("A", "com.example.a"), form("tags", "a"))
	apk1 := e.MustAPIUpload(app.APIKey, "apk1", "a.apk", e2e.APK("com.example.a"), form("tags", "a,b"))
	apk2 := e.MustAPIUpload(app.APIKey, "apk2", "b.apk", e2e.APK("com.example.a"), nil)
	e.MustAPIUpload(other.APIKey, "other", "o.apk", e2e.APK("com.example.o"), form("tags", "a"))

	list := func(q url.Values) []int64 {
		t.Helper()
		res := e.APICall("package_list", app.APIKey, q)
		e2e.ExpectStatus(t, res, http.StatusOK)
		arr, ok := res.JSON(t).([]any)
		if !ok {
			t.Fatalf("package_list is not array: %s", res.Body)
		}
		var ids []int64
		for _, v := range arr {
			m := v.(map[string]any)
			id := e2e.QueryInt(t, jsonStr(t, m, "package_url"), "id")
			checkPackageJSON(t, e, m, e.Package(id))
			ids = append(ids, id)
		}
		return ids
	}
	expectIDs := func(what string, got []int64, want ...int64) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("%s = %v, want %v", what, got, want)
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("%s = %v, want %v", what, got, want)
			}
		}
	}

	expectIDs("all", list(nil), apk2, apk1, ipa)
	expectIDs("limit", list(form("limit", "2")), apk2, apk1)
	expectIDs("offset", list(form("limit", "2", "offset", "2")), ipa)
	expectIDs("platform", list(form("platform", "Android")), apk2, apk1)
	expectIDs("tag a", list(form("tags", "a")), apk1, ipa)
	expectIDs("tag a,b", list(form("tags", "a,b")), apk1)
	expectIDs("tag a + iOS", list(form("tags", "a", "platform", "iOS")), ipa)
	expectIDs("unknown tag", list(form("tags", "nosuchtag")))
}

func TestAPIPackageEdit(t *testing.T) {
	e := e2e.New(t)
	owner := e.NewUser("owner")
	app := e.CreateApp(owner, e2e.Unique("app"))
	other := e.CreateApp(owner, e2e.Unique("other"))
	id := e.MustAPIUpload(app.APIKey, "before", "a.apk", e2e.APK("com.example.a"),
		form("description", "desc", "tags", "old", "protect", "1"))

	edit := func(q url.Values) *e2e.Response {
		q.Set("id", itoa(id))
		return e.APICall("package_edit", app.APIKey, q)
	}

	res := edit(form("title", "after"))
	e2e.ExpectStatus(t, res, http.StatusOK)
	checkPackageJSON(t, e, res.JSONObject(t), e.Package(id))
	pkg := e.Package(id)
	expectEq(t, "title", pkg.Str("title"), "after")
	expectEq(t, "description kept", pkg.Str("description"), "desc")
	expectEq(t, "protect kept", pkg.Str("protect"), "1")
	expectEq(t, "tags kept", strings.Join(sortedTagNames(e, id), ","), "old")

	res = edit(form("description", "new desc", "protect", "false", "tags", "n1,n2"))
	e2e.ExpectStatus(t, res, http.StatusOK)
	pkg = e.Package(id)
	expectEq(t, "title kept", pkg.Str("title"), "after")
	expectEq(t, "description", pkg.Str("description"), "new desc")
	expectEq(t, "protect", pkg.Str("protect"), "0")
	expectEq(t, "tags", strings.Join(sortedTagNames(e, id), ","), "n1,n2")

	res = edit(url.Values{})
	e2e.ExpectStatus(t, res, http.StatusBadRequest)
	expectEq(t, "error", jsonStr(t, res.JSONObject(t), "error"), "no parameter")

	res = edit(form("title", ""))
	e2e.ExpectStatus(t, res, http.StatusBadRequest)
	expectEq(t, "error", jsonStr(t, res.JSONObject(t), "error"), "title must not empty")

	res = e.APICall("package_edit", other.APIKey, form("id", itoa(id), "title", "hijack"))
	e2e.ExpectStatus(t, res, http.StatusForbidden)

	res = e.APICall("package_edit", app.APIKey, form("title", "x"))
	e2e.ExpectStatus(t, res, http.StatusBadRequest)
	expectEq(t, "error", jsonStr(t, res.JSONObject(t), "error"), "id required")

	res = e.APICall("package_edit", app.APIKey, form("id", "99999999", "title", "x"))
	e2e.ExpectStatus(t, res, http.StatusBadRequest)
	expectEq(t, "title unchanged", e.Package(id).Str("title"), "after")
}

func TestAPIAttachDetach(t *testing.T) {
	e := e2e.New(t)
	owner := e.NewUser("owner")
	app := e.CreateApp(owner, e2e.Unique("app"))
	id := e.MustAPIUpload(app.APIKey, "p", "a.apk", e2e.APK("com.example.a"), nil)

	data := []byte("mapping file contents")
	res := e.NewClient().PostMultipart("/api/package_attach", form("api_key", app.APIKey, "id", itoa(id)),
		e2e.File{Field: "file", Name: "mapping.txt", Data: data, ContentType: "text/plain"})
	e2e.ExpectStatus(t, res, http.StatusOK)
	m := res.JSONObject(t)
	af := e.DB.Row("SELECT * FROM attached_file WHERE id = ?", jsonNum(t, m, "id"))
	checkAttachedJSON(t, m, af)
	expectEq(t, "package_id", af.Int("package_id"), id)
	expectEq(t, "file_type", af.Str("file_type"), "")
	expectEq(t, "original_file_name", af.Str("original_file_name"), "mapping.txt")
	checkStored(t, e, e2e.AttachedKey(af), data, mimeStored, false)

	// package に含まれる
	res = e.APICall("package_list", app.APIKey, nil)
	pkgJSON := res.JSON(t).([]any)[0].(map[string]any)
	expectEq(t, "attached in list", len(jsonList(t, pkgJSON, "attached_files")), 1)

	res = e.APICall("package_detach", app.APIKey, form("id", itoa(id), "attached_file_id", "99999999"))
	e2e.ExpectStatus(t, res, http.StatusNotFound)

	res = e.APICall("package_detach", app.APIKey, form("id", itoa(id)))
	e2e.ExpectStatus(t, res, http.StatusBadRequest)

	res = e.APICall("package_detach", app.APIKey, form("id", itoa(id), "attached_file_id", af.Str("id")))
	e2e.ExpectStatus(t, res, http.StatusOK)
	checkAttachedJSON(t, res.JSONObject(t), af)
	expectEq(t, "attached rows", e.DB.Count("SELECT count(*) FROM attached_file WHERE id = ?", af.Str("id")), 0)
	checkNotStored(t, e, e2e.AttachedKey(af))
}

func TestAPIAttachDownload(t *testing.T) {
	e := e2e.New(t)
	if e.Config.Target == "php" {
		t.Skip("PHP 版 (4124ca6) には無い機能")
	}
	owner := e.NewUser("owner")
	app := e.CreateApp(owner, e2e.Unique("app"))
	id := e.MustAPIUpload(app.APIKey, "p", "a.apk", e2e.APK("com.example.a"), nil)
	other := e.MustAPIUpload(app.APIKey, "q", "b.apk", e2e.APK("com.example.a"), nil)

	data := []byte("mapping file contents")
	res := e.NewClient().PostMultipart("/api/package_attach", form("api_key", app.APIKey, "id", itoa(id)),
		e2e.File{Field: "file", Name: "mapping.txt", Data: data, ContentType: "text/plain"})
	e2e.ExpectStatus(t, res, http.StatusOK)
	af := e.DB.Row("SELECT * FROM attached_file WHERE id = ?", jsonNum(t, res.JSONObject(t), "id"))

	res = e.APICall("package_attach_download", app.APIKey, form("id", itoa(id)))
	e2e.ExpectStatus(t, res, http.StatusBadRequest)
	// 別のパッケージの添付ファイルは見つからない
	res = e.APICall("package_attach_download", app.APIKey, form("id", itoa(other), "attached_file_id", af.Str("id")))
	e2e.ExpectStatus(t, res, http.StatusNotFound)

	res = e.APICall("package_attach_download", app.APIKey, form("id", itoa(id), "attached_file_id", af.Str("id")))
	e2e.ExpectStatus(t, res, http.StatusOK)
	m := res.JSONObject(t)
	expectEq(t, "keys", strings.Join(jsonKeys(m), ","), "attached_file,download_url,expire")
	checkAttachedJSON(t, m["attached_file"].(map[string]any), af)
	exp := e.ParseLocalTime(jsonStr(t, m, "expire"))
	if d := time.Until(exp); d < 58*time.Minute || d > 62*time.Minute {
		t.Fatalf("expire = %v (in %v), want about 60 minutes later", m["expire"], d)
	}
	dl := e.Download(jsonStr(t, m, "download_url"))
	e2e.ExpectStatus(t, dl, http.StatusOK)
	if !bytes.Equal(dl.Body, data) {
		t.Fatal("downloaded content mismatch")
	}
}

func TestAPIDelete(t *testing.T) {
	e := e2e.New(t)
	owner := e.NewUser("owner")
	app := e.CreateApp(owner, e2e.Unique("app"))
	other := e.CreateApp(owner, e2e.Unique("other"))

	p1 := e.MustAPIUpload(app.APIKey, "p1", "a.apk", e2e.APK("com.example.a"), nil)
	time.Sleep(1100 * time.Millisecond) // created を別の秒にする
	p2 := e.MustAPIUpload(app.APIKey, "p2", "b.apk", e2e.APK("com.example.a"), form("tags", "t"))
	pkg1, pkg2 := e.Package(p1), e.Package(p2)
	expectEq(t, "last_upload", e.Application(app.ID).Str("last_upload"), pkg2.Str("created"))

	res := e.APICall("delete", other.APIKey, form("id", itoa(p2)))
	e2e.ExpectStatus(t, res, http.StatusBadRequest)
	expectEq(t, "error", jsonStr(t, res.JSONObject(t), "error"), "Invalid package id")

	res = e.APICall("delete", app.APIKey, form("id", itoa(p2)))
	e2e.ExpectStatus(t, res, http.StatusOK)
	expectEq(t, "package rows", e.DB.Count("SELECT count(*) FROM package WHERE id = ?", p2), 0)
	expectEq(t, "package_tag rows", e.DB.Count("SELECT count(*) FROM package_tag WHERE package_id = ?", p2), 0)
	if e.PHPBug("api/delete で最新パッケージを消すと, 1 つ前のパッケージのファイルを消し, 削除対象のファイルが残る. レスポンスも 1 つ前のパッケージになる") {
		expectEq(t, "response id (bug)", jsonNum(t, res.JSONObject(t), "id"), itoa(p1))
		checkNotStored(t, e, e2e.PackageKey(pkg1))
		checkStored(t, e, e2e.PackageKey(pkg2), nil, "", false)
	} else {
		checkPackageJSONWithoutDB(t, e, res.JSONObject(t), pkg2)
		checkNotStored(t, e, e2e.PackageKey(pkg2))
		checkStored(t, e, e2e.PackageKey(pkg1), nil, "", false)
	}

	a := e.Application(app.ID)
	expectEq(t, "last_upload rolled back", a.Str("last_upload"), pkg1.Str("created"))
	expectEq(t, "date_to_sort", a.Str("date_to_sort"), pkg1.Str("created"))

	// 最後の 1 つを消すと last_upload は NULL
	res = e.APICall("delete", app.APIKey, form("id", itoa(p1)))
	if e.PHPBug("api/delete で最後のパッケージを消すと, DB からは消えるが 500 を返す (null に対する deleteFile)") {
		e2e.ExpectStatus(t, res, http.StatusInternalServerError)
	} else {
		e2e.ExpectStatus(t, res, http.StatusOK)
		checkPackageJSONWithoutDB(t, e, res.JSONObject(t), pkg1)
		checkNotStored(t, e, e2e.PackageKey(pkg1))
	}
	expectEq(t, "package rows", e.DB.Count("SELECT count(*) FROM package WHERE id = ?", p1), 0)
	a = e.Application(app.ID)
	expectEq(t, "last_upload null", a.Null("last_upload"), true)
	expectEq(t, "date_to_sort", a.Str("date_to_sort"), a.Str("created"))
}

func TestAPICreateToken(t *testing.T) {
	e := e2e.New(t)
	owner := e.NewUser("owner")
	app := e.CreateApp(owner, e2e.Unique("app"))
	apk := e2e.APK("com.example.a")
	id := e.MustAPIUpload(app.APIKey, "p", "a.apk", apk, nil)

	res := e.APICall("create_token", app.APIKey, form("id", itoa(id), "mail", "stranger@example.com"))
	e2e.ExpectStatus(t, res, http.StatusBadRequest)
	expectEq(t, "error", jsonStr(t, res.JSONObject(t), "error"), "Invalid mail address")

	res = e.APICall("create_token", app.APIKey, form("id", "99999999", "mail", owner.Mail))
	e2e.ExpectStatus(t, res, http.StatusBadRequest)
	expectEq(t, "error", jsonStr(t, res.JSONObject(t), "error"), "Invalid package id")

	res = e.APICall("create_token", app.APIKey, form("id", itoa(id), "mail", owner.Mail, "expire_hour", "3"))
	e2e.ExpectStatus(t, res, http.StatusOK)
	m := res.JSONObject(t)
	installURL := jsonStr(t, m, "install_url")
	delete(m, "install_url")
	checkPackageJSON(t, e, m, e.Package(id))

	prefix := e.Config.BaseURL + "/package/install?token="
	if !strings.HasPrefix(installURL, prefix) {
		t.Fatalf("install_url = %q", installURL)
	}
	token := strings.TrimPrefix(installURL, prefix)
	expectMatch(t, "token", reToken32, token)

	// memcached の形式 (Go 版と PHP 版で共有するため固定する)
	raw := e.MemcacheGet("pkg_install_token_" + token)
	if raw == nil {
		t.Fatalf("install token is not in memcached (prefix %q)", e.Config.CachePrefix)
	}
	var td map[string]any
	if err := json.Unmarshal(raw, &td); err != nil {
		t.Fatalf("token data is not json: %q", raw)
	}
	expectEq(t, "token.mail", td["mail"], any(owner.Mail))
	expectEq(t, "token.package_id", td["package_id"], any(itoa(id)))
	exp := e.ParseLocalTime(td["expire"].(string))
	if d := time.Until(exp); d < 3*time.Hour-2*time.Minute || d > 3*time.Hour+2*time.Minute {
		t.Fatalf("token.expire = %v (in %v), want about 3h later in %s", td["expire"], d, e.Config.TZ)
	}

	// トークンだけで (ログインせずに) インストールできる
	c := e.NewClient()
	c.UserAgent = e2e.UAAndroid
	res = c.Get(installURL)
	e2e.ExpectStatus(t, res, http.StatusFound)
	dl := e.Download(res.Location())
	e2e.ExpectStatus(t, dl, http.StatusOK)
	if !bytes.Equal(dl.Body, apk) {
		t.Fatal("downloaded content mismatch")
	}
	log := e.DB.Row("SELECT * FROM install_log WHERE package_id = ?", id)
	expectEq(t, "install_log.mail", log.Str("mail"), owner.Mail)

	res = e.NewClient().Get("/package/install?token=invalidtoken")
	e2e.ExpectStatus(t, res, http.StatusForbidden)
}

func sortedTagNames(e *e2e.Env, pkgID int64) []string {
	var names []string
	for _, r := range e.DB.Rows("SELECT t.name FROM package_tag j JOIN tag t ON t.id = j.tag_id WHERE j.package_id = ? ORDER BY t.name", pkgID) {
		names = append(names, r.Str("name"))
	}
	return names
}
