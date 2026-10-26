package scenario

import (
	"net/http"
	"testing"
	"time"

	e2e "emlauncher-e2e"
)

func TestUploadNotification(t *testing.T) {
	e := e2e.New(t)
	owner := e.NewUser("owner")
	app := e.CreateApp(owner, e2e.Unique("app"))
	first := e.MustAPIUpload(app.APIKey, "first", "a.apk", e2e.APK("com.example.a"), nil)

	subscriber := e.NewUser("subscriber")
	muted := e.NewUser("muted")
	for _, u := range []*e2e.User{subscriber, muted} {
		e2e.ExpectStatus(t, u.Get("/package/install?id="+itoa(first)), http.StatusFound)
	}
	e2e.ExpectStatus(t, muted.Get("/ajax/notification_setting?id="+itoa(app.ID)+"&value=0"), http.StatusOK)

	// notify 無しでは送らない
	e.MustAPIUpload(app.APIKey, "silent", "a.apk", e2e.APK("com.example.a"), nil)

	// API から notify 付き
	pid := e.MustAPIUpload(app.APIKey, "notified build", "a.apk", e2e.APK("com.example.a"),
		form("notify", "true", "description", "release note", "tags", "rc"))
	mails := e.Mailbox.WaitTo(subscriber.Mail, 1)
	expectEq(t, "mails", len(mails), 1)
	m := mails[0]
	expectEq(t, "subject", m.Subject, "New Package Uploaded to "+app.Title)
	expectContains(t, "body", m.Body, `New package "notified build" was uploaded to "`+app.Title+`".`)
	expectContains(t, "body", m.Body, "release note")
	expectContains(t, "body", m.Body, "Platform: Android")
	expectContains(t, "body", m.Body, "rc")
	expectContains(t, "body", m.Body, "Package URL: "+e.Config.BaseURL+"/package?id="+itoa(pid))
	// 受信者同士でアドレスが見えないよう Bcc で送る
	expectEq(t, "Bcc", m.Header.Get("Bcc") != "", true)
	expectContains(t, "To", m.Header.Get("To"), "bcc@example.com")

	// 画面から notify 付き
	res := e2e.UIUploadTemp(owner, "b.apk", e2e.APK("com.example.a"))
	f := e2e.UIUploadPostForm(res.JSONObject(t), 10)
	f.Set("title", "ui notified")
	f.Set("notify", "1")
	owner.ExpectRedirect(owner.PostForm("/app/upload_post?id="+itoa(app.ID), f), "/package?id=")
	mails = e.Mailbox.WaitTo(subscriber.Mail, 2)
	expectContains(t, "body", mails[1].Body, `"ui notified"`)

	time.Sleep(500 * time.Millisecond)
	expectEq(t, "mails to muted", len(e.Mailbox.To(muted.Mail)), 0)
	expectEq(t, "mails to owner", len(e.Mailbox.To(owner.Mail)), 0)
}
