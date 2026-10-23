package migration

import (
	e2e "emlauncher-e2e"
	"net/http"
	"net/url"
	"regexp"
	"testing"
)

var (
	reResetKey        = regexp.MustCompile(`/login/password_reset\?key=([0-9a-f]{40})`)
	reInstallTokenURL = regexp.MustCompile(`/package/install\?token=([0-9A-Za-z]{32})`)
)

// TestSeed は移行元 (PHP 版) に, 運用中にありそうな中間状態を画面と API だけで作る.
// 空の環境 (e2e/env/up.sh 直後) で実行すること.
func TestSeed(t *testing.T) {
	e := phase(t, "seed")
	s := &State{
		Users:       map[string]UserState{},
		Apps:        map[string]AppState{},
		Packages:    map[string]PkgState{},
		PackageList: map[string][]PackageSummary{},
	}

	// owner: 管理者が mail だけ登録し, 本人がパスワードリセットで設定する (README の手順)
	ownerMail := "owner@example.com"
	e.DB.Exec("INSERT INTO user_pass (mail) VALUES (?)", ownerMail)
	oc := e.NewClient()
	e2e.ExpectStatus(t, oc.PostForm("/login/password_confirm", url.Values{"email": {ownerMail}}), http.StatusOK)
	key := reResetKey.FindStringSubmatch(e.Mailbox.WaitTo(ownerMail, 1)[0].Body)[1]
	e2e.ExpectStatus(t, oc.PostForm("/login/password_commit", url.Values{"key": {key}, "password": {"owner-password"}}), http.StatusOK)
	oc.Login(ownerMail, "owner-password")
	owner := &e2e.User{Mail: ownerMail, Password: "owner-password", Client: oc}
	s.Users["owner"] = UserState{owner.Mail, owner.Password}

	// member: passhash を直接持つユーザ
	member := e.NewUser("member")
	s.Users["member"] = UserState{member.Mail, member.Password}

	mainApp := e.CreateApp(owner, "Main App")
	subApp := e.CreateApp(member, "Sub App")
	owner.ExpectRedirect(owner.PostForm("/app/preferences_update_owners",
		url.Values{"id": {itoa(mainApp.ID)}, "owners[]": {member.Mail}}), "/app/preferences")
	// API キーを一度更新しておく
	owner.PostForm("/app/preferences_refresh_apikey", url.Values{"id": {itoa(mainApp.ID)}, "api-key": {mainApp.APIKey}})
	mainApp.APIKey = e.Application(mainApp.ID).Str("api_key")
	s.Apps["main"] = AppState{mainApp.ID, mainApp.Title, mainApp.APIKey}
	s.Apps["sub"] = AppState{subApp.ID, subApp.Title, subApp.APIKey}

	addPkg := func(name, app string, id int64) {
		pkg := e.Package(id)
		k := e2e.PackageKey(pkg)
		androidKey := k
		if apk := e.DB.Rows("SELECT * FROM attached_file WHERE package_id = ? AND file_type = 'apk'", id); len(apk) > 0 {
			androidKey = e2e.AttachedKey(apk[0])
		}
		s.Packages[name] = PkgState{
			ID: id, App: app, Title: pkg.Str("title"), FileName: pkg.Str("original_file_name"),
			Key: k, SHA256: sha(e.Storage.Read(k)), AndroidKey: androidKey,
		}
	}

	addPkg("ipa", "main", e.MustAPIUpload(mainApp.APIKey, "iOS release", "Main.ipa",
		e2e.IPA("Main", "com.example.main"), url.Values{"tags": {"ios,release"}, "protect": {"true"}}))
	addPkg("apk", "main", e.MustAPIUpload(mainApp.APIKey, "Android build", "main.apk",
		e2e.APK("com.example.main"), url.Values{"tags": {"android"}, "description": {"first android build"}}))

	res := e.APIUpload(mainApp.APIKey, url.Values{"title": {"Bundle build"}, "tags": {"android"}},
		e2e.File{Field: "file", Name: "main.aab", Data: e2e.AAB("com.example.main")},
		e2e.File{Field: "dsym", Name: "main.dSYM.zip", Data: []byte("dsym"), ContentType: "application/zip"})
	e2e.ExpectStatus(t, res, http.StatusOK)
	addPkg("aab", "main", e2e.QueryInt(t, res.JSONObject(t)["package_url"].(string), "id"))

	// 画面からのアップロード
	tmp := e2e.UIUploadTemp(owner, "ui.ipa", e2e.IPA("UI", "com.example.main"))
	e2e.ExpectStatus(t, tmp, http.StatusOK)
	f := e2e.UIUploadPostForm(tmp.JSONObject(t), 600)
	f.Set("title", "UI upload")
	f.Add("tags[]", "ios")
	res = owner.PostForm("/app/upload_post?id="+itoa(mainApp.ID), f)
	owner.ExpectRedirect(res, "/package?id=")
	addPkg("ui-ipa", "main", e2e.QueryInt(t, res.Location(), "id"))

	addPkg("txt", "main", e.MustAPIUpload(mainApp.APIKey, "Readme", "readme.txt", []byte("readme"), nil))
	addPkg("sub-apk", "sub", e.MustAPIUpload(subApp.APIKey, "Sub build", "sub.apk", e2e.APK("com.example.sub"), nil))

	// 削除済みパッケージ (id の欠番)
	gone := e.MustAPIUpload(mainApp.APIKey, "to be deleted", "gone.apk", e2e.APK("com.example.main"), nil)
	confirm := owner.Get("/package/delete_confirm?id=" + itoa(gone))
	owner.ExpectRedirect(owner.PostForm("/package/delete",
		url.Values{"id": {itoa(gone)}, "token": {e2e.HiddenToken(t, confirm)}}), "/app?id=")

	// 画面から添付・編集
	apkID := itoa(s.Packages["apk"].ID)
	owner.ExpectRedirect(owner.PostMultipart("/package/attach", url.Values{"id": {apkID}},
		e2e.File{Field: "file", Name: "notes.txt", Data: []byte("release notes")}), "/package?id="+apkID)
	owner.ExpectRedirect(owner.PostForm("/package/edit_commit?id="+apkID, url.Values{
		"title": {"Android build (edited)"}, "description": {"first android build"}, "tags[]": {"android", "edited"}}), "/package?id="+apkID)
	p := s.Packages["apk"]
	p.Title = "Android build (edited)"
	s.Packages["apk"] = p

	// インストール
	member.UserAgent = e2e.UAAndroid
	for _, name := range []string{"apk", "aab"} {
		e2e.ExpectStatus(t, member.Get("/package/install?id="+itoa(s.Packages[name].ID)), http.StatusFound)
	}
	member.UserAgent = e2e.UAiOS
	e2e.ExpectStatus(t, member.Get("/package/install?id="+itoa(s.Packages["ipa"].ID)), http.StatusFound)
	member.UserAgent = e2e.UAPC
	e2e.ExpectStatus(t, owner.Get("/package/install?id="+itoa(s.Packages["txt"].ID)), http.StatusFound)
	e2e.ExpectStatus(t, member.Get("/package/install?id="+itoa(s.Packages["sub-apk"].ID)), http.StatusFound)
	// member は Main App の更新通知を切る
	e2e.ExpectStatus(t, member.Get("/ajax/notification_setting?id="+itoa(mainApp.ID)+"&value=0"), http.StatusOK)

	// コメント
	for _, c := range []struct {
		u   *e2e.User
		msg string
		pkg string
	}{
		{member, "member comment on apk", apkID},
		{owner, "owner reply", ""},
		{member, "another comment", ""},
	} {
		c.u.ExpectRedirect(c.u.PostForm("/app/comment_post",
			url.Values{"id": {itoa(mainApp.ID)}, "message": {c.msg}, "package_id": {c.pkg}}), "/app/comment")
		s.CommentCount++
	}

	// ゲストパス (有効 / 失効)
	res = owner.Get("/package/create_guestpass?id=" + apkID)
	gp := e2e.QueryInt(t, res.Location(), "guestpass_id")
	s.GuestPassActive = e.DB.Row("SELECT token FROM guest_pass WHERE id = ?", gp).Str("token")
	ipaID := itoa(s.Packages["ipa"].ID)
	res = owner.Get("/package/create_guestpass?id=" + ipaID)
	gp = e2e.QueryInt(t, res.Location(), "guestpass_id")
	s.GuestPassExpired = e.DB.Row("SELECT token FROM guest_pass WHERE id = ?", gp).Str("token")
	owner.ExpectRedirect(owner.Get("/package/expire_guestpass?id="+ipaID+"&guestpass_id="+itoa(gp)), "/package")
	guest := e.NewClient()
	guest.UserAgent = e2e.UAAndroid
	e2e.ExpectStatus(t, guest.Get("/package/guestpass_install?token="+s.GuestPassActive), http.StatusFound)

	// インストールトークン (memcached)
	res = e.APICall("create_token", mainApp.APIKey, url.Values{
		"id": {itoa(s.Packages["aab"].ID)}, "mail": {owner.Mail}, "expire_hour": {"48"}})
	e2e.ExpectStatus(t, res, http.StatusOK)
	s.InstallTokenAPI = reInstallTokenURL.FindStringSubmatch(res.JSONObject(t)["install_url"].(string))[1]
	page := member.Get("/package/create_token?id=" + apkID)
	s.InstallTokenUI = reInstallTokenURL.FindStringSubmatch(page.String())[1]

	// upload_post されずに残った一時ファイル
	tmp = e2e.UIUploadTemp(owner, "orphan.apk", e2e.APK("com.example.orphan"))
	s.OrphanTemp = "temp-data/" + tmp.JSONObject(t)["temp_name"].(string)

	s.PackageList["main"] = packageList(t, e, mainApp.APIKey)
	s.PackageList["sub"] = packageList(t, e, subApp.APIKey)
	s.DB = snapshotDB(e)
	s.Storage = snapshotStorage(e)
	saveState(t, s)
}

func itoa(n int64) string { return e2e.Itoa(n) }
