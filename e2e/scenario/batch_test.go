package scenario

import (
	"testing"

	"emlauncher-e2e/internal/fixture"
	"emlauncher-e2e/internal/harness"
)

// TestDeleteOldPackagesBatch は古いパッケージ削除バッチを確認する.
// バッチは DB の全アプリが対象なので, 他のテストのアプリのパッケージも消えうる.
func TestDeleteOldPackagesBatch(t *testing.T) {
	e := harness.New(t)
	owner := e.NewUser("owner")
	app := e.CreateApp(owner, harness.Unique("app"))

	var ids []int64
	ids = append(ids, e.MustAPIUpload(app.APIKey, "p0 protected", "a.apk", fixture.APK("com.example.a"), form("protect", "1")))
	ids = append(ids, e.MustAPIUpload(app.APIKey, "p1", "b.aab", fixture.AAB("com.example.a"), form("tags", "t")))
	for i := 2; i < 6; i++ {
		ids = append(ids, e.MustAPIUpload(app.APIKey, "p", "c.apk", fixture.APK("com.example.a"), nil))
	}
	var pkgs []harness.Row
	for _, id := range ids {
		pkgs = append(pkgs, e.Package(id))
	}
	att := e.DB.Row("SELECT * FROM attached_file WHERE package_id = ?", ids[1])
	lastUpload := e.Application(app.ID).Str("last_upload")

	t.Log(e.RunBatch(3))

	// 保護されていないものは新しい 3 つを残す. 保護されたものは残る.
	for i, id := range ids {
		exists := e.DB.Count("SELECT count(*) FROM package WHERE id = ?", id) == 1
		want := i == 0 || i >= 3
		expectEq(t, "package "+itoa(id)+" exists", exists, want)
		if want {
			checkStored(t, e, harness.PackageKey(pkgs[i]), nil, "", false)
		} else {
			checkNotStored(t, e, harness.PackageKey(pkgs[i]))
		}
	}
	expectEq(t, "package_tag", e.DB.Count("SELECT count(*) FROM package_tag WHERE package_id = ?", ids[1]), 0)
	expectEq(t, "attached_file", e.DB.Count("SELECT count(*) FROM attached_file WHERE package_id = ?", ids[1]), 0)
	checkNotStored(t, e, harness.AttachedKey(att))
	expectEq(t, "last_upload", e.Application(app.ID).Str("last_upload"), lastUpload)
}
