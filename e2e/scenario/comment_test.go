package scenario

import (
	"net/http"
	"testing"

	"github.com/KLabTools/emlauncher/e2e/internal/fixture"
	"github.com/KLabTools/emlauncher/e2e/internal/harness"
)

func TestComments(t *testing.T) {
	e := harness.New(t)
	owner := e.NewUser("owner")
	coOwner := e.NewUser("coowner")
	app := e.CreateApp(owner, harness.Unique("app"))
	owner.PostForm("/app/preferences_update_owners", form("id", itoa(app.ID), "owners[]", coOwner.Mail))
	pid := e.MustAPIUpload(app.APIKey, "commented build", "a.apk", fixture.APK("com.example.a"), nil)

	user := e.NewUser("user")
	harness.ExpectStatus(t, user.Get("/package/install?id="+itoa(pid)), http.StatusFound)
	harness.ExpectStatus(t, user.Get("/app/comment?id="+itoa(app.ID)), http.StatusOK)

	res := user.PostForm("/app/comment_post", form("id", itoa(app.ID), "message", "first comment コメント", "package_id", itoa(pid)))
	user.ExpectRedirect(res, "/app/comment?id="+itoa(app.ID))
	res = user.PostForm("/app/comment_post", form("id", itoa(app.ID), "message", "second comment", "package_id", ""))
	user.ExpectRedirect(res, "/app/comment?id="+itoa(app.ID))

	rows := e.DB.Rows("SELECT * FROM comment WHERE app_id = ? ORDER BY id", app.ID)
	expectEq(t, "comments", len(rows), 2)
	expectEq(t, "number", rows[0].Str("number"), "1")
	expectEq(t, "number", rows[1].Str("number"), "2")
	expectEq(t, "package_id", rows[0].Int("package_id"), pid)
	expectEq(t, "package_id null", rows[1].Null("package_id"), true)
	expectEq(t, "mail", rows[0].Str("mail"), user.Mail)
	expectEq(t, "message", rows[0].Str("message"), "first comment コメント")
	e.ExpectRecent("created", rows[1].Str("created"))

	a := e.Application(app.ID)
	expectEq(t, "last_commented", a.Str("last_commented"), rows[1].Str("created"))
	expectEq(t, "date_to_sort", a.Str("date_to_sort"), rows[1].Str("created"))

	page := user.Get("/app/comment?id=" + itoa(app.ID))
	harness.ExpectStatus(t, page, http.StatusOK)
	expectContains(t, "comment page", page.String(), "first comment コメント")
	expectContains(t, "app page", user.Get("/app?id="+itoa(app.ID)).String(), "second comment")

	// オーナー全員に通知される
	for _, o := range []*harness.User{owner, coOwner} {
		mails := e.Mailbox.WaitTo(o.Mail, 2)
		expectEq(t, "subject", mails[0].Subject, "New Comment to "+app.Title)
		expectContains(t, "body", mails[0].Body, "first comment コメント")
		expectContains(t, "body", mails[0].Body, "Target Package: commented build")
		expectContains(t, "body", mails[0].Body, e.Config.BaseURL+"/app/comment?id="+itoa(app.ID)+"#comment-1")
		expectContains(t, "body", mails[1].Body, "#comment-2")
	}
	expectEq(t, "mails to commenter", len(e.Mailbox.To(user.Mail)), 0)
}
