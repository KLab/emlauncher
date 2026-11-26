package web

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"

	"github.com/KLab/emlauncher-go/internal/db"
)

const (
	qrDefaultSize = 150
	qrMaxSize     = 300
	qrMargin      = 10
)

// qrCode はサイトの中のパス q を指す QR コードの PNG を返す. s は QR コードの大きさ (最大 300).
// ログイン不要.
func (s *Server) qrCode(w http.ResponseWriter, r *http.Request) {
	f := formOf(r)
	size := int(f.intParam("s", qrDefaultSize))
	if size <= 0 {
		size = qrDefaultSize
	}
	size = min(size, qrMaxSize)
	hints := map[gozxing.EncodeHintType]any{
		gozxing.EncodeHintType_ERROR_CORRECTION: "H",
		gozxing.EncodeHintType_MARGIN:           0,
	}
	code, err := qrcode.NewQRCodeWriter().Encode(baseURL(r)+f.Get("q"), gozxing.BarcodeFormat_QR_CODE, size, size, hints)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	b := code.Bounds()
	img := image.NewRGBA(image.Rect(0, 0, b.Dx()+2*qrMargin, b.Dy()+2*qrMargin))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(img, b.Add(image.Pt(qrMargin, qrMargin)), code, b.Min, draw.Src)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	buf.WriteTo(w)
}

// doc は API などのドキュメントの画面.
func (s *Server) doc(w http.ResponseWriter, r *http.Request) error {
	name := chi.URLParam(r, "page")
	if name == "keystore" {
		info, err := s.bundletool.KeystoreInfo(r.Context())
		if err != nil {
			s.log.Error("keystore info", "err", err)
			info = err.Error()
		}
		return s.renderOK(w, r, "doc/keystore", info)
	}
	if _, ok := s.templates["doc/"+name]; !ok {
		notFoundText(w, r)
		return nil
	}
	return s.renderOK(w, r, "doc/"+name, nil)
}

// top はアプリの一覧 (更新の新しい順).
func (s *Server) top(w http.ResponseWriter, r *http.Request) error {
	const perPage = 20
	ctx := r.Context()
	q := s.svc.Queries()
	count, err := q.CountApplications(ctx)
	if err != nil {
		return err
	}
	cur := int(formOf(r).intParam("page", 1))
	maxPage := int((count + perPage - 1) / perPage)
	offset := (max(0, min(cur, maxPage)-1)) * perPage
	apps, err := q.ListApplicationsByDateToSort(ctx, db.ListApplicationsByDateToSortParams{Limit: perPage, Offset: int32(offset)})
	if err != nil {
		return err
	}
	items, err := s.appListItems(r, apps)
	if err != nil {
		return err
	}
	return s.renderOK(w, r, "top/index", struct {
		Apps   []appListItem
		Paging pagingView
	}{items, paging(cur, maxPage, func(n int) string { return baseURL(r) + "/?page=" + strconv.Itoa(n) })})
}

// appListItems はアプリの一覧の行 (コメント数, インストールしたユーザ数, 新着の表示) を作る.
func (s *Server) appListItems(r *http.Request, apps []db.Application) ([]appListItem, error) {
	ctx := r.Context()
	q := s.svc.Queries()
	ids := make([]int64, len(apps))
	for i, a := range apps {
		ids[i] = a.ID
	}
	comments := map[int64]int64{}
	users := map[int64]int64{}
	if len(ids) > 0 {
		cs, err := q.CountCommentsByApps(ctx, ids)
		if err != nil {
			return nil, err
		}
		for _, c := range cs {
			comments[c.AppID] = c.Comments
		}
		us, err := q.CountAppInstallUsers(ctx, ids)
		if err != nil {
			return nil, err
		}
		for _, u := range us {
			users[u.AppID] = u.Users
		}
	}
	installed, err := s.installedApps(ctx, loginUser(r))
	if err != nil {
		return nil, err
	}
	items := make([]appListItem, len(apps))
	for i, a := range apps {
		items[i] = s.appListItem(a, lastInstalled(installed, a.ID))
		items[i].Comments = comments[a.ID]
		items[i].InstallUsers = users[a.ID]
	}
	return items, nil
}

func (s *Server) myappsOwn(w http.ResponseWriter, r *http.Request) error {
	apps, err := s.svc.Queries().ListApplicationsByOwner(r.Context(), loginUser(r))
	if err != nil {
		return err
	}
	items, err := s.appListItems(r, apps)
	if err != nil {
		return err
	}
	return s.renderOK(w, r, "myapps/own", struct{ Apps []appListItem }{items})
}

func (s *Server) myappsInstalled(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	ius, err := s.svc.InstalledApps(ctx, loginUser(r))
	if err != nil {
		return err
	}
	ids := make([]int64, len(ius))
	for i, iu := range ius {
		ids[i] = iu.AppID
	}
	apps := map[int64]db.Application{}
	if len(ids) > 0 {
		as, err := s.svc.Queries().GetApplicationsByIDs(ctx, ids)
		if err != nil {
			return err
		}
		for _, a := range as {
			apps[a.ID] = a
		}
	}
	var items []appListItem
	for _, iu := range ius {
		app, ok := apps[iu.AppID]
		if !ok {
			continue
		}
		it := s.appListItem(app, &iu.LastInstalled)
		it.Notify = iu.Notify != 0
		items = append(items, it)
	}
	return s.renderOK(w, r, "myapps/installed", struct{ Apps []appListItem }{items})
}

// myappsDelete はアプリをインストール済みの一覧から外す.
func (s *Server) myappsDelete(w http.ResponseWriter, r *http.Request) error {
	if err := s.svc.RemoveInstalledApp(r.Context(), formOf(r).intParam("id", 0), loginUser(r)); err != nil {
		return err
	}
	return redirect(w, r, "/myapps/installed")
}

// guestpassList はユーザがオーナーのアプリの有効なゲストパス.
func (s *Server) guestpassList(w http.ResponseWriter, r *http.Request) error {
	rows, err := s.svc.OwnedGuestPasses(r.Context(), loginUser(r))
	if err != nil {
		return err
	}
	type item struct {
		ID, AppID, PackageID int64
		AppTitle, IconURL    string
		Expired              string
		InstallCount         int64
	}
	var items []item
	for _, g := range rows {
		items = append(items, item{
			ID: g.ID, AppID: g.AppID, PackageID: g.PackageID, AppTitle: g.AppTitle,
			IconURL:      s.svc.IconURL(db.Application{IconKey: g.AppIconKey}),
			Expired:      s.svc.FormatTime(g.Expired),
			InstallCount: g.InstallCount,
		})
	}
	return s.renderOK(w, r, "guestpass/list", struct{ GuestPasses []item }{items})
}
