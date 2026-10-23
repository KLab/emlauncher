package web

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/KLabTools/emlauncher/go/internal/core"
	"github.com/KLabTools/emlauncher/go/internal/model"
)

const linesInPage = 20

// appContext はアプリの画面 (?id=) の共通情報.
type appContext struct {
	*uiContext
	app  *model.Application
	view *appView
}

// appPage は id のアプリを読み込んでから h を呼ぶ.
func (s *Server) appPage(h func(*appContext)) func(*uiContext) {
	return func(c *uiContext) {
		ac, ok := s.loadApp(c)
		if ok {
			h(ac)
		}
	}
}

// appOwnerPage はオーナーだけが使える画面.
func (s *Server) appOwnerPage(h func(*appContext)) func(*uiContext) {
	return func(c *uiContext) {
		ac, ok := s.loadApp(c)
		if !ok {
			return
		}
		if !ac.view.IsOwner {
			s.errorPage(c, http.StatusForbidden, "Permission Denied")
			return
		}
		h(ac)
	}
}

func (s *Server) loadApp(c *uiContext) (*appContext, bool) {
	app, err := model.GetApplication(c.r.Context(), s.Core.DB, c.form.int("id", 0))
	if err != nil {
		s.serverError(c, err)
		return nil, false
	}
	if app == nil {
		s.errorPage(c, http.StatusNotFound, "Not Found")
		return nil, false
	}
	v, err := s.appView(c.r.Context(), app, c.user)
	if err != nil {
		s.serverError(c, err)
		return nil, false
	}
	logAttr(c.r, "app_id", app.ID)
	return &appContext{uiContext: c, app: app, view: v}, true
}

// トップ: アプリ一覧 (新着順)

type topAppView struct {
	ID           int64
	Title        string
	IconURL      string
	Comments     int
	InstallUsers int
	TimeLabel    string
	Time         string
	Badge        string
}

func (s *Server) topIndex(c *uiContext) {
	ctx := c.r.Context()
	count, err := model.CountApplications(ctx, s.Core.DB)
	if err != nil {
		s.serverError(c, err)
		return
	}
	maxPage := int(math.Ceil(float64(count) / linesInPage))
	cur := int(c.form.int("page", 1))
	offset := max(0, min(cur, maxPage)-1) * linesInPage
	apps, err := model.ApplicationsByDateToSort(ctx, s.Core.DB, offset, linesInPage)
	if err != nil {
		s.serverError(c, err)
		return
	}
	views, err := s.appListViews(c, apps)
	if err != nil {
		s.serverError(c, err)
		return
	}
	s.render(c, http.StatusOK, "top/index", map[string]any{
		"Apps":   views,
		"Paging": paging(absURL(c.r, "/", ""), cur, maxPage),
	})
}

func (s *Server) appListViews(c *uiContext, apps []model.Application) ([]topAppView, error) {
	ctx := c.r.Context()
	var ids []int64
	for _, a := range apps {
		ids = append(ids, a.ID)
	}
	comments, err := model.CommentCounts(ctx, s.Core.DB, ids)
	if err != nil {
		return nil, err
	}
	installed, err := s.installedApps(ctx, c.user)
	if err != nil {
		return nil, err
	}
	var views []topAppView
	for i := range apps {
		a := &apps[i]
		users, err := model.InstallUsersByApp(ctx, s.Core.DB, a.ID)
		if err != nil {
			return nil, err
		}
		v := topAppView{
			ID: a.ID, Title: a.Title, IconURL: s.iconURL(a), Comments: comments[a.ID], InstallUsers: len(users),
			TimeLabel: "created", Badge: s.updateBadge(a, installed),
		}
		t := a.Created
		if a.LastUpload.Valid {
			v.TimeLabel, t = "last uploaded", a.LastUpload.Time
		}
		v.Time = t.In(s.Core.Clock.Location()).Format("2006-01-02 15:04")
		views = append(views, v)
	}
	return views, nil
}

// アプリの画面: パッケージ一覧

type commentView struct {
	Number   int64
	Message  string
	Created  string
	Package  *pkgView
	PageLink string
}

func (s *Server) appIndex(c *appContext) {
	ctx := c.r.Context()
	pfs := map[string]string{"android": "Android", "ios": "iOS", "other": "unknown", "all": ""}
	pf := c.form.str("pf")
	if _, ok := pfs[pf]; !ok {
		ua := c.r.UserAgent()
		switch {
		case isAndroid(ua):
			pf = "android"
		case isIOS(ua):
			pf = "ios"
		default:
			pf = "all"
		}
	}
	var activeTags []int64
	for _, t := range strings.Fields(c.form.str("tags")) {
		if id, err := strconv.ParseInt(t, 10, 64); err == nil {
			activeTags = append(activeTags, id)
		}
	}
	cur := max(int(c.form.int("page", 1)), 1)
	pkgs, err := model.PackagesByApp(ctx, s.Core.DB, c.app.ID, pfs[pf], activeTags, (cur-1)*linesInPage, linesInPage+1)
	if err != nil {
		s.serverError(c.uiContext, err)
		return
	}
	hasNext := len(pkgs) > linesInPage
	if hasNext {
		pkgs = pkgs[:linesInPage]
	}
	installed, err := model.PackageInstalledDates(ctx, s.Core.DB, c.user, c.app.ID)
	if err != nil {
		s.serverError(c.uiContext, err)
		return
	}
	var pkgViews []*pkgView
	for i := range pkgs {
		var at *time.Time
		if t, ok := installed[pkgs[i].ID]; ok {
			at = &t
		}
		v, err := s.pkgView(ctx, &pkgs[i], at)
		if err != nil {
			s.serverError(c.uiContext, err)
			return
		}
		pkgViews = append(pkgViews, v)
	}
	tags, err := model.TagsByApp(ctx, s.Core.DB, c.app.ID)
	if err != nil {
		s.serverError(c.uiContext, err)
		return
	}
	var tagViews []tagView
	for _, t := range tags {
		tagViews = append(tagViews, tagView{ID: t.ID, Name: t.Name, Active: slices.Contains(activeTags, t.ID)})
	}
	commentCount, err := model.CountComments(ctx, s.Core.DB, c.app.ID)
	if err != nil {
		s.serverError(c.uiContext, err)
		return
	}
	topComments, err := s.commentViews(c, 0, 2, commentCount)
	if err != nil {
		s.serverError(c.uiContext, err)
		return
	}
	self := requestURL(c.r)
	s.render(c.uiContext, http.StatusOK, "app/index", map[string]any{
		"App":          c.view,
		"Pf":           pf,
		"Tags":         tagViews,
		"FilterOpen":   truthy(c.form.str("filter_open")),
		"Packages":     pkgViews,
		"CommentCount": commentCount,
		"TopComments":  topComments,
		"CurrentPage":  cur,
		"PrevURL":      setQuery(self, "page", strconv.Itoa(cur-1)),
		"NextURL":      setQuery(self, "page", strconv.Itoa(cur+1)),
		"HasNext":      hasNext,
	})
}

func (s *Server) commentViews(c *appContext, offset, limit, total int) ([]commentView, error) {
	ctx := c.r.Context()
	comments, err := model.CommentsByApp(ctx, s.Core.DB, c.app.ID, offset, limit)
	if err != nil {
		return nil, err
	}
	var views []commentView
	for _, cm := range comments {
		v := commentView{Number: cm.Number, Message: cm.Message,
			Created: cm.Created.In(s.Core.Clock.Location()).Format("2006-01-02 15:04")}
		page := (total-int(cm.Number))/linesInPage + 1
		v.PageLink = fmt.Sprintf("/app/comment?id=%d&page=%d#comment-%d", c.app.ID, page, cm.Number)
		if cm.PackageID.Valid {
			p, err := model.GetPackage(ctx, s.Core.DB, cm.PackageID.Int64)
			if err != nil {
				return nil, err
			}
			if p != nil {
				if v.Package, err = s.pkgView(ctx, p, nil); err != nil {
					return nil, err
				}
			}
		}
		views = append(views, v)
	}
	return views, nil
}

// アプリの作成

var dataURLRe = regexp.MustCompile(`^data:[^;]+;base64,(.+)$`)

// iconData は画面から送られた data URL の画像を取り出す. 空なら nil, 形式が違えば ok=false.
func iconData(v string) ([]byte, bool) {
	if v == "" {
		return nil, true
	}
	m := dataURLRe.FindStringSubmatch(v)
	if m == nil {
		return nil, false
	}
	b, err := base64.StdEncoding.DecodeString(m[1])
	return b, err == nil
}

func (s *Server) appNew(c *uiContext) {
	s.render(c, http.StatusOK, "app/new", nil)
}

func (s *Server) appCreate(c *uiContext) {
	title := c.form.str("title")
	image, ok := iconData(c.form.str("icon-data"))
	if title == "" || !ok || image == nil {
		badRequest(c)
		return
	}
	app, err := s.Core.CreateApp(c.r.Context(), c.user, core.AppInfo{
		Title: title, Description: c.form.str("description"), Repository: c.form.str("repository"),
	}, image)
	if errors.Is(err, core.ErrBadRequest) {
		badRequest(c)
		return
	}
	if err != nil {
		s.serverError(c, err)
		return
	}
	redirect(c.w, c.r, fmt.Sprintf("/app?id=%d", app.ID))
}

// コメント

func (s *Server) appComment(c *appContext) {
	ctx := c.r.Context()
	count, err := model.CountComments(ctx, s.Core.DB, c.app.ID)
	if err != nil {
		s.serverError(c.uiContext, err)
		return
	}
	maxPage := int(math.Ceil(float64(count) / linesInPage))
	cur := int(c.form.int("page", 1))
	offset := max(0, min(cur, maxPage)-1) * linesInPage
	comments, err := s.commentViews(c, offset, linesInPage, count)
	if err != nil {
		s.serverError(c.uiContext, err)
		return
	}
	ids, err := model.InstalledPackageIDs(ctx, s.Core.DB, c.user, c.app.ID)
	if err != nil {
		s.serverError(c.uiContext, err)
		return
	}
	pkgs, err := model.GetPackages(ctx, s.Core.DB, ids)
	if err != nil {
		s.serverError(c.uiContext, err)
		return
	}
	var installed []*pkgView
	for i := range pkgs {
		v, err := s.pkgView(ctx, &pkgs[i], nil)
		if err != nil {
			s.serverError(c.uiContext, err)
			return
		}
		installed = append(installed, v)
	}
	s.render(c.uiContext, http.StatusOK, "app/comment", map[string]any{
		"App":               c.view,
		"CommentCount":      count,
		"Comments":          comments,
		"InstalledPackages": installed,
		"Paging":            paging(requestURL(c.r), cur, maxPage),
	})
}

func (s *Server) appCommentPost(c *appContext) {
	ctx := c.r.Context()
	app, cm, err := s.Core.PostComment(ctx, c.app.ID, c.user, c.form.int("package_id", 0), c.form.str("message"))
	if err != nil {
		s.serverError(c.uiContext, err)
		return
	}
	pageURL := absURL(c.r, fmt.Sprintf("/app/comment?id=%d#comment-%d", app.ID, cm.Number), "")
	if err := s.Core.NotifyNewComment(ctx, app, cm, pageURL); err != nil {
		s.Log.Error("notify comment", "app_id", app.ID, "error", err)
	}
	redirect(c.w, c.r, fmt.Sprintf("/app/comment?id=%d", app.ID))
}

// 設定 (オーナーのみ)

func (s *Server) appPreferences(c *appContext) {
	ctx := c.r.Context()
	tags, err := model.TagsByApp(ctx, s.Core.DB, c.app.ID)
	if err != nil {
		s.serverError(c.uiContext, err)
		return
	}
	unused, err := model.UnusedTags(ctx, s.Core.DB, c.app.ID)
	if err != nil {
		s.serverError(c.uiContext, err)
		return
	}
	type prefTag struct {
		ID     int64
		Name   string
		Unused bool
	}
	var views []prefTag
	for _, t := range tags {
		views = append(views, prefTag{t.ID, t.Name, slices.ContainsFunc(unused, func(u model.Tag) bool { return u.ID == t.ID })})
	}
	var others []string
	for _, o := range c.view.Owners {
		if o != c.user {
			others = append(others, o)
		}
	}
	s.render(c.uiContext, http.StatusOK, "app/preferences", map[string]any{
		"App":         c.view,
		"Tags":        views,
		"HasUnused":   len(unused) > 0,
		"OtherOwners": others,
	})
}

func (s *Server) appRefreshAPIKey(c *appContext) {
	if c.r.Method == http.MethodPost {
		if err := s.Core.RefreshAPIKey(c.r.Context(), c.app.ID, c.form.str("api-key")); err != nil {
			s.serverError(c.uiContext, err)
			return
		}
	}
	redirect(c.w, c.r, fmt.Sprintf("/app/preferences?id=%d#refresh-apikey", c.app.ID))
}

func (s *Server) appUpdateInfo(c *appContext) {
	title := c.form.str("title")
	image, ok := iconData(c.form.str("icon-data"))
	if title == "" || !ok {
		badRequest(c.uiContext)
		return
	}
	err := s.Core.UpdateAppInfo(c.r.Context(), c.app.ID, core.AppInfo{
		Title: title, Description: c.form.str("description"), Repository: c.form.str("repository"),
	}, image)
	if errors.Is(err, core.ErrBadRequest) {
		badRequest(c.uiContext)
		return
	}
	if err != nil {
		s.serverError(c.uiContext, err)
		return
	}
	redirect(c.w, c.r, fmt.Sprintf("/app/preferences?id=%d#edit-info", c.app.ID))
}

func (s *Server) appDeleteTags(c *appContext) {
	var ids []int64
	for _, v := range c.form.list("tags") {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			ids = append(ids, id)
		}
	}
	if err := s.Core.DeleteTags(c.r.Context(), c.app.ID, ids); err != nil {
		s.serverError(c.uiContext, err)
		return
	}
	redirect(c.w, c.r, fmt.Sprintf("/app/preferences?id=%d#delete-tags", c.app.ID))
}

func (s *Server) appUpdateOwners(c *appContext) {
	var owners []string
	for _, o := range c.form.list("owners") {
		if o = strings.TrimSpace(o); o != "" && !slices.Contains(owners, o) {
			owners = append(owners, o)
		}
	}
	// 更新した本人は必ずオーナーに残る
	if !slices.Contains(owners, c.user) {
		owners = append(owners, c.user)
	}
	if err := s.Core.SetOwners(c.r.Context(), c.app.ID, owners); err != nil {
		s.serverError(c.uiContext, err)
		return
	}
	redirect(c.w, c.r, fmt.Sprintf("/app/preferences?id=%d#owners", c.app.ID))
}

// アップロード (オーナーのみ)

func (s *Server) appUpload(c *appContext) {
	tags, err := model.TagsByApp(c.r.Context(), s.Core.DB, c.app.ID)
	if err != nil {
		s.serverError(c.uiContext, err)
		return
	}
	s.render(c.uiContext, http.StatusOK, "app/upload", map[string]any{"App": c.view, "Tags": tags})
}

func (s *Server) appUploadPost(c *appContext) {
	ctx := c.r.Context()
	tempName, title := c.form.str("temp_name"), c.form.str("title")
	if tempName == "" || title == "" {
		badRequest(c.uiContext)
		return
	}
	t := core.TempPackage{
		TempName:     tempName,
		Platform:     c.form.str("platform"),
		Identifier:   c.form.str("identifier"),
		OriginalName: c.form.str("file_name"),
	}
	if n, err := strconv.ParseInt(c.form.str("file_size"), 10, 64); err == nil {
		t.FileSize.Int64, t.FileSize.Valid = n, true
	}
	for _, a := range c.form.nested("attached_files") {
		size, _ := strconv.ParseInt(a["size"], 10, 64)
		t.Attached = append(t.Attached, core.TempAttached{FileName: a["file_name"], TempName: a["temp_name"], Type: a["type"], Size: size})
	}
	info := core.PackageInfo{
		Title:       title,
		Description: c.form.ptr("description"),
		Tags:        c.form.list("tags"),
		Protect:     truthy(c.form.str("protect")),
	}
	pkg, err := s.Core.CommitTempPackage(ctx, c.app.ID, info, t)
	if errors.Is(err, core.ErrBadRequest) {
		badRequest(c.uiContext)
		return
	}
	if err != nil {
		s.serverError(c.uiContext, err)
		return
	}
	logAttr(c.r, "pkg_id", pkg.ID)
	if truthy(c.form.str("notify")) {
		if err := s.Core.NotifyPackageUploaded(ctx, c.app, pkg, absURL(c.r, fmt.Sprintf("/package?id=%d", pkg.ID), "")); err != nil {
			s.Log.Error("notify failed", "pkg_id", pkg.ID, "error", err)
		}
	}
	redirect(c.w, c.r, fmt.Sprintf("/package?id=%d", pkg.ID))
}

// UA の判定 (PHP 版の mfwUserAgent と同じ).
func isAndroid(ua string) bool { return strings.Contains(ua, "Android") }

func isIOS(ua string) bool {
	return !isAndroid(ua) && (strings.Contains(ua, "(iPhone;") || strings.Contains(ua, "(iPod;") ||
		strings.Contains(ua, "(iPod touch;") || strings.Contains(ua, "(iPad;"))
}
