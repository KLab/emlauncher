package web

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"net/http"

	"github.com/skip2/go-qrcode"

	"emlauncher/internal/model"
)

// マイアプリ

type installedAppView struct {
	topAppView
	Notify bool
}

func (s *Server) myappsInstalled(c *uiContext) {
	ctx := c.r.Context()
	ias, err := model.InstalledApps(ctx, s.Core.DB, c.user)
	if err != nil {
		s.serverError(c, err)
		return
	}
	var ids []int64
	for _, ia := range ias {
		ids = append(ids, ia.AppID)
	}
	apps, err := model.GetApplications(ctx, s.Core.DB, ids)
	if err != nil {
		s.serverError(c, err)
		return
	}
	byID := map[int64]model.Application{}
	for _, a := range apps {
		byID[a.ID] = a
	}
	var ordered []model.Application
	var notify []bool
	for _, ia := range ias {
		if a, ok := byID[ia.AppID]; ok {
			ordered = append(ordered, a)
			notify = append(notify, ia.Notify)
		}
	}
	views, err := s.appListViews(c, ordered)
	if err != nil {
		s.serverError(c, err)
		return
	}
	var out []installedAppView
	for i, v := range views {
		out = append(out, installedAppView{v, notify[i]})
	}
	s.render(c, http.StatusOK, "myapps/installed", map[string]any{"Apps": out})
}

func (s *Server) myappsOwn(c *uiContext) {
	apps, err := model.OwnApplications(c.r.Context(), s.Core.DB, c.user)
	if err != nil {
		s.serverError(c, err)
		return
	}
	views, err := s.appListViews(c, apps)
	if err != nil {
		s.serverError(c, err)
		return
	}
	s.render(c, http.StatusOK, "myapps/own", map[string]any{"Apps": views})
}

// myappsDelete はインストール済み一覧から外す (install_log は残る).
func (s *Server) myappsDelete(c *uiContext) {
	ctx := c.r.Context()
	ia, err := model.GetInstalledApp(ctx, s.Core.DB, c.user, c.form.int("id", 0))
	if err != nil {
		s.serverError(c, err)
		return
	}
	if ia != nil {
		if err := ia.Delete(ctx, s.Core.DB); err != nil {
			s.serverError(c, err)
			return
		}
	}
	redirect(c.w, c.r, "/myapps/installed")
}

func (s *Server) guestpassList(c *uiContext) {
	ctx := c.r.Context()
	gps, err := model.ActiveGuestPassesByOwner(ctx, s.Core.DB, c.user, s.Core.Clock.Now())
	if err != nil {
		s.serverError(c, err)
		return
	}
	var views []guestPassView
	for _, g := range gps {
		app, err := model.GetApplication(ctx, s.Core.DB, g.AppID)
		if err != nil {
			s.serverError(c, err)
			return
		}
		n, err := model.GuestPassInstallCount(ctx, s.Core.DB, g.ID)
		if err != nil {
			s.serverError(c, err)
			return
		}
		v := guestPassView{ID: g.ID, PackageID: g.PackageID, AppID: g.AppID, Expired: s.Core.Clock.Format(g.Expired), InstallCount: n}
		if app != nil {
			v.AppTitle, v.IconURL = app.Title, s.iconURL(app)
		}
		views = append(views, v)
	}
	s.render(c, http.StatusOK, "guestpass/list", map[string]any{"GuestPasses": views})
}

// ドキュメント

func (s *Server) docPage(c *uiContext) {
	s.render(c, http.StatusOK, "doc/"+c.action, nil)
}

func (s *Server) docKeystore(c *uiContext) {
	info, err := s.Core.Bundletool.KeystoreInfo(c.r.Context())
	if err != nil {
		s.Log.Error("keystore", "error", err)
		info = err.Error()
	}
	s.render(c, http.StatusOK, "doc/keystore", info)
}

// QR コード (ログイン不要). q は "/app?id=1" のようなパスで, このサーバの URL にして QR コードにする.

const (
	qrDefaultSize = 150
	qrMaxSize     = 300
	qrMargin      = 10
)

func (s *Server) qrCode(c *uiContext) {
	size := int(c.form.int("s", qrDefaultSize))
	if size > qrMaxSize {
		size = qrMaxSize
	}
	if size <= 0 {
		size = qrDefaultSize
	}
	q, err := qrcode.New(absURL(c.r, c.form.str("q"), ""), qrcode.High)
	if err != nil {
		s.serverError(c, err)
		return
	}
	q.DisableBorder = true
	code := q.Image(size)
	img := image.NewRGBA(image.Rect(0, 0, size+2*qrMargin, size+2*qrMargin))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(img, code.Bounds().Add(image.Pt(qrMargin, qrMargin)), code, image.Point{}, draw.Src)
	var buf bytes.Buffer
	png.Encode(&buf, img)
	c.w.Header().Set("Content-Type", "image/png")
	c.w.Write(buf.Bytes())
}

// ajax (ログイン必須, JSON)

func (s *Server) ajaxNotificationSetting(c *uiContext) {
	ctx := c.r.Context()
	appID := c.form.int("id", 0)
	ia, err := model.GetInstalledApp(ctx, s.Core.DB, c.user, appID)
	if err != nil {
		writeJSON(c.w, http.StatusInternalServerError, map[string]string{"error": err.Error(), "exception": "Exception"})
		return
	}
	if ia == nil {
		writeJSON(c.w, http.StatusNotFound, map[string]string{"error": "installed application not found."})
		return
	}
	if err := ia.UpdateNotify(ctx, s.Core.DB, truthy(c.form.str("value"))); err != nil {
		writeJSON(c.w, http.StatusInternalServerError, map[string]string{"error": err.Error(), "exception": "Exception"})
		return
	}
	logAttr(c.r, "app_id", appID)
	writeJSON(c.w, http.StatusOK, map[string]any{
		"app_id":         fmt.Sprint(ia.AppID),
		"mail":           ia.Mail,
		"notify":         ia.Notify,
		"last_installed": s.Core.Clock.Format(ia.LastInstalled),
	})
}

func (s *Server) ajaxUploadPackageTemporary(c *uiContext) {
	file := c.form.file("file")
	if file == nil {
		writeJSON(c.w, http.StatusBadRequest, map[string]string{"error": "upload_file error: no file"})
		return
	}
	t, err := s.Core.UploadTemp(c.r.Context(), *file)
	if err != nil {
		s.Log.Error("upload temporary", "error", err)
		writeJSON(c.w, http.StatusInternalServerError, map[string]string{"error": err.Error(), "exception": "Exception"})
		return
	}
	writeJSON(c.w, http.StatusOK, t)
}
