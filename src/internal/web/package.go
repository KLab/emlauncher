package web

import (
	"context"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"text/template"
	"time"

	"emlauncher/internal/db"
	"emlauncher/internal/pkgfile"
	"emlauncher/internal/randstr"
	"emlauncher/internal/service"
)

type ctxPackage struct{}

// packageOf はリクエストの id のパッケージとそのアプリ.
func (s *Server) packageOf(r *http.Request) (db.Package, db.Application, error) {
	if pa, ok := r.Context().Value(ctxPackage{}).(pkgAndApp); ok {
		return pa.pkg, pa.app, nil
	}
	return s.loadPackage(r, formOf(r).intParam("id", 0))
}

type pkgAndApp struct {
	pkg db.Package
	app db.Application
}

func (s *Server) loadPackage(r *http.Request, id int64) (db.Package, db.Application, error) {
	ctx := r.Context()
	pkg, err := s.svc.GetPackage(ctx, id)
	if errors.Is(err, service.ErrNotFound) {
		return pkg, db.Application{}, errNotFound
	}
	if err != nil {
		return pkg, db.Application{}, err
	}
	logPackage(r, pkg)
	app, err := s.svc.GetApp(ctx, pkg.AppID)
	return pkg, app, err
}

// logPackage はアクセスログにパッケージを記録する.
func logPackage(r *http.Request, p db.Package) {
	logAttr(r, "app_id", p.AppID)
	logAttr(r, "pkg_id", p.ID)
	logAttr(r, "platform", p.Platform)
}

// requirePackageOwner はリクエストの id のパッケージのアプリのオーナーでなければ Permission Denied にする.
func (s *Server) requirePackageOwner(next http.Handler) http.Handler {
	return s.page(func(w http.ResponseWriter, r *http.Request) error {
		pkg, app, err := s.packageOf(r)
		if err != nil {
			return err
		}
		owner, err := s.svc.IsOwner(r.Context(), app.ID, loginUser(r))
		if err != nil {
			return err
		}
		if !owner {
			return errPermissionDenied
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxPackage{}, pkgAndApp{pkg, app})))
		return nil
	})
}

// packagePage はパッケージの画面に共通のデータ.
type packagePage struct {
	App     *appView
	Package *pkgView
}

func (s *Server) packagePage(r *http.Request) (db.Package, packagePage, error) {
	pkg, app, err := s.packageOf(r)
	if err != nil {
		return pkg, packagePage{}, err
	}
	av, err := s.appView(r.Context(), app, loginUser(r))
	if err != nil {
		return pkg, packagePage{}, err
	}
	d, err := s.svc.Detail(r.Context(), pkg)
	if err != nil {
		return pkg, packagePage{}, err
	}
	installed, err := s.svc.InstalledDates(r.Context(), app.ID, loginUser(r))
	if err != nil {
		return pkg, packagePage{}, err
	}
	return pkg, packagePage{App: av, Package: s.pkgView(d, installed)}, nil
}

func (s *Server) packageIndex(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	pkg, pp, err := s.packagePage(r)
	if err != nil {
		return err
	}
	users, err := s.svc.PackageInstallUsers(ctx, pkg.ID)
	if err != nil {
		return err
	}
	type guestPass struct {
		ID, PackageID int64
		Expired       string
		InstallCount  int64
	}
	var gps []guestPass
	if pp.App.IsOwner {
		rows, err := s.svc.ActiveGuestPasses(ctx, pkg.ID)
		if err != nil {
			return err
		}
		for _, g := range rows {
			gps = append(gps, guestPass{ID: g.ID, PackageID: g.PackageID, Expired: s.svc.FormatTime(g.Expired), InstallCount: g.InstallCount})
		}
	}
	type attached struct {
		ID         int64
		Name, Size string
	}
	files, err := s.svc.Queries().ListAttachedFiles(ctx, pkg.ID)
	if err != nil {
		return err
	}
	var att []attached
	for _, a := range files {
		att = append(att, attached{ID: a.ID, Name: a.OriginalFileName, Size: formatSize(a.FileSize)})
	}
	return s.renderOK(w, r, "package/index", struct {
		packagePage
		InstallUsers []string
		GuestPasses  []guestPass
		Attached     []attached
	}{pp, users, gps, att})
}

// packageInstall はパッケージのダウンロード (iOS 端末では itms-services) にリダイレクトし,
// インストール履歴を記録する. token (インストールトークン) があれば, ログインせずにトークンを
// 発行したユーザとしてインストールできる.
func (s *Server) packageInstall(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	f := formOf(r)
	var pkg db.Package
	var mail string
	if f.has("token") {
		t, err := s.svc.InstallTokenByToken(ctx, f.Get("token"))
		if errors.Is(err, service.ErrNotFound) {
			s.log.Warn("invalid install token", "token", f.Get("token"))
			return errorStatus(http.StatusForbidden, "invalid token")
		}
		if err != nil {
			return err
		}
		if pkg, err = s.svc.GetPackage(ctx, t.PackageID); err != nil {
			return err
		}
		mail = t.Mail
		logUser(r, mail)
		logPackage(r, pkg)
	} else {
		sess := s.sessions.load(r)
		if sess.Mail == "" {
			s.redirectToLogin(w, r, sess)
			return nil
		}
		var err error
		if pkg, _, err = s.loadPackage(r, f.intParam("id", 0)); err != nil {
			return err
		}
		mail = sess.Mail
	}
	u, err := s.installURL(r, pkg)
	if err != nil {
		return err
	}
	if err := s.svc.RecordInstall(ctx, pkg, mail, r.UserAgent()); err != nil {
		return err
	}
	return redirect(w, r, u)
}

// installURL は端末に合わせたインストールの URL. iOS 端末で IPA なら itms-services (plist の URL に
// 短時間のトークンを付ける), Android 端末で AAB なら生成した APK, それ以外はパッケージそのもの.
func (s *Server) installURL(r *http.Request, pkg db.Package) (string, error) {
	ctx := r.Context()
	if pkg.Platform == string(pkgfile.IOS) && isIOS(r) {
		token, err := s.svc.CreatePlistToken(ctx, pkg.ID)
		if err != nil {
			return "", err
		}
		scheme := ""
		if s.cfg.EnableHTTPS {
			scheme = "https"
		}
		plist := absURL(r, scheme, "/package/install_plist?"+url.Values{"id": {fmt.Sprint(pkg.ID)}, "t": {token}}.Encode())
		return "itms-services://?action=download-manifest&url=" + url.QueryEscape(plist), nil
	}
	return s.svc.PackageDownloadURL(ctx, pkg, isAndroid(r))
}

var plistTemplate = template.Must(template.New("plist").Funcs(template.FuncMap{"xml": html.EscapeString}).Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>items</key>
    <array>
      <dict>
        <key>assets</key>
        <array>
          <dict>
            <key>kind</key>
            <string>software-package</string>
            <key>url</key>
            <string>{{xml .IPAURL}}</string>
          </dict>
          <dict>
            <key>kind</key>
            <string>full-size-image</string>
            <key>needs-shine</key>
            <true/>
            <key>url</key>
            <string>{{xml .ImageURL}}</string>
          </dict>
          <dict>
            <key>kind</key>
            <string>display-image</string>
            <key>needs-shine</key>
            <true/>
            <key>url</key>
            <string>{{xml .ImageURL}}</string>
          </dict>
        </array>
        <key>metadata</key>
        <dict>
          <key>bundle-identifier</key>
          <string>{{xml .BundleID}}</string>
          <key>bundle-version</key>
          <string>1.0</string>
          <key>kind</key>
          <string>software</string>
          <key>subtitle</key>
          <string>{{xml .Subtitle}}</string>
          <key>title</key>
          <string>{{xml .Title}}</string>
        </dict>
      </dict>
    </array>
  </dict>
</plist>
`))

// installPlist は iOS のインストーラが読む manifest (plist). セッションの代わりに t (トークン) で認証する.
func (s *Server) installPlist(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	f := formOf(r)
	pkg, app, err := s.loadPackage(r, f.intParam("id", 0))
	if err != nil {
		return err
	}
	ok, err := s.svc.CheckPlistToken(ctx, f.Get("t"), pkg.ID)
	if err != nil {
		return err
	}
	if !ok {
		return errPermissionDenied
	}
	ipaURL, err := s.svc.PackageDownloadURL(ctx, pkg, false)
	if err != nil {
		return err
	}
	if len(ipaURL) > 0 && ipaURL[0] == '/' {
		ipaURL = baseURL(r) + ipaURL
	}
	imageURL := s.svc.IconURL(app)
	if len(imageURL) > 0 && imageURL[0] == '/' {
		imageURL = baseURL(r) + imageURL
	}
	w.Header().Set("Content-Type", "text/xml")
	return plistTemplate.Execute(w, map[string]string{
		"IPAURL": ipaURL, "ImageURL": imageURL, "BundleID": pkg.Identifier.String,
		"Subtitle": pkg.Title, "Title": app.Title,
	})
}

// guestpassInstall はゲストパスでのインストール (ログイン不要).
func (s *Server) guestpassInstall(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	token := formOf(r).Get("token")
	gp, err := s.svc.GuestPassByToken(ctx, token)
	if errors.Is(err, service.ErrNotFound) {
		return errNotFound
	}
	if err != nil {
		return err
	}
	logAttr(r, "guestpass_id", gp.ID)
	if s.svc.Expired(gp) {
		s.log.Warn("expired guest pass", "token", token, "app_id", gp.AppID, "package_id", gp.PackageID)
		return errorStatus(http.StatusForbidden, "Invalid token")
	}
	pkg, _, err := s.loadPackage(r, gp.PackageID)
	if err != nil {
		return err
	}
	u, err := s.installURL(r, pkg)
	if err != nil {
		return err
	}
	if err := s.svc.RecordGuestInstall(ctx, gp, r.UserAgent(), remoteIP(r)); err != nil {
		return err
	}
	return redirect(w, r, u)
}

// tokenPage はインストールトークン・ゲストパスの画面.
type tokenPage struct {
	packagePage
	URL         string
	Expire      string
	GeneratedBy string
	QRPath      string
}

// packageCreateToken はインストールトークン (1 時間有効) を発行する. オーナー以外も発行できる.
func (s *Server) packageCreateToken(w http.ResponseWriter, r *http.Request) error {
	pkg, pp, err := s.packagePage(r)
	if err != nil {
		return err
	}
	t, err := s.svc.CreateInstallToken(r.Context(), pkg, loginUser(r), time.Hour)
	if err != nil {
		return err
	}
	path := "/package/install?token=" + t.Token
	return s.renderOK(w, r, "package/create_token", tokenPage{
		packagePage: pp, URL: baseURL(r) + path, Expire: s.svc.FormatTime(t.Expire), GeneratedBy: loginUser(r), QRPath: path,
	})
}

func (s *Server) packageCreateGuestPass(w http.ResponseWriter, r *http.Request) error {
	pkg, app, err := s.packageOf(r)
	if err != nil {
		return err
	}
	owner, err := s.svc.IsOwner(r.Context(), app.ID, loginUser(r))
	if err != nil {
		return err
	}
	if !owner {
		return errNotFound
	}
	gp, err := s.svc.CreateGuestPass(r.Context(), pkg, loginUser(r))
	if err != nil {
		return err
	}
	logAttr(r, "guestpass_id", gp.ID)
	return redirect(w, r, fmt.Sprintf("/package/guestpass?id=%d&guestpass_id=%d", pkg.ID, gp.ID))
}

// guestPassOf はリクエストの guestpass_id のゲストパス. パッケージのアプリのオーナーでなければ notOwner.
func (s *Server) guestPassOf(r *http.Request, pkg db.Package, app db.Application, notOwner error) (db.GuestPass, error) {
	logAttr(r, "guestpass_id", formOf(r).intParam("guestpass_id", 0))
	gp, err := s.svc.GetGuestPass(r.Context(), formOf(r).intParam("guestpass_id", 0))
	if errors.Is(err, service.ErrNotFound) || err == nil && gp.PackageID != pkg.ID {
		return gp, notOwner
	}
	if err != nil {
		return gp, err
	}
	owner, err := s.svc.IsOwner(r.Context(), app.ID, loginUser(r))
	if err != nil {
		return gp, err
	}
	if !owner {
		return gp, notOwner
	}
	return gp, nil
}

func (s *Server) packageGuestPass(w http.ResponseWriter, r *http.Request) error {
	pkg, pp, err := s.packagePage(r)
	if err != nil {
		return err
	}
	_, app, err := s.packageOf(r)
	if err != nil {
		return err
	}
	gp, err := s.guestPassOf(r, pkg, app, errForbidden)
	if err != nil {
		return err
	}
	path := "/package/guestpass_install?token=" + gp.Token
	return s.renderOK(w, r, "package/guestpass", tokenPage{
		packagePage: pp, URL: baseURL(r) + path, Expire: s.svc.FormatTime(gp.Expired), GeneratedBy: loginUser(r), QRPath: path,
	})
}

func (s *Server) packageExpireGuestPass(w http.ResponseWriter, r *http.Request) error {
	pkg, app, err := s.packageOf(r)
	if err != nil {
		return err
	}
	gp, err := s.guestPassOf(r, pkg, app, errNotFound)
	if err != nil {
		return err
	}
	if err := s.svc.ExpireGuestPass(r.Context(), gp.ID); err != nil {
		return err
	}
	return redirect(w, r, fmt.Sprintf("/package/?id=%d", pkg.ID))
}

func (s *Server) packageEdit(w http.ResponseWriter, r *http.Request) error {
	pkg, pp, err := s.packagePage(r)
	if err != nil {
		return err
	}
	tags, err := s.svc.Queries().ListTagsByApp(r.Context(), pkg.AppID)
	if err != nil {
		return err
	}
	var tv []tagView
	for _, t := range tags {
		active := false
		for _, pt := range pp.Package.Tags {
			active = active || pt.ID == t.ID
		}
		tv = append(tv, tagView{ID: t.ID, Name: t.Name, Active: active})
	}
	return s.renderOK(w, r, "package/edit", struct {
		packagePage
		Tags []tagView
	}{pp, tv})
}

func (s *Server) packageEditCommit(w http.ResponseWriter, r *http.Request) error {
	f := formOf(r)
	pkg, _, err := s.packageOf(r)
	if err != nil {
		return err
	}
	title := f.Get("title")
	if title == "" {
		return errBadRequest
	}
	protect := truthy(f.Get("protect"))
	tags := f.Values["tags[]"]
	if tags == nil {
		tags = []string{}
	}
	_, err = s.svc.EditPackage(r.Context(), pkg.ID, service.PackageEdit{
		Title: &title, Description: f.optional("description"), Protect: &protect, Tags: &tags,
	})
	if err != nil {
		return err
	}
	return redirect(w, r, fmt.Sprintf("/package?id=%d", pkg.ID))
}

func (s *Server) packageDeleteConfirm(w http.ResponseWriter, r *http.Request) error {
	_, pp, err := s.packagePage(r)
	if err != nil {
		return err
	}
	sess := s.sessions.load(r)
	sess.PackageDeleteToken = randstr.New(32)
	s.sessions.save(w, r, sess)
	return s.renderOK(w, r, "package/delete_confirm", struct {
		packagePage
		Token string
	}{pp, sess.PackageDeleteToken})
}

// checkToken は削除確認画面で発行したトークンを確かめる. 一致したらセッションから消す (使い捨て).
func (s *Server) checkToken(w http.ResponseWriter, r *http.Request, field func(*session) *string) error {
	sess := s.sessions.load(r)
	want := field(sess)
	// トークンは POST の本文で送られたものだけ受け付ける
	if *want == "" || formOf(r).post.Get("token") != *want {
		return errBadRequest
	}
	*want = ""
	s.sessions.save(w, r, sess)
	return nil
}

func (s *Server) packageDelete(w http.ResponseWriter, r *http.Request) error {
	pkg, app, err := s.packageOf(r)
	if err != nil {
		return err
	}
	if err := s.checkToken(w, r, func(s *session) *string { return &s.PackageDeleteToken }); err != nil {
		return err
	}
	if _, err := s.svc.DeletePackage(r.Context(), pkg.ID); err != nil {
		return err
	}
	return redirect(w, r, fmt.Sprintf("/app?id=%d", app.ID))
}

func (s *Server) packageAttach(w http.ResponseWriter, r *http.Request) error {
	pkg, _, err := s.packageOf(r)
	if err != nil {
		return err
	}
	file := formOf(r).file("file")
	if file == nil {
		return badRequest("file required")
	}
	_, err = s.svc.AttachFile(r.Context(), pkg, service.Attachment{Path: file.Path, Name: file.Name, Size: file.Size})
	if err != nil {
		return err
	}
	return redirect(w, r, fmt.Sprintf("/package?id=%d", pkg.ID))
}

// attachedOf はリクエストの attached_id の, パッケージの添付ファイル.
func (s *Server) attachedOf(r *http.Request, pkg db.Package) (db.AttachedFile, error) {
	logAttr(r, "attached_id", formOf(r).intParam("attached_id", 0))
	a, err := s.svc.GetAttachedFile(r.Context(), pkg.ID, formOf(r).intParam("attached_id", 0))
	if errors.Is(err, service.ErrNotFound) {
		return a, errNotFound
	}
	return a, err
}

// packageAttachDownload は添付ファイルのダウンロードにリダイレクトする. オーナー以外もダウンロードできる.
func (s *Server) packageAttachDownload(w http.ResponseWriter, r *http.Request) error {
	pkg, _, err := s.packageOf(r)
	if err != nil {
		return err
	}
	a, err := s.attachedOf(r, pkg)
	if err != nil {
		return err
	}
	u, err := s.svc.AttachedDownloadURL(r.Context(), a)
	if err != nil {
		return err
	}
	return redirect(w, r, u)
}

func (s *Server) packageAttachDeleteConfirm(w http.ResponseWriter, r *http.Request) error {
	pkg, pp, err := s.packagePage(r)
	if err != nil {
		return err
	}
	a, err := s.attachedOf(r, pkg)
	if err != nil {
		return err
	}
	sess := s.sessions.load(r)
	sess.AttachDeleteToken = randstr.New(32)
	s.sessions.save(w, r, sess)
	return s.renderOK(w, r, "package/attach_delete_confirm", struct {
		packagePage
		Attached struct {
			ID               int64
			OriginalFileName string
		}
		Token string
	}{pp, struct {
		ID               int64
		OriginalFileName string
	}{a.ID, a.OriginalFileName}, sess.AttachDeleteToken})
}

func (s *Server) packageAttachDelete(w http.ResponseWriter, r *http.Request) error {
	pkg, _, err := s.packageOf(r)
	if err != nil {
		return err
	}
	a, err := s.attachedOf(r, pkg)
	if err != nil {
		return err
	}
	if err := s.checkToken(w, r, func(s *session) *string { return &s.AttachDeleteToken }); err != nil {
		return err
	}
	if err := s.svc.DeleteAttachedFile(r.Context(), a); err != nil {
		return err
	}
	return redirect(w, r, fmt.Sprintf("/package?id=%d", pkg.ID))
}
