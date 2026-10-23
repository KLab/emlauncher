package scenario

import (
	"image/color"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"testing"

	"emlauncher-e2e/internal/fixture"
	"emlauncher-e2e/internal/harness"
)

func TestCreateApp(t *testing.T) {
	e := harness.New(t)
	owner := e.NewUser("owner")
	title := harness.Unique("app")
	app := e.CreateApp(owner, title)

	a := e.Application(app.ID)
	expectEq(t, "title", a.Str("title"), title)
	expectEq(t, "description", a.Str("description"), "description of "+title)
	expectEq(t, "repository", a.Str("repository"), "https://example.com/"+title+".git")
	expectMatch(t, "api_key", reRand16, a.Str("api_key"))
	expectMatch(t, "icon_key", regexp.MustCompile(`^app-icons/`+itoa(app.ID)+`/[0-9A-Za-z]{16}\.png$`), a.Str("icon_key"))
	e.ExpectRecent("created", a.Str("created"))
	expectEq(t, "date_to_sort", a.Str("date_to_sort"), a.Str("created"))
	expectEq(t, "last_upload null", a.Null("last_upload"), true)
	expectEq(t, "last_commented null", a.Null("last_commented"), true)

	owners := e.DB.Rows("SELECT owner_mail FROM application_owner WHERE app_id = ?", app.ID)
	expectEq(t, "owners", len(owners), 1)
	expectEq(t, "owner", owners[0].Str("owner_mail"), owner.Mail)

	// アイコンは 144x144 の PNG, S3 では public-read
	checkStored(t, e, a.Str("icon_key"), nil, "image/png", true)
	w, h := pngSize(t, e.Storage.Read(a.Str("icon_key")))
	expectEq(t, "icon size", [2]int{w, h}, [2]int{144, 144})

	page := owner.Get("/app?id=" + itoa(app.ID))
	harness.ExpectStatus(t, page, http.StatusOK)
	expectContains(t, "app page", page.String(), title)
	expectContains(t, "app page", page.String(), a.Str("icon_key"))

	// 一覧 (トップ, 自分のアプリ)
	expectContains(t, "top page", owner.Get("/").String(), title)
	expectContains(t, "own apps", owner.Get("/myapps/own").String(), title)

	// 他のユーザも閲覧できる
	other := e.NewUser("other")
	harness.ExpectStatus(t, other.Get("/app?id="+itoa(app.ID)), http.StatusOK)
	if strings.Contains(other.Get("/myapps/own").String(), title) {
		t.Fatal("other user's own apps contains the app")
	}

	harness.ExpectStatus(t, owner.Get("/app/new"), http.StatusOK)
	expectErrorPage(t, e, owner.Get("/app?id=99999999"), http.StatusNotFound, "Not Found")

	// アイコン無しは作れない
	res := owner.PostForm("/app/create", form("title", "no icon"))
	harness.ExpectStatus(t, res, http.StatusBadRequest)
}

func TestAppPreferences(t *testing.T) {
	e := harness.New(t)
	owner := e.NewUser("owner")
	other := e.NewUser("other")
	app := e.CreateApp(owner, harness.Unique("app"))
	id := itoa(app.ID)

	expectErrorPage(t, e, other.Get("/app/preferences?id="+id), http.StatusForbidden, "Permission Denied")
	harness.ExpectStatus(t, owner.Get("/app/preferences?id="+id), http.StatusOK)

	t.Run("refresh api key", func(t *testing.T) {
		// 古いキーが一致しなければ変わらない
		res := owner.PostForm("/app/preferences_refresh_apikey", form("id", id, "api-key", "wrong"))
		owner.ExpectRedirect(res, "/app/preferences?id="+id)
		expectEq(t, "api_key", e.Application(app.ID).Str("api_key"), app.APIKey)

		res = owner.PostForm("/app/preferences_refresh_apikey", form("id", id, "api-key", app.APIKey))
		owner.ExpectRedirect(res, "/app/preferences?id="+id)
		newKey := e.Application(app.ID).Str("api_key")
		expectMatch(t, "api_key", reRand16, newKey)
		if newKey == app.APIKey {
			t.Fatal("api key not changed")
		}
		harness.ExpectStatus(t, e.APICall("package_list", app.APIKey, nil), http.StatusBadRequest)
		harness.ExpectStatus(t, e.APICall("package_list", newKey, nil), http.StatusOK)
		app.APIKey = newKey
	})

	t.Run("update info", func(t *testing.T) {
		oldIcon := e.Application(app.ID).Str("icon_key")
		res := owner.PostForm("/app/preferences_update", form("id", id,
			"title", app.Title+"-renamed", "description", "new desc", "repository", "https://example.com/new.git"))
		owner.ExpectRedirect(res, "/app/preferences?id="+id)
		a := e.Application(app.ID)
		expectEq(t, "title", a.Str("title"), app.Title+"-renamed")
		expectEq(t, "description", a.Str("description"), "new desc")
		expectEq(t, "repository", a.Str("repository"), "https://example.com/new.git")
		expectEq(t, "icon kept", a.Str("icon_key"), oldIcon)

		res = owner.PostForm("/app/preferences_update", form("id", id, "title", app.Title,
			"icon-data", harness.IconDataURL(fixture.PNG(50, 80, color.RGBA{0xff, 0, 0, 0xff}))))
		owner.ExpectRedirect(res, "/app/preferences?id="+id)
		newIcon := e.Application(app.ID).Str("icon_key")
		if newIcon == oldIcon {
			t.Fatal("icon_key not changed")
		}
		checkStored(t, e, newIcon, nil, "image/png", true)
		checkNotStored(t, e, oldIcon)

		harness.ExpectStatus(t, owner.PostForm("/app/preferences_update", form("id", id, "title", "")), http.StatusBadRequest)
		expectErrorPage(t, e, other.PostForm("/app/preferences_update", form("id", id, "title", "hijack")), http.StatusForbidden, "Permission Denied")
		expectEq(t, "title", e.Application(app.ID).Str("title"), app.Title)
	})

	t.Run("owners", func(t *testing.T) {
		res := owner.PostForm("/app/preferences_update_owners", form("id", id, "owners[]", other.Mail, "owners[]", ""))
		owner.ExpectRedirect(res, "/app/preferences?id="+id)
		expectEq(t, "owners", ownerMails(e, app.ID), sortedJoin(owner.Mail, other.Mail))
		harness.ExpectStatus(t, other.Get("/app/preferences?id="+id), http.StatusOK)

		// 更新した本人は必ず残る
		res = other.PostForm("/app/preferences_update_owners", form("id", id, "owners[]", ""))
		other.ExpectRedirect(res, "/app/preferences?id="+id)
		expectEq(t, "owners", ownerMails(e, app.ID), other.Mail)
		expectErrorPage(t, e, owner.Get("/app/preferences?id="+id), http.StatusForbidden, "Permission Denied")

		other.PostForm("/app/preferences_update_owners", form("id", id, "owners[]", owner.Mail))
		expectEq(t, "owners", ownerMails(e, app.ID), sortedJoin(owner.Mail, other.Mail))
	})

	t.Run("delete tags", func(t *testing.T) {
		p := e.MustAPIUpload(app.APIKey, "p", "a.apk", fixture.APK("com.example.a"), form("tags", "keep,drop"))
		drop := e.DB.Row("SELECT id FROM tag WHERE app_id = ? AND name = 'drop'", app.ID).Str("id")
		res := owner.PostForm("/app/preferences_delete_tags", form("id", id, "tags[]", drop))
		owner.ExpectRedirect(res, "/app/preferences?id="+id)
		expectEq(t, "tags", strings.Join(sortedTagNames(e, p), ","), "keep")
		expectEq(t, "tag rows", e.DB.Count("SELECT count(*) FROM tag WHERE id = ?", drop), 0)
	})
}

func ownerMails(e *harness.Env, appID int64) string {
	var mails []string
	for _, r := range e.DB.Rows("SELECT owner_mail FROM application_owner WHERE app_id = ? ORDER BY owner_mail", appID) {
		mails = append(mails, r.Str("owner_mail"))
	}
	return strings.Join(mails, ",")
}

func sortedJoin(ss ...string) string {
	sort.Strings(ss)
	return strings.Join(ss, ",")
}
