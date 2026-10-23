package web

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"text/template"
	"time"

	"github.com/KLabTools/emlauncher/go/internal/core"
	"github.com/KLabTools/emlauncher/go/internal/model"
	"github.com/KLabTools/emlauncher/go/internal/random"
)

const (
	downloadURLExpire = 60 * time.Minute
	installTokenTTL   = time.Hour
	guestPassTTL      = 7 * 24 * time.Hour

	sesPackageDeleteToken  = "package_delete_token"
	sesAttachedDeleteToken = "attached_delete_token"
)

// pkgContext はパッケージの画面 (?id=) の共通情報.
type pkgContext struct {
	*uiContext
	pkg      *model.Package
	app      *model.Application
	appView  *appView
	attached *model.AttachedFile
}

func (s *Server) pkgPage(h func(*pkgContext)) func(*uiContext) {
	return func(c *uiContext) {
		if pc, ok := s.loadPackage(c, c.form.int("id", 0)); ok {
			h(pc)
		}
	}
}

func (s *Server) pkgOwnerPage(h func(*pkgContext)) func(*uiContext) {
	return s.pkgPage(func(pc *pkgContext) {
		if !pc.appView.IsOwner {
			s.errorPage(pc.uiContext, http.StatusForbidden, "Permission Denied")
			return
		}
		h(pc)
	})
}

// attachedPage は attached_id の添付ファイルを読み込む (パッケージのものでなければ 404).
func (s *Server) attachedPage(h func(*pkgContext)) func(*pkgContext) {
	return func(pc *pkgContext) {
		af, err := model.GetAttachedFile(pc.r.Context(), s.Core.DB, pc.form.int("attached_id", 0))
		if err != nil {
			s.serverError(pc.uiContext, err)
			return
		}
		if af == nil || af.PackageID != pc.pkg.ID {
			s.errorPage(pc.uiContext, http.StatusNotFound, "Not Found")
			return
		}
		pc.attached = af
		logAttr(pc.r, "attached_id", af.ID)
		h(pc)
	}
}

func (s *Server) loadPackage(c *uiContext, id int64) (*pkgContext, bool) {
	ctx := c.r.Context()
	pkg, err := model.GetPackage(ctx, s.Core.DB, id)
	if err != nil {
		s.serverError(c, err)
		return nil, false
	}
	if pkg == nil {
		s.errorPage(c, http.StatusNotFound, "Not Found")
		return nil, false
	}
	app, err := model.GetApplication(ctx, s.Core.DB, pkg.AppID)
	if err != nil || app == nil {
		s.serverError(c, fmt.Errorf("application of package %d: %v", pkg.ID, err))
		return nil, false
	}
	v, err := s.appView(ctx, app, c.user)
	if err != nil {
		s.serverError(c, err)
		return nil, false
	}
	logAttr(c.r, "app_id", app.ID)
	logAttr(c.r, "pkg_id", pkg.ID)
	return &pkgContext{uiContext: c, pkg: pkg, app: app, appView: v}, true
}

// view はテンプレートに渡すデータ (App, Package と追加の項目).
func (s *Server) pkgData(pc *pkgContext, extra map[string]any) (map[string]any, error) {
	ctx := pc.r.Context()
	dates, err := model.PackageInstalledDates(ctx, s.Core.DB, pc.user, pc.app.ID)
	if err != nil {
		return nil, err
	}
	var at *time.Time
	if t, ok := dates[pc.pkg.ID]; ok {
		at = &t
	}
	pv, err := s.pkgView(ctx, pc.pkg, at)
	if err != nil {
		return nil, err
	}
	d := map[string]any{"App": pc.appView, "Package": pv}
	for k, v := range extra {
		d[k] = v
	}
	return d, nil
}

func (s *Server) renderPkg(pc *pkgContext, name string, extra map[string]any) {
	d, err := s.pkgData(pc, extra)
	if err != nil {
		s.serverError(pc.uiContext, err)
		return
	}
	s.render(pc.uiContext, http.StatusOK, name, d)
}

type guestPassView struct {
	ID           int64
	PackageID    int64
	AppID        int64
	AppTitle     string
	IconURL      string
	Expired      string
	InstallCount int
}

type attachedView struct {
	ID   int64
	Name string
	Size string
}

func (s *Server) pkgIndex(pc *pkgContext) {
	ctx := pc.r.Context()
	users, err := model.InstallUsers(ctx, s.Core.DB, pc.pkg.ID)
	if err != nil {
		s.serverError(pc.uiContext, err)
		return
	}
	var gps []guestPassView
	if pc.appView.IsOwner {
		list, err := model.ActiveGuestPassesByPackage(ctx, s.Core.DB, pc.pkg.ID, s.Core.Clock.Now())
		if err != nil {
			s.serverError(pc.uiContext, err)
			return
		}
		for _, g := range list {
			n, err := model.GuestPassInstallCount(ctx, s.Core.DB, g.ID)
			if err != nil {
				s.serverError(pc.uiContext, err)
				return
			}
			gps = append(gps, guestPassView{ID: g.ID, PackageID: g.PackageID, Expired: s.Core.Clock.Format(g.Expired), InstallCount: n})
		}
	}
	afs, err := model.AttachedFilesByPackage(ctx, s.Core.DB, pc.pkg.ID)
	if err != nil {
		s.serverError(pc.uiContext, err)
		return
	}
	var attached []attachedView
	for _, a := range afs {
		attached = append(attached, attachedView{ID: a.ID, Name: a.OriginalFileName, Size: formatSize(a.FileSize)})
	}
	s.renderPkg(pc, "package/index", map[string]any{"InstallUsers": users, "GuestPasses": gps, "Attached": attached})
}

// インストール

// installURL はインストール (ダウンロード) 先の URL.
//   - iOS 端末で iOS のパッケージ: itms-services (plist は一時トークン付きの URL で取得させる)
//   - Android 端末で AAB: 生成した universal APK (なければ AAB そのもの)
//   - それ以外: パッケージそのもの
func (s *Server) installURL(ctx context.Context, r *http.Request, pkg *model.Package) (string, error) {
	ua := r.UserAgent()
	if pkg.Platform == "iOS" && isIOS(ua) {
		token, err := s.Core.CreatePlistToken(pkg.ID)
		if err != nil {
			return "", err
		}
		scheme := ""
		if s.Config.EnableHTTPS {
			scheme = "https"
		}
		plist := absURL(r, fmt.Sprintf("/package/install_plist?id=%d&t=%s", pkg.ID, token), scheme)
		return "itms-services://?action=download-manifest&url=" + url.QueryEscape(plist), nil
	}
	key, name := pkg.FileKey(), pkg.OriginalFileName.String
	if isAndroid(ua) {
		var err error
		if key, name, err = core.AndroidDownloadTarget(ctx, s.Core.DB, pkg); err != nil {
			return "", err
		}
	}
	return s.downloadURL(ctx, r, key, name)
}

// downloadURL は期限付きのダウンロード URL (LocalFile なら相対パスなので絶対 URL にする).
func (s *Server) downloadURL(ctx context.Context, r *http.Request, key, name string) (string, error) {
	u, err := s.Core.Storage.DownloadURL(ctx, key, downloadURLExpire, name)
	if err != nil {
		return "", err
	}
	if len(u) > 0 && u[0] == '/' {
		u = absURL(r, u, "")
	}
	return u, nil
}

func (s *Server) pkgInstall(c *uiContext) {
	ctx := c.r.Context()
	var pc *pkgContext
	var mailAddr string
	if token := c.form.ptr("token"); token != nil {
		// トークン付きのインストールリンクはトークンだけで認証する
		t, pkgID, err := s.Core.LookupInstallToken(*token)
		if err != nil {
			s.serverError(c, err)
			return
		}
		if t == nil {
			s.Log.Info("invalid install token", "token", *token)
			http.Error(c.w, "invalid token", http.StatusForbidden)
			return
		}
		c.user = t.Mail
		var ok bool
		if pc, ok = s.loadPackage(c, pkgID); !ok {
			return
		}
		mailAddr = t.Mail
	} else {
		if c.user == "" {
			s.redirectToLogin(c)
			return
		}
		var ok bool
		if pc, ok = s.loadPackage(c, c.form.int("id", 0)); !ok {
			return
		}
		mailAddr = c.user
	}
	u, err := s.installURL(ctx, c.r, pc.pkg)
	if err != nil {
		s.serverError(c, err)
		return
	}
	if err := s.Core.LogInstall(ctx, pc.pkg, mailAddr, c.r.UserAgent()); err != nil {
		s.serverError(c, err)
		return
	}
	logAttr(c.r, "user", mailAddr)
	logAttr(c.r, "platform", pc.pkg.Platform)
	redirect(c.w, c.r, u)
}

func (s *Server) pkgGuestPassInstall(c *uiContext) {
	ctx := c.r.Context()
	token := c.form.ptr("token")
	if token == nil {
		// トークンが無ければ通常のログインが必要 (PHP 版と同じ)
		if c.user == "" {
			s.redirectToLogin(c)
			return
		}
		s.errorPage(c, http.StatusNotFound, "Not Found")
		return
	}
	gp, err := model.GetGuestPassByToken(ctx, s.Core.DB, *token)
	if err != nil {
		s.serverError(c, err)
		return
	}
	if gp == nil {
		s.errorPage(c, http.StatusNotFound, "Not Found")
		return
	}
	if gp.Expired.Before(s.Core.Clock.Now()) {
		s.Log.Info("expired guestpass", "guestpass_id", gp.ID)
		s.errorPage(c, http.StatusForbidden, "Invalid token")
		return
	}
	c.user = gp.Mail
	pc, ok := s.loadPackage(c, gp.PackageID)
	if !ok {
		return
	}
	u, err := s.installURL(ctx, c.r, pc.pkg)
	if err != nil {
		s.serverError(c, err)
		return
	}
	if err := model.LogGuestPassInstall(ctx, s.Core.DB, gp.ID, c.r.UserAgent(), remoteHost(c.r), s.Core.Clock.Now()); err != nil {
		s.serverError(c, err)
		return
	}
	logAttr(c.r, "guestpass_id", gp.ID)
	redirect(c.w, c.r, u)
}

// remoteHost はゲストパスのログに残す IP (PHP 版と同じく X-Forwarded-For をそのまま使う).
func remoteHost(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	return remoteAddr(r)
}

var plistTmpl = template.Must(template.New("plist").Funcs(template.FuncMap{"xml": xmlEscape}).Parse(`<?xml version="1.0" encoding="UTF-8"?>
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
          <string>{{xml .PackageTitle}}</string>
          <key>title</key>
          <string>{{xml .AppTitle}}</string>
        </dict>
      </dict>
    </array>
  </dict>
</plist>
`))

func xmlEscape(s string) string {
	var b bytes.Buffer
	template.HTMLEscape(&b, []byte(s))
	return b.String()
}

// pkgInstallPlist は iOS のインストーラが取りに来る manifest. セッションは使えないので一時トークンで認証する.
func (s *Server) pkgInstallPlist(c *uiContext) {
	ctx := c.r.Context()
	pkg, err := model.GetPackage(ctx, s.Core.DB, c.form.int("id", 0))
	if err != nil {
		s.serverError(c, err)
		return
	}
	if pkg == nil {
		s.errorPage(c, http.StatusNotFound, "Not Found")
		return
	}
	if !s.Core.CheckPlistToken(c.form.str("t"), pkg.ID) {
		s.errorPage(c, http.StatusForbidden, "Permission Denied")
		return
	}
	app, err := model.GetApplication(ctx, s.Core.DB, pkg.AppID)
	if err != nil || app == nil {
		s.serverError(c, fmt.Errorf("application of package %d: %v", pkg.ID, err))
		return
	}
	ipaURL, err := s.downloadURL(ctx, c.r, pkg.FileKey(), pkg.OriginalFileName.String)
	if err != nil {
		s.serverError(c, err)
		return
	}
	imageURL := s.iconURL(app)
	if len(imageURL) > 0 && imageURL[0] == '/' {
		imageURL = absURL(c.r, imageURL, "")
	}
	var buf bytes.Buffer
	plistTmpl.Execute(&buf, map[string]string{
		"IPAURL": ipaURL, "ImageURL": imageURL, "BundleID": pkg.Identifier.String,
		"PackageTitle": pkg.Title, "AppTitle": app.Title,
	})
	c.w.Header().Set("Content-Type", "text/xml")
	c.w.Write(buf.Bytes())
}

func (s *Server) pkgCreateToken(pc *pkgContext) {
	token, expire, err := s.Core.CreateInstallToken(pc.user, pc.pkg.ID, installTokenTTL)
	if err != nil {
		s.serverError(pc.uiContext, err)
		return
	}
	path := "/package/install?token=" + token
	s.renderPkg(pc, "package/create_token", map[string]any{
		"URL": absURL(pc.r, path, ""), "QRPath": path, "Expire": s.Core.Clock.Format(expire), "GeneratedBy": pc.user,
	})
}

// ゲストパス (オーナーのみ)

func (s *Server) pkgCreateGuestPass(pc *pkgContext) {
	if !pc.appView.IsOwner {
		s.errorPage(pc.uiContext, http.StatusNotFound, "Not Found")
		return
	}
	now := s.Core.Clock.Now()
	gp := &model.GuestPass{
		AppID: pc.app.ID, PackageID: pc.pkg.ID, Mail: pc.user, Token: random.String(32),
		Created: now, Expired: now.Add(guestPassTTL),
	}
	if err := model.InsertGuestPass(pc.r.Context(), s.Core.DB, gp); err != nil {
		s.serverError(pc.uiContext, err)
		return
	}
	redirect(pc.w, pc.r, fmt.Sprintf("/package/guestpass?id=%d&guestpass_id=%d", pc.pkg.ID, gp.ID))
}

// ownGuestPass は guestpass_id のゲストパス. オーナーでなければ nil.
func (s *Server) ownGuestPass(pc *pkgContext) (*model.GuestPass, error) {
	gp, err := model.GetGuestPass(pc.r.Context(), s.Core.DB, pc.form.int("guestpass_id", 0))
	if err != nil || gp == nil || !pc.appView.IsOwner || gp.PackageID != pc.pkg.ID {
		return nil, err
	}
	return gp, nil
}

func (s *Server) pkgGuestPass(pc *pkgContext) {
	gp, err := s.ownGuestPass(pc)
	if err != nil {
		s.serverError(pc.uiContext, err)
		return
	}
	if gp == nil {
		s.errorPage(pc.uiContext, http.StatusForbidden, "Forbidden")
		return
	}
	path := "/package/guestpass_install?token=" + gp.Token
	s.renderPkg(pc, "package/guestpass", map[string]any{
		"URL": absURL(pc.r, path, ""), "QRPath": path, "Expire": s.Core.Clock.Format(gp.Expired), "GeneratedBy": gp.Mail,
	})
}

func (s *Server) pkgExpireGuestPass(pc *pkgContext) {
	gp, err := s.ownGuestPass(pc)
	if err != nil {
		s.serverError(pc.uiContext, err)
		return
	}
	if gp == nil {
		s.errorPage(pc.uiContext, http.StatusNotFound, "Not Found")
		return
	}
	if err := gp.Expire(pc.r.Context(), s.Core.DB, s.Core.Clock.Now()); err != nil {
		s.serverError(pc.uiContext, err)
		return
	}
	redirect(pc.w, pc.r, fmt.Sprintf("/package/?id=%d", pc.pkg.ID))
}

// 編集・削除・添付 (オーナーのみ)

func (s *Server) pkgEdit(pc *pkgContext) {
	ctx := pc.r.Context()
	appTags, err := model.TagsByApp(ctx, s.Core.DB, pc.app.ID)
	if err != nil {
		s.serverError(pc.uiContext, err)
		return
	}
	pkgTags, err := model.TagsByPackage(ctx, s.Core.DB, pc.pkg.ID)
	if err != nil {
		s.serverError(pc.uiContext, err)
		return
	}
	var tags []tagView
	for _, t := range appTags {
		tags = append(tags, tagView{ID: t.ID, Name: t.Name,
			Active: slices.ContainsFunc(pkgTags, func(p model.Tag) bool { return p.ID == t.ID })})
	}
	s.renderPkg(pc, "package/edit", map[string]any{"Tags": tags})
}

func (s *Server) pkgEditCommit(pc *pkgContext) {
	title := pc.form.str("title")
	if title == "" {
		badRequest(pc.uiContext)
		return
	}
	desc := pc.form.str("description")
	protect := truthy(pc.form.str("protect"))
	tags := pc.form.list("tags")
	_, err := s.Core.EditPackage(pc.r.Context(), pc.app.ID, pc.pkg.ID, core.PackageEdit{
		Title: &title, Description: &desc, Protect: &protect, Tags: &tags,
	})
	if err != nil {
		s.serverError(pc.uiContext, err)
		return
	}
	redirect(pc.w, pc.r, fmt.Sprintf("/package?id=%d", pc.pkg.ID))
}

// checkDeleteToken は削除確認画面で発行したトークン (CSRF 対策) を確認して消す.
func checkDeleteToken(c *uiContext, key string) bool {
	token := ""
	if v, ok := c.form.values["token"]; ok && c.r.Method == http.MethodPost && len(v) > 0 {
		token = v[0]
	}
	want := c.sess.Get(key)
	if token == "" || token != want {
		return false
	}
	c.sess.Delete(key)
	return true
}

func (s *Server) pkgDeleteConfirm(pc *pkgContext) {
	token := random.String(32)
	pc.sess.Set(sesPackageDeleteToken, token)
	s.renderPkg(pc, "package/delete_confirm", map[string]any{"Token": token})
}

func (s *Server) pkgDelete(pc *pkgContext) {
	if !checkDeleteToken(pc.uiContext, sesPackageDeleteToken) {
		s.errorPage(pc.uiContext, http.StatusBadRequest, "Bad Request")
		return
	}
	if _, _, err := s.Core.DeletePackage(pc.r.Context(), pc.app.ID, pc.pkg.ID); err != nil {
		s.serverError(pc.uiContext, err)
		return
	}
	redirect(pc.w, pc.r, fmt.Sprintf("/app?id=%d", pc.app.ID))
}

func (s *Server) pkgAttach(pc *pkgContext) {
	file := pc.form.file("file")
	if file == nil {
		badRequest(pc.uiContext)
		return
	}
	if _, err := s.Core.AttachFile(pc.r.Context(), pc.pkg, *file); err != nil {
		s.serverError(pc.uiContext, err)
		return
	}
	redirect(pc.w, pc.r, fmt.Sprintf("/package?id=%d", pc.pkg.ID))
}

func (s *Server) pkgAttachDownload(pc *pkgContext) {
	u, err := s.downloadURL(pc.r.Context(), pc.r, pc.attached.FileKey(), pc.attached.OriginalFileName)
	if err != nil {
		s.serverError(pc.uiContext, err)
		return
	}
	redirect(pc.w, pc.r, u)
}

func (s *Server) pkgAttachDeleteConfirm(pc *pkgContext) {
	token := random.String(32)
	pc.sess.Set(sesAttachedDeleteToken, token)
	s.renderPkg(pc, "package/attach_delete_confirm", map[string]any{"Token": token, "Attached": pc.attached})
}

func (s *Server) pkgAttachDelete(pc *pkgContext) {
	if !checkDeleteToken(pc.uiContext, sesAttachedDeleteToken) {
		s.errorPage(pc.uiContext, http.StatusBadRequest, "Bad Request")
		return
	}
	if err := s.Core.DetachFile(pc.r.Context(), pc.attached); err != nil {
		s.serverError(pc.uiContext, err)
		return
	}
	redirect(pc.w, pc.r, fmt.Sprintf("/package?id=%d", pc.pkg.ID))
}
