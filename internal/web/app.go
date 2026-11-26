package web

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/KLab/emlauncher-go/internal/db"
	"github.com/KLab/emlauncher-go/internal/pkgfile"
	"github.com/KLab/emlauncher-go/internal/service"
)

type ctxApp struct{}

// appOf はリクエストの id のアプリ. requireAppOwner で読んだものがあればそれを使う.
func (s *Server) appOf(r *http.Request) (db.Application, error) {
	if app, ok := r.Context().Value(ctxApp{}).(db.Application); ok {
		return app, nil
	}
	id := formOf(r).intParam("id", 0)
	logAttr(r, "app_id", id)
	app, err := s.svc.GetApp(r.Context(), id)
	if errors.Is(err, service.ErrNotFound) {
		return app, errNotFound
	}
	return app, err
}

// requireAppOwner はリクエストの id のアプリのオーナーでなければ Permission Denied にする.
func (s *Server) requireAppOwner(next http.Handler) http.Handler {
	return s.page(func(w http.ResponseWriter, r *http.Request) error {
		app, err := s.appOf(r)
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
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxApp{}, app)))
		return nil
	})
}

// appPage はアプリの画面 (見出しと情報パネル) に共通のデータ.
type appPage struct {
	App *appView
}

func (s *Server) appPage(r *http.Request) (db.Application, appPage, error) {
	app, err := s.appOf(r)
	if err != nil {
		return app, appPage{}, err
	}
	v, err := s.appView(r.Context(), app, loginUser(r))
	return app, appPage{App: v}, err
}

func (s *Server) appNew(w http.ResponseWriter, r *http.Request) error {
	return s.renderOK(w, r, "app/new", nil)
}

var dataURL = regexp.MustCompile(`^data:[^;]+;base64,(.+)$`)

// iconData は画面から送られたアイコン (data URL) の画像. 空なら nil.
func iconData(v string) ([]byte, error) {
	if v == "" {
		return nil, nil
	}
	m := dataURL.FindStringSubmatch(v)
	if m == nil {
		return nil, badRequest("invalid icon-data")
	}
	b, err := base64.StdEncoding.DecodeString(m[1])
	if err != nil {
		return nil, badRequest("invalid icon-data")
	}
	return b, nil
}

func (s *Server) appCreate(w http.ResponseWriter, r *http.Request) error {
	f := formOf(r)
	icon, err := iconData(f.Get("icon-data"))
	if err != nil {
		return err
	}
	if f.Get("title") == "" || icon == nil {
		return errBadRequest
	}
	id, err := s.svc.CreateApp(r.Context(), loginUser(r), service.AppInfo{
		Title: f.Get("title"), Icon: icon, Description: f.optional("description"), Repository: f.optional("repository"),
	})
	if err != nil {
		return err
	}
	logAttr(r, "app_id", id)
	return redirect(w, r, fmt.Sprintf("/app?id=%d", id))
}

// appIndex はアプリのパッケージの一覧. pf (android / ios / other / all) と tags (タグの id を空白区切り)
// で絞り込む. pf が無ければ端末に合わせる.
func (s *Server) appIndex(w http.ResponseWriter, r *http.Request) error {
	const perPage = 20
	ctx := r.Context()
	f := formOf(r)
	app, ap, err := s.appPage(r)
	if err != nil {
		return err
	}
	platforms := map[string]string{"android": "Android", "ios": "iOS", "other": "unknown", "all": ""}
	pf := f.Get("pf")
	if _, ok := platforms[pf]; !ok {
		switch {
		case isAndroid(r):
			pf = "android"
		case isIOS(r):
			pf = "ios"
		default:
			pf = "all"
		}
	}
	var activeTags []int64
	for _, t := range strings.Fields(f.Get("tags")) {
		if id, err := strconv.ParseInt(t, 10, 64); err == nil && !slices.Contains(activeTags, id) {
			activeTags = append(activeTags, id)
		}
	}
	cur := max(1, int(f.intParam("page", 1)))
	pkgs, err := s.svc.ListPackages(ctx, service.PackageQuery{
		AppID: app.ID, Platform: platforms[pf], TagIDs: activeTags, Limit: perPage + 1, Offset: (cur - 1) * perPage,
	})
	if err != nil {
		return err
	}
	hasNext := len(pkgs) > perPage
	if hasNext {
		pkgs = pkgs[:perPage]
	}
	views, err := s.pkgViews(r, app.ID, pkgs)
	if err != nil {
		return err
	}

	appTags, err := s.svc.Queries().ListTagsByApp(ctx, app.ID)
	if err != nil {
		return err
	}
	var tags []tagView
	for _, t := range appTags {
		tags = append(tags, tagView{ID: t.ID, Name: t.Name, Active: slices.Contains(activeTags, t.ID)})
	}

	commentCount, err := s.svc.CountComments(ctx, app.ID)
	if err != nil {
		return err
	}
	topComments, err := s.commentViews(r, app.ID, 2, 0, commentCount)
	if err != nil {
		return err
	}

	return s.renderOK(w, r, "app/index", struct {
		appPage
		Pf           string
		FilterOpen   bool
		Tags         []tagView
		Packages     []*pkgView
		CurrentPage  int
		HasNext      bool
		PrevURL      string
		NextURL      string
		CommentCount int64
		TopComments  []commentView
	}{
		appPage: ap, Pf: pf, FilterOpen: truthy(f.Get("filter_open")), Tags: tags, Packages: views,
		CurrentPage: cur, HasNext: hasNext, PrevURL: withPage(r, cur-1), NextURL: withPage(r, cur+1),
		CommentCount: commentCount, TopComments: topComments,
	})
}

// pkgViews はパッケージの一覧の表示を作る.
func (s *Server) pkgViews(r *http.Request, appID int64, pkgs []db.Package) ([]*pkgView, error) {
	details, err := s.svc.Details(r.Context(), pkgs...)
	if err != nil {
		return nil, err
	}
	installed, err := s.svc.InstalledDates(r.Context(), appID, loginUser(r))
	if err != nil {
		return nil, err
	}
	views := make([]*pkgView, len(details))
	for i, d := range details {
		views[i] = s.pkgView(d, installed)
	}
	return views, nil
}

type commentView struct {
	Number   int64
	Message  string
	Created  string
	PageLink string   // コメントのあるページへのリンク
	Package  *pkgView // 対象のパッケージ (無ければ nil)
}

// commentViews はコメントを新しい順に読む. total はコメントの総数 (ページの計算に使う).
func (s *Server) commentViews(r *http.Request, appID int64, limit, offset int, total int64) ([]commentView, error) {
	const perPage = 20
	comments, err := s.svc.ListComments(r.Context(), appID, limit, offset)
	if err != nil {
		return nil, err
	}
	var ids []int64
	for _, c := range comments {
		if c.PackageID.Valid {
			ids = append(ids, c.PackageID.Int64)
		}
	}
	pkgs, err := s.svc.PackagesByID(r.Context(), ids)
	if err != nil {
		return nil, err
	}
	views := make([]commentView, len(comments))
	for i, c := range comments {
		page := (total-c.Number)/perPage + 1
		views[i] = commentView{
			Number:   c.Number,
			Message:  c.Message,
			Created:  c.Created.In(s.cfg.Location).Format("2006-01-02 15:04"),
			PageLink: fmt.Sprintf("/app/comment?id=%d&page=%d#comment-%d", appID, page, c.Number),
		}
		if p, ok := pkgs[c.PackageID.Int64]; ok && c.PackageID.Valid {
			views[i].Package = s.pkgView(service.PackageDetail{Package: p}, nil)
		}
	}
	return views, nil
}

func (s *Server) appComment(w http.ResponseWriter, r *http.Request) error {
	const perPage = 20
	ctx := r.Context()
	app, ap, err := s.appPage(r)
	if err != nil {
		return err
	}
	count, err := s.svc.CountComments(ctx, app.ID)
	if err != nil {
		return err
	}
	cur := int(formOf(r).intParam("page", 1))
	maxPage := int((count + perPage - 1) / perPage)
	offset := max(0, min(cur, maxPage)-1) * perPage
	comments, err := s.commentViews(r, app.ID, perPage, offset, count)
	if err != nil {
		return err
	}
	installed, err := s.svc.InstalledPackages(ctx, app.ID, loginUser(r))
	if err != nil {
		return err
	}
	var installedViews []*pkgView
	for _, p := range installed {
		installedViews = append(installedViews, s.pkgView(service.PackageDetail{Package: p}, nil))
	}
	return s.renderOK(w, r, "app/comment", struct {
		appPage
		CommentCount      int64
		Comments          []commentView
		InstalledPackages []*pkgView
		Paging            pagingView
	}{ap, count, comments, installedViews, paging(cur, maxPage, func(n int) string { return withPage(r, n) })})
}

func (s *Server) appCommentPost(w http.ResponseWriter, r *http.Request) error {
	f := formOf(r)
	app, err := s.appOf(r)
	if err != nil {
		return err
	}
	_, err = s.svc.PostComment(r.Context(), app.ID, loginUser(r), f.intParam("package_id", 0), f.Get("message"), baseURL(r))
	if err != nil {
		return err
	}
	return redirect(w, r, fmt.Sprintf("/app/comment?id=%d", app.ID))
}

func (s *Server) appUpload(w http.ResponseWriter, r *http.Request) error {
	app, ap, err := s.appPage(r)
	if err != nil {
		return err
	}
	tags, err := s.svc.Queries().ListTagsByApp(r.Context(), app.ID)
	if err != nil {
		return err
	}
	var tv []tagView
	for _, t := range tags {
		tv = append(tv, tagView{ID: t.ID, Name: t.Name})
	}
	return s.renderOK(w, r, "app/upload", struct {
		appPage
		Tags []tagView
	}{ap, tv})
}

var attachedFileParam = regexp.MustCompile(`^attached_files\[(\d+)\]\[(file_name|temp_name|type)\]$`)

// appUploadPost は画面からのアップロードの 2 段階目 (ajax で一時ファイルに置いたものを登録する).
func (s *Server) appUploadPost(w http.ResponseWriter, r *http.Request) error {
	f := formOf(r)
	app, err := s.appOf(r)
	if err != nil {
		return err
	}
	if f.Get("temp_name") == "" || f.Get("title") == "" {
		return errBadRequest
	}
	su := service.StagedUpload{
		FileName:   f.Get("file_name"),
		TempName:   f.Get("temp_name"),
		Platform:   pkgfile.Platform(f.Get("platform")),
		Identifier: f.optional("identifier"),
	}
	attached := map[int]*service.StagedAttachment{}
	for k, v := range f.Values {
		m := attachedFileParam.FindStringSubmatch(k)
		if m == nil || len(v) == 0 {
			continue
		}
		i, _ := strconv.Atoi(m[1])
		if attached[i] == nil {
			attached[i] = &service.StagedAttachment{}
		}
		switch m[2] {
		case "file_name":
			attached[i].FileName = v[0]
		case "temp_name":
			attached[i].TempName = v[0]
		case "type":
			attached[i].Type = v[0]
		}
	}
	for i := 0; i < len(attached); i++ {
		a, ok := attached[i]
		if !ok {
			return badRequest("invalid attached_files")
		}
		su.Attached = append(su.Attached, *a)
	}
	pkg, err := s.svc.CommitUpload(r.Context(), app.ID, su, service.NewPackage{
		Title:       f.Get("title"),
		Description: f.optional("description"),
		Tags:        f.Values["tags[]"],
		Protect:     truthy(f.Get("protect")),
		Notify:      truthy(f.Get("notify")),
		BaseURL:     baseURL(r),
	})
	if err != nil {
		return err
	}
	logPackage(r, pkg)
	return redirect(w, r, fmt.Sprintf("/package?id=%d", pkg.ID))
}

func (s *Server) appPreferences(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	app, ap, err := s.appPage(r)
	if err != nil {
		return err
	}
	tags, err := s.svc.Queries().ListTagsByApp(ctx, app.ID)
	if err != nil {
		return err
	}
	unused, err := s.svc.Queries().ListUnusedTagIDsByApp(ctx, app.ID)
	if err != nil {
		return err
	}
	var tv []tagView
	for _, t := range tags {
		tv = append(tv, tagView{ID: t.ID, Name: t.Name, Unused: slices.Contains(unused, t.ID)})
	}
	var others []string
	for _, o := range ap.App.Owners {
		if o != loginUser(r) {
			others = append(others, o)
		}
	}
	return s.renderOK(w, r, "app/preferences", struct {
		appPage
		Tags        []tagView
		HasUnused   bool
		OtherOwners []string
	}{ap, tv, len(unused) > 0, others})
}

func (s *Server) appPreferencesUpdate(w http.ResponseWriter, r *http.Request) error {
	f := formOf(r)
	app, err := s.appOf(r)
	if err != nil {
		return err
	}
	icon, err := iconData(f.Get("icon-data"))
	if err != nil {
		return err
	}
	if f.Get("title") == "" {
		return errBadRequest
	}
	err = s.svc.UpdateApp(r.Context(), app.ID, service.AppInfo{
		Title: f.Get("title"), Icon: icon, Description: f.optional("description"), Repository: f.optional("repository"),
	})
	if err != nil {
		return err
	}
	return redirect(w, r, fmt.Sprintf("/app/preferences?id=%d#edit-info", app.ID))
}

func (s *Server) appRefreshAPIKey(w http.ResponseWriter, r *http.Request) error {
	app, err := s.appOf(r)
	if err != nil {
		return err
	}
	if err := s.svc.RefreshAPIKey(r.Context(), app.ID, formOf(r).post.Get("api-key")); err != nil {
		return err
	}
	return redirect(w, r, fmt.Sprintf("/app/preferences?id=%d#refresh-apikey", app.ID))
}

func (s *Server) appDeleteTags(w http.ResponseWriter, r *http.Request) error {
	app, err := s.appOf(r)
	if err != nil {
		return err
	}
	var ids []int64
	for _, v := range formOf(r).Values["tags[]"] {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			ids = append(ids, id)
		}
	}
	if err := s.svc.DeleteTags(r.Context(), app.ID, ids); err != nil {
		return err
	}
	return redirect(w, r, fmt.Sprintf("/app/preferences?id=%d#delete-tags", app.ID))
}

func (s *Server) appUpdateOwners(w http.ResponseWriter, r *http.Request) error {
	app, err := s.appOf(r)
	if err != nil {
		return err
	}
	// オーナーは POST の本文で送られたものだけ受け付ける (別のサイトからの GET のリンクで変えさせない).
	// 画面のフォームは追加用の欄 owners[] を必ず送る
	owners, ok := formOf(r).post["owners[]"]
	if !ok {
		return errBadRequest
	}
	if err := s.svc.SetOwners(r.Context(), app.ID, loginUser(r), owners); err != nil {
		return err
	}
	return redirect(w, r, fmt.Sprintf("/app/preferences?id=%d#owners", app.ID))
}
