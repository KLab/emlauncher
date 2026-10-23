package scenario

import (
	"net/http"
	"regexp"
	"strings"
	"testing"

	"emlauncher-e2e/internal/fixture"
	"emlauncher-e2e/internal/harness"
)

func TestUIUploadIPA(t *testing.T) {
	e := harness.New(t)
	owner := e.NewUser("owner")
	app := e.CreateApp(owner, harness.Unique("app"))
	harness.ExpectStatus(t, owner.Get("/app/upload?id="+itoa(app.ID)), http.StatusOK)

	ipa := fixture.IPA("Sample", "com.example.e2e.ui")
	res := harness.UIUploadTemp(owner, "sample.ipa", ipa)
	harness.ExpectStatus(t, res, http.StatusOK)
	temp := res.JSONObject(t)
	expectEq(t, "file_name", jsonStr(t, temp, "file_name"), "sample.ipa")
	expectEq(t, "platform", jsonStr(t, temp, "platform"), "iOS")
	expectEq(t, "identifier", jsonStr(t, temp, "identifier"), "com.example.e2e.ui")
	expectEq(t, "attached_files", len(jsonList(t, temp, "attached_files")), 0)
	tempName := jsonStr(t, temp, "temp_name")
	expectMatch(t, "temp_name", regexp.MustCompile(`^[0-9A-Za-z]{16}\.ipa$`), tempName)
	checkStored(t, e, "temp-data/"+tempName, ipa, mimeStored, false)

	f := harness.UIUploadPostForm(temp, len(ipa))
	f.Set("title", "ui upload")
	f.Set("description", "from ui")
	f.Add("tags[]", "ui")
	f.Add("tags[]", "beta")
	f.Add("tags[]", "")
	f.Set("protect", "1")
	res = owner.PostForm("/app/upload_post?id="+itoa(app.ID), f)
	owner.ExpectRedirect(res, "/package?id=")
	pkg := e.Package(harness.QueryInt(t, res.Location(), "id"))

	expectEq(t, "app_id", pkg.Int("app_id"), app.ID)
	expectEq(t, "platform", pkg.Str("platform"), "iOS")
	expectEq(t, "title", pkg.Str("title"), "ui upload")
	expectEq(t, "description", pkg.Str("description"), "from ui")
	expectEq(t, "identifier", pkg.Str("identifier"), "com.example.e2e.ui")
	expectEq(t, "original_file_name", pkg.Str("original_file_name"), "sample.ipa")
	expectEq(t, "file_size", pkg.Int("file_size"), int64(len(ipa)))
	expectEq(t, "protect", pkg.Str("protect"), "1")
	expectMatch(t, "file_name", regexp.MustCompile(`^[0-9A-Za-z]{16}\.ipa$`), pkg.Str("file_name"))
	e.ExpectRecent("created", pkg.Str("created"))
	expectEq(t, "tags", strings.Join(sortedTagNames(e, pkg.Int("id")), ","), "beta,ui")

	// 一時ファイルは正式なキーへ移動する
	checkNotStored(t, e, "temp-data/"+tempName)
	checkStored(t, e, harness.PackageKey(pkg), ipa, mimeStored, false)
	expectEq(t, "last_upload", e.Application(app.ID).Str("last_upload"), pkg.Str("created"))

	page := owner.Get("/package?id=" + pkg.Str("id"))
	harness.ExpectStatus(t, page, http.StatusOK)
	expectContains(t, "package page", page.String(), "ui upload")
}

func TestUIUploadAAB(t *testing.T) {
	e := harness.New(t)
	owner := e.NewUser("owner")
	app := e.CreateApp(owner, harness.Unique("app"))

	aab := fixture.AAB("com.example.e2e.uibundle")
	res := harness.UIUploadTemp(owner, "bundle.aab", aab)
	harness.ExpectStatus(t, res, http.StatusOK)
	temp := res.JSONObject(t)
	expectEq(t, "platform", jsonStr(t, temp, "platform"), "Android")
	expectEq(t, "identifier", jsonStr(t, temp, "identifier"), "com.example.e2e.uibundle")
	afs := jsonList(t, temp, "attached_files")
	expectEq(t, "attached_files", len(afs), 1)
	af := afs[0].(map[string]any)
	expectEq(t, "attached.file_name", jsonStr(t, af, "file_name"), "bundle.apk")
	expectEq(t, "attached.type", jsonStr(t, af, "type"), "apk")
	afTemp := jsonStr(t, af, "temp_name")
	expectMatch(t, "attached.temp_name", regexp.MustCompile(`^[0-9A-Za-z]{16}\.apk$`), afTemp)
	checkStored(t, e, "temp-data/"+afTemp, nil, mimeStored, false)
	apkData := e.Storage.Read("temp-data/" + afTemp)
	expectEq(t, "attached.size", jsonNum(t, af, "size"), itoa(int64(len(apkData))))

	f := harness.UIUploadPostForm(temp, len(aab))
	f.Set("title", "ui aab")
	res = owner.PostForm("/app/upload_post?id="+itoa(app.ID), f)
	owner.ExpectRedirect(res, "/package?id=")
	pkg := e.Package(harness.QueryInt(t, res.Location(), "id"))
	expectEq(t, "protect", pkg.Str("protect"), "0")
	checkStored(t, e, harness.PackageKey(pkg), aab, mimeStored, false)

	att := e.DB.Row("SELECT * FROM attached_file WHERE package_id = ?", pkg.Str("id"))
	expectEq(t, "attached.original_file_name", att.Str("original_file_name"), "bundle.apk")
	expectEq(t, "attached.file_type", att.Str("file_type"), "apk")
	expectEq(t, "attached.file_size", att.Int("file_size"), int64(len(apkData)))
	checkStored(t, e, harness.AttachedKey(att), apkData, mimeStored, false)
	checkNotStored(t, e, "temp-data/"+afTemp)
}

func TestUIUploadPermission(t *testing.T) {
	e := harness.New(t)
	owner := e.NewUser("owner")
	other := e.NewUser("other")
	app := e.CreateApp(owner, harness.Unique("app"))

	expectErrorPage(t, e, other.Get("/app/upload?id="+itoa(app.ID)), http.StatusForbidden, "Permission Denied")

	res := harness.UIUploadTemp(owner, "a.apk", fixture.APK("com.example.a"))
	harness.ExpectStatus(t, res, http.StatusOK)
	f := harness.UIUploadPostForm(res.JSONObject(t), 10)
	f.Set("title", "hijack")
	expectErrorPage(t, e, other.PostForm("/app/upload_post?id="+itoa(app.ID), f), http.StatusForbidden, "Permission Denied")
	expectEq(t, "packages", e.DB.Count("SELECT count(*) FROM package WHERE app_id = ?", app.ID), 0)

	// title が無ければ 400
	f.Del("title")
	harness.ExpectStatus(t, owner.PostForm("/app/upload_post?id="+itoa(app.ID), f), http.StatusBadRequest)
}
