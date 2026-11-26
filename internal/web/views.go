package web

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/KLab/emlauncher-go/internal/db"
	"github.com/KLab/emlauncher-go/internal/service"
)

// 画面に渡すデータ (テンプレートから参照する名前で持つ).

// appView はアプリの見出し・情報パネルに出すもの.
type appView struct {
	ID          int64
	Title       string
	Description string
	Repository  string
	APIKey      string
	IconURL     string
	IsOwner     bool

	LastUpload    string
	LastCommented string
	Created       string
	InstallUsers  []string
	Owners        []string

	// Repository が http(s) の URL なら, ホストとパス. RepoIcon は GitHub などのアイコン.
	RepoHost, RepoPath, RepoIcon string
}

var repositoryURL = regexp.MustCompile(`^https?://([^/]*)/(.*)$`)

// appView はアプリの画面の共通部分を読む.
func (s *Server) appView(ctx context.Context, app db.Application, user string) (*appView, error) {
	owners, err := s.svc.Owners(ctx, app.ID)
	if err != nil {
		return nil, err
	}
	users, err := s.svc.AppInstallUsers(ctx, app.ID)
	if err != nil {
		return nil, err
	}
	v := &appView{
		ID:            app.ID,
		Title:         app.Title,
		Description:   app.Description.String,
		Repository:    app.Repository.String,
		APIKey:        app.ApiKey,
		IconURL:       s.svc.IconURL(app),
		IsOwner:       slices.Contains(owners, user),
		LastUpload:    s.nullTime(app.LastUpload),
		LastCommented: s.nullTime(app.LastCommented),
		Created:       s.svc.FormatTime(app.Created),
		InstallUsers:  []string{},
		Owners:        owners,
	}
	for _, u := range users {
		v.InstallUsers = append(v.InstallUsers, u.Mail)
	}
	if m := repositoryURL.FindStringSubmatch(v.Repository); m != nil {
		v.RepoHost, v.RepoPath = m[1], m[2]
		switch {
		case m[1] == "github.com":
			v.RepoIcon = "fa-github"
		case strings.Contains(m[1], "github"): // GitHub Enterprise
			v.RepoIcon = "fa-github-square"
		case strings.Contains(m[1], "bitbucket"):
			v.RepoIcon = "fa-bitbucket"
		}
	}
	return v, nil
}

// pkgView はパッケージの一覧・詳細に出すもの.
type pkgView struct {
	ID               int64
	Title            string
	Description      string
	ShortDescription string
	Identifier       string
	OriginalName     string
	PlatformIcon     string
	PlatformName     string
	Protect          bool
	FileSize         string // "1.5 MB". 不明なら空
	Created          string
	CreatedShort     string
	InstalledAt      string // ログイン中のユーザが最後にインストールした日時. 無ければ空
	InstallURL       string
	SizeWarned       bool
	SizeLimitMB      int
	Tags             []tagView
}

type tagView struct {
	ID     int64
	Name   string
	Active bool
	Unused bool
}

const shortDescriptionLength = 100

// pkgView はパッケージの表示を作る. installed はユーザのインストール日時 (InstalledDates).
func (s *Server) pkgView(d service.PackageDetail, installed map[int64]time.Time) *pkgView {
	p := d.Package
	v := &pkgView{
		ID:           p.ID,
		Title:        p.Title,
		Description:  p.Description.String,
		Identifier:   p.Identifier.String,
		OriginalName: p.OriginalFileName.String,
		Protect:      p.Protect != 0,
		Created:      s.svc.FormatTime(p.Created),
		CreatedShort: p.Created.In(s.cfg.Location).Format("2006-01-02 15:04"),
		InstallURL:   fmt.Sprintf("/package/install?id=%d", p.ID),
		SizeLimitMB:  s.cfg.FileSizeWarningMB(p.Platform),
	}
	v.PlatformIcon, v.PlatformName = platformIcon(p.Platform)
	v.ShortDescription = v.Description
	if utf8.RuneCountInString(v.ShortDescription) > shortDescriptionLength {
		v.ShortDescription = string([]rune(v.ShortDescription)[:shortDescriptionLength]) + "..."
	}
	if v.OriginalName == "" {
		v.OriginalName = "--------." + extOf(p.FileName)
	}
	if p.FileSize.Valid && p.FileSize.Int64 != 0 {
		v.FileSize = formatSize(p.FileSize.Int64)
		v.SizeWarned = v.SizeLimitMB > 0 && p.FileSize.Int64 > int64(v.SizeLimitMB)*1024*1024
	}
	if t, ok := installed[p.ID]; ok {
		v.InstalledAt = s.svc.FormatTime(t)
	}
	for _, t := range d.Tags {
		v.Tags = append(v.Tags, tagView{ID: t.ID, Name: t.Name})
	}
	return v
}

func platformIcon(platform string) (icon, name string) {
	switch platform {
	case "Android":
		return "fa-android", "Android"
	case "iOS":
		return "fa-apple", "iOS"
	}
	return "fa-desktop", "unknown"
}

func extOf(name string) string {
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		return name[i+1:]
	}
	return ""
}

// formatSize は 1024 単位で B / KB / MB / GB にし, 小数点以下 1 桁に丸める (PHP 版と同じ表記).
func formatSize(size int64) string {
	units := []string{"B", "KB", "MB", "GB"}
	v := float64(size)
	i := 0
	for ; i < len(units)-1 && v >= 1024; i++ {
		v = math.Round(v/1024*10) / 10
	}
	return strconv.FormatFloat(v, 'f', -1, 64) + " " + units[i]
}

func (s *Server) nullTime(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return s.svc.FormatTime(t.Time)
}

// appListItem はアプリの一覧 (トップ, 自分のアプリ, インストール済み) の 1 行.
type appListItem struct {
	ID           int64
	Title        string
	IconURL      string
	Comments     int64
	InstallUsers int64
	TimeLabel    string // "last uploaded" / "created"
	Time         string
	Badge        string // "UPDATE" / "NEW" / ""
	Notify       bool
}

// appListItem はアプリの一覧の行を作る. installedAt はユーザがアプリを最後にインストールした日時.
//
// UPDATE: インストールした後に新しいパッケージがアップロードされた. NEW: 昨日以降にアップロード (作成) された.
func (s *Server) appListItem(app db.Application, installedAt *time.Time) appListItem {
	it := appListItem{ID: app.ID, Title: app.Title, IconURL: s.svc.IconURL(app), TimeLabel: "created"}
	updated := app.Created
	if app.LastUpload.Valid {
		it.TimeLabel, updated = "last uploaded", app.LastUpload.Time
	}
	updated = updated.In(s.cfg.Location)
	it.Time = updated.Format("2006-01-02 15:04")
	now := time.Now().In(s.cfg.Location)
	yesterday := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, s.cfg.Location)
	switch {
	case installedAt != nil && app.LastUpload.Valid && app.LastUpload.Time.After(*installedAt):
		it.Badge = "UPDATE"
	case updated.After(yesterday):
		it.Badge = "NEW"
	}
	return it
}

// installedApps はユーザがインストール済みのアプリの最後のインストール日時 (app_id がキー).
func (s *Server) installedApps(ctx context.Context, user string) (map[int64]*db.AppInstallUser, error) {
	ius, err := s.svc.InstalledApps(ctx, user)
	if err != nil {
		return nil, err
	}
	m := make(map[int64]*db.AppInstallUser, len(ius))
	for i := range ius {
		m[ius[i].AppID] = &ius[i]
	}
	return m, nil
}

func lastInstalled(m map[int64]*db.AppInstallUser, appID int64) *time.Time {
	if iu, ok := m[appID]; ok {
		return &iu.LastInstalled
	}
	return nil
}

// pagingView はページ番号のリンク.
type pagingView struct {
	Items []pagingItem
}

type pagingItem struct {
	Num      int
	URL      string
	Current  bool // 表示中のページ (リンクにしない)
	Active   bool // 最初・最後のページへのリンクで, それが表示中のページ
	Omission bool // 省略 (..)
}

// paging は表示中のページ cur の前後 5 ページと, 最初・最後のページへのリンクを作る (PHP 版と同じ).
// pageURL はページ番号のリンク先.
func paging(cur, max int, pageURL func(n int) string) pagingView {
	const candidate = 5
	var start, end int
	switch {
	case max <= candidate+2:
		start, end = 1, max
	case cur <= (candidate+1)/2:
		start, end = 1, candidate+1
	case cur >= max-candidate/2:
		start, end = max-candidate, max
	default:
		start = cur - (candidate+1)/2 + 1
		if start < 1 {
			start = 1
		}
		end = min(max, start+candidate-1)
	}
	var v pagingView
	if start != 1 {
		v.Items = append(v.Items, pagingItem{Num: 1, URL: pageURL(1), Active: cur == 1})
	}
	if start > 2 {
		v.Items = append(v.Items, pagingItem{Omission: true})
	}
	for i := start; i <= end; i++ {
		v.Items = append(v.Items, pagingItem{Num: i, URL: pageURL(i), Current: i == cur})
	}
	if end < max-1 {
		v.Items = append(v.Items, pagingItem{Omission: true})
	}
	if end != max {
		v.Items = append(v.Items, pagingItem{Num: max, URL: pageURL(max), Active: cur == max})
	}
	return v
}

// withPage は URL のクエリの page を n にしたもの.
func withPage(r *http.Request, n int) string {
	q := r.URL.Query()
	q.Set("page", strconv.Itoa(n))
	return baseURL(r) + r.URL.Path + "?" + q.Encode()
}
