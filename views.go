package emlauncher

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// 画面に渡すデータ (ビュー). 時刻などは表示用の文字列にしておく.

type appView struct {
	ID            int64
	Title         string
	Description   string
	IconURL       string
	APIKey        string
	LastUpload    string
	LastCommented string
	Created       string
	IsOwner       bool
	InstallUsers  []string
	Repository    string
	RepoHost      string
	RepoPath      string
	RepoIcon      string
	Owners        []string
}

var repoURLRe = regexp.MustCompile(`^https?://([^/]*)/(.*)$`)

func (s *Server) iconURL(a *Application) string {
	return s.Core.Storage.PublicURL(a.IconKey.String)
}

func (s *Server) appView(ctx context.Context, a *Application, user string) (*appView, error) {
	clk := s.Core.Clock
	v := &appView{
		ID:          a.ID,
		Title:       a.Title,
		Description: a.Description.String,
		IconURL:     s.iconURL(a),
		APIKey:      a.APIKey,
		Created:     clk.Format(a.Created),
		Repository:  a.Repository.String,
	}
	if a.LastUpload.Valid {
		v.LastUpload = clk.Format(a.LastUpload.Time)
	}
	if a.LastCommented.Valid {
		v.LastCommented = clk.Format(a.LastCommented.Time)
	}
	if m := repoURLRe.FindStringSubmatch(v.Repository); m != nil {
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
	var err error
	if v.Owners, err = Owners(ctx, s.Core.DB, a.ID); err != nil {
		return nil, err
	}
	for _, o := range v.Owners {
		if o == user {
			v.IsOwner = true
		}
	}
	users, err := InstallUsersByApp(ctx, s.Core.DB, a.ID)
	if err != nil {
		return nil, err
	}
	v.InstallUsers = []string{}
	for _, u := range users {
		v.InstallUsers = append(v.InstallUsers, u.Mail)
	}
	return v, nil
}

type tagView struct {
	ID     int64
	Name   string
	Active bool
}

type pkgView struct {
	ID               int64
	AppID            int64
	Title            string
	Platform         string
	PlatformIcon     string
	PlatformName     string
	Description      string
	ShortDescription string
	Identifier       string
	OriginalName     string
	FileSize         string
	Created          string
	CreatedShort     string
	Protect          bool
	Tags             []tagView
	InstallURL       string
	InstalledAt      string
	SizeLimitMB      int
	SizeWarned       bool
}

const shortDescriptionLength = 100

func (s *Server) pkgView(ctx context.Context, p *Package, installedAt *time.Time) (*pkgView, error) {
	clk := s.Core.Clock
	v := &pkgView{
		ID:           p.ID,
		AppID:        p.AppID,
		Title:        p.Title,
		Platform:     p.Platform,
		Description:  p.Description.String,
		Identifier:   p.Identifier.String,
		OriginalName: p.OriginalFileName.String,
		Created:      clk.Format(p.Created),
		CreatedShort: p.Created.In(clk.Location()).Format("2006-01-02 15:04"),
		Protect:      p.Protect,
		InstallURL:   fmt.Sprintf("/package/install?id=%d", p.ID),
	}
	switch p.Platform {
	case "Android":
		v.PlatformIcon, v.PlatformName = "fa-android", "Android"
		v.SizeLimitMB = s.Config.Package.FileSizeWarningAndroid
	case "iOS":
		v.PlatformIcon, v.PlatformName = "fa-apple", "iOS"
		v.SizeLimitMB = s.Config.Package.FileSizeWarningIOS
	default:
		v.PlatformIcon, v.PlatformName = "fa-desktop", "unknown"
	}
	v.SizeWarned = v.SizeLimitMB > 0 && p.FileSize.Int64 > int64(v.SizeLimitMB)*1024*1024
	if v.OriginalName == "" {
		v.OriginalName = "--------." + strings.TrimPrefix(fileExt(p.FileName), ".")
	}
	if p.FileSize.Valid && p.FileSize.Int64 > 0 {
		v.FileSize = formatSize(p.FileSize.Int64)
	}
	v.ShortDescription = v.Description
	if utf8.RuneCountInString(v.ShortDescription) > shortDescriptionLength {
		v.ShortDescription = string([]rune(v.ShortDescription)[:shortDescriptionLength]) + "..."
	}
	if installedAt != nil {
		v.InstalledAt = clk.Format(*installedAt)
	}
	tags, err := TagsByPackage(ctx, s.Core.DB, p.ID)
	if err != nil {
		return nil, err
	}
	for _, t := range tags {
		v.Tags = append(v.Tags, tagView{ID: t.ID, Name: t.Name})
	}
	return v, nil
}

func fileExt(name string) string {
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		return name[i:]
	}
	return ""
}

// formatSize は PHP 版と同じく 1024 で割りながら小数第 1 位に丸める.
func formatSize(n int64) string {
	units := []string{"B", "KB", "MB", "GB"}
	size := float64(n)
	i := 0
	for ; i < len(units)-1; i++ {
		if size < 1024 {
			break
		}
		size = math.Round(size/1024*10) / 10
	}
	return strings.TrimSuffix(strings.TrimSuffix(fmt.Sprintf("%.1f", size), "0"), ".") + " " + units[i]
}

// updateBadge はアプリ一覧の UPDATE / NEW ラベル.
// インストール後に更新されていれば UPDATE, 昨日以降に更新・作成されていれば NEW.
func (s *Server) updateBadge(a *Application, installed map[int64]time.Time) string {
	if inst, ok := installed[a.ID]; ok && a.LastUpload.Valid && a.LastUpload.Time.After(inst) {
		return "UPDATE"
	}
	update := a.Created
	if a.LastUpload.Valid {
		update = a.LastUpload.Time
	}
	now := s.Core.Clock.Now()
	y, m, d := now.AddDate(0, 0, -1).Date()
	if update.After(time.Date(y, m, d, 0, 0, 0, 0, now.Location())) {
		return "NEW"
	}
	return ""
}

// installedApps はユーザがインストールしたアプリと最終インストール日時.
func (s *Server) installedApps(ctx context.Context, user string) (map[int64]time.Time, error) {
	ias, err := InstalledApps(ctx, s.Core.DB, user)
	if err != nil {
		return nil, err
	}
	m := map[int64]time.Time{}
	for _, ia := range ias {
		m[ia.AppID] = ia.LastInstalled
	}
	return m, nil
}

type pagingView struct {
	Items []pagingItem
}

type pagingItem struct {
	Num      int
	URL      string
	Current  bool
	Active   bool
	Omission bool
}

// paging は PHP 版の blocks/paging.php と同じページ番号の並び (前後を省略する) を作る.
func paging(base string, cur, max int) pagingView {
	const candidate = 5
	var start, end int
	switch {
	case max <= candidate+2:
		start, end = 1, max
	case cur <= (candidate+1)/2:
		start, end = 1, candidate+1
	case cur >= max-candidate/2:
		end = max
		start = end - candidate
	default:
		start = 1
		if v := cur - (candidate+1)/2 + 1; v > start {
			start = v
		}
		end = start + candidate - 1
		if end > max {
			end = max
		}
	}
	link := func(n int) string { return setQuery(base, "page", fmt.Sprint(n)) }
	var v pagingView
	if start != 1 {
		v.Items = append(v.Items, pagingItem{Num: 1, URL: link(1), Active: cur == 1})
	}
	if start > 2 {
		v.Items = append(v.Items, pagingItem{Omission: true})
	}
	for i := start; i <= end; i++ {
		v.Items = append(v.Items, pagingItem{Num: i, URL: link(i), Current: i == cur})
	}
	if end < max-1 {
		v.Items = append(v.Items, pagingItem{Omission: true})
	}
	if end != max && max > 0 {
		v.Items = append(v.Items, pagingItem{Num: max, URL: link(max), Active: cur == max})
	}
	return v
}

// setQuery は URL のクエリパラメータを置き換える (PHP 版の mfwHttp::composeUrl).
func setQuery(rawURL, key, value string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	q := u.Query()
	q.Set(key, value)
	u.RawQuery = q.Encode()
	return u.String()
}
