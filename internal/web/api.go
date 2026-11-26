package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/KLab/emlauncher-go/internal/db"
	"github.com/KLab/emlauncher-go/internal/service"
)

// API (/api/...) は api_key でアプリを特定する. 応答は JSON. JSON のキーは PHP 版と同じで,
// DB の数値 (id, file_size) は文字列で返す (API ドキュメントの Caveats のとおり型は不定としている).

// requireAPIKey は API キーのアプリを context に入れる. 無効なキーなら 400.
func (s *Server) requireAPIKey(next http.Handler) http.Handler {
	return s.api(func(w http.ResponseWriter, r *http.Request) error {
		key, ok := s.requestAPIKey(r)
		if !ok && !s.cfg.APIKeyParamEnabled && formOf(r).has("api_key") {
			return badRequest("api_key request parameter is disabled. Use the Authorization: Bearer header.")
		}
		app, err := s.svc.GetAppByAPIKey(r.Context(), key)
		if errors.Is(err, service.ErrNotFound) {
			return badRequest("Invalid api_key")
		}
		if err != nil {
			return err
		}
		logAttr(r, "app_id", app.ID)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxApp{}, app)))
		return nil
	})
}

var bearerRe = regexp.MustCompile(`(?i)^Bearer +(\S+) *$`)

// requestAPIKey は API キー. Authorization: Bearer ヘッダを優先し, 無ければ
// (api_key_param_enabled なら) リクエストパラメータ api_key を使う.
func (s *Server) requestAPIKey(r *http.Request) (string, bool) {
	for _, v := range r.Header.Values("Authorization") {
		if m := bearerRe.FindStringSubmatch(v); m != nil {
			return m[1], true
		}
	}
	if !s.cfg.APIKeyParamEnabled {
		return "", false
	}
	f := formOf(r)
	return f.Get("api_key"), f.has("api_key")
}

func apiApp(r *http.Request) db.Application {
	return r.Context().Value(ctxApp{}).(db.Application)
}

// requireAPIPackage は id のパッケージ (api_key のアプリのもの) を context に入れる.
func (s *Server) requireAPIPackage(next http.Handler) http.Handler {
	return s.api(func(w http.ResponseWriter, r *http.Request) error {
		f := formOf(r)
		if !f.has("id") {
			return badRequest("id required")
		}
		id := f.intParam("id", 0)
		pkg, err := s.svc.GetPackage(r.Context(), id)
		if errors.Is(err, service.ErrNotFound) {
			return badRequest(fmt.Sprintf("The package (id=%d) not found.", id))
		}
		if err != nil {
			return err
		}
		logAttr(r, "pkg_id", pkg.ID)
		logAttr(r, "platform", pkg.Platform)
		app := apiApp(r)
		if pkg.AppID != app.ID {
			return errForbidden
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxPackage{}, pkgAndApp{pkg, app})))
		return nil
	})
}

func apiPackage(r *http.Request) db.Package {
	return r.Context().Value(ctxPackage{}).(pkgAndApp).pkg
}

type packageJSON struct {
	PackageURL       string         `json:"package_url"`
	ApplicationURL   string         `json:"application_url"`
	ID               string         `json:"id"`
	Platform         string         `json:"platform"`
	Title            string         `json:"title"`
	Description      *string        `json:"description"`
	Identifier       *string        `json:"identifier"`
	OriginalFileName *string        `json:"original_file_name"`
	FileSize         *string        `json:"file_size"`
	Protect          bool           `json:"protect"`
	Created          string         `json:"created"`
	Tags             []string       `json:"tags"`
	InstallCount     int64          `json:"install_count"`
	AttachedFiles    []attachedJSON `json:"attached_files"`
}

type attachedJSON struct {
	ID               string `json:"id"`
	OriginalFileName string `json:"original_file_name"`
	FileSize         string `json:"file_size"`
	Created          string `json:"created"`
}

func (s *Server) packageJSON(r *http.Request, d service.PackageDetail) packageJSON {
	p := d.Package
	j := packageJSON{
		PackageURL:     fmt.Sprintf("%s/package?id=%d", baseURL(r), p.ID),
		ApplicationURL: fmt.Sprintf("%s/app?id=%d", baseURL(r), p.AppID),
		ID:             strconv.FormatInt(p.ID, 10),
		Platform:       p.Platform,
		Title:          p.Title,
		Protect:        p.Protect != 0,
		Created:        s.svc.FormatTime(p.Created),
		Tags:           []string{},
		InstallCount:   d.InstallCount,
		AttachedFiles:  []attachedJSON{},
	}
	if p.Description.Valid {
		j.Description = &p.Description.String
	}
	if p.Identifier.Valid {
		j.Identifier = &p.Identifier.String
	}
	if p.OriginalFileName.Valid {
		j.OriginalFileName = &p.OriginalFileName.String
	}
	if p.FileSize.Valid {
		size := strconv.FormatInt(p.FileSize.Int64, 10)
		j.FileSize = &size
	}
	for _, t := range d.Tags {
		j.Tags = append(j.Tags, t.Name)
	}
	for _, a := range d.Attached {
		j.AttachedFiles = append(j.AttachedFiles, s.attachedJSON(a))
	}
	return j
}

func (s *Server) attachedJSON(a db.AttachedFile) attachedJSON {
	return attachedJSON{
		ID:               strconv.FormatInt(a.ID, 10),
		OriginalFileName: a.OriginalFileName,
		FileSize:         strconv.FormatInt(a.FileSize, 10),
		Created:          s.svc.FormatTime(a.Created),
	}
}

// parseBool は API の真偽値. 空, "0", false / no / null などは偽 (PHP 版と同じ).
func parseBool(v string) bool {
	if !truthy(v) {
		return false
	}
	return !slices.Contains([]string{"False", "false", "F", "f", "No", "no", "Null", "null"}, v)
}

// splitTags はカンマ区切りのタグ名.
func splitTags(v string) []string {
	if v == "" {
		return []string{}
	}
	return strings.Split(v, ",")
}

func localFile(f *uploadedFile) *service.LocalFile {
	if f == nil {
		return nil
	}
	return &service.LocalFile{Path: f.Path, Name: f.Name, Size: f.Size}
}

func (s *Server) apiUpload(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return errorStatus(http.StatusMethodNotAllowed, "Method Not Allowed")
	}
	f := formOf(r)
	file, title := f.file("file"), f.Get("title")
	var missing []string
	if file == nil {
		missing = append(missing, "file")
	}
	if title == "" {
		missing = append(missing, "title")
	}
	if len(missing) > 0 {
		return badRequest(fmt.Sprintf("A required field (%s) is not present.", strings.Join(missing, ",")))
	}
	d, err := s.svc.Upload(r.Context(), apiApp(r).ID, *localFile(file), service.UploadOptions{
		DSYM: localFile(f.file("dsym")),
		APK:  localFile(f.file("apk")),
	}, service.NewPackage{
		Title:       title,
		Description: f.optional("description"),
		Tags:        splitTags(f.Get("tags")),
		Protect:     parseBool(f.Get("protect")),
		Notify:      parseBool(f.Get("notify")),
		BaseURL:     baseURL(r),
	})
	if err != nil {
		return err
	}
	logPackage(r, d.Package)
	return writeJSON(w, http.StatusOK, s.packageJSON(r, d))
}

func (s *Server) apiPackageList(w http.ResponseWriter, r *http.Request) error {
	f := formOf(r)
	app := apiApp(r)
	limit := min(max(int(f.phpInt("limit", 20)), 0), 100)
	offset := max(int(f.phpInt("offset", 0)), 0)
	var names []string
	for _, t := range splitTags(f.Get("tags")) {
		if t != "" && !slices.Contains(names, t) {
			names = append(names, t)
		}
	}
	tagIDs, ok, err := s.svc.TagIDsByName(r.Context(), app.ID, names)
	if err != nil {
		return err
	}
	list := []packageJSON{}
	if !ok {
		// 存在しないタグが指定されたら空
		return writeJSON(w, http.StatusOK, list)
	}
	pkgs, err := s.svc.ListPackages(r.Context(), service.PackageQuery{
		AppID: app.ID, Platform: f.Get("platform"), TagIDs: tagIDs, Limit: limit, Offset: offset,
	})
	if err != nil {
		return err
	}
	details, err := s.svc.Details(r.Context(), pkgs...)
	if err != nil {
		return err
	}
	for _, d := range details {
		list = append(list, s.packageJSON(r, d))
	}
	return writeJSON(w, http.StatusOK, list)
}

// apiTargetPackage は id のパッケージ. 無いか別のアプリのものなら "Invalid package id".
func (s *Server) apiTargetPackage(r *http.Request) (db.Package, error) {
	id := formOf(r).intParam("id", 0)
	logAttr(r, "pkg_id", id)
	pkg, err := s.svc.GetPackage(r.Context(), id)
	if errors.Is(err, service.ErrNotFound) || err == nil && pkg.AppID != apiApp(r).ID {
		return pkg, badRequest("Invalid package id")
	}
	return pkg, err
}

func (s *Server) apiDelete(w http.ResponseWriter, r *http.Request) error {
	pkg, err := s.apiTargetPackage(r)
	if err != nil {
		return err
	}
	d, err := s.svc.DeletePackage(r.Context(), pkg.ID)
	if errors.Is(err, service.ErrNotFound) {
		return badRequest("Invalid package id")
	}
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, s.packageJSON(r, d))
}

func (s *Server) apiCreateToken(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	f := formOf(r)
	pkg, err := s.apiTargetPackage(r)
	if err != nil {
		return err
	}
	mail := f.Get("mail")
	owners, err := s.svc.Owners(ctx, pkg.AppID)
	if err != nil {
		return err
	}
	if !slices.Contains(owners, mail) {
		return badRequest("Invalid mail address")
	}
	hours := 1.0
	if v := f.Get("expire_hour"); v != "" && v != "0" {
		if hours, err = strconv.ParseFloat(v, 64); err != nil || hours <= 0 {
			return badRequest("Invalid expire_hour")
		}
	}
	t, err := s.svc.CreateInstallToken(ctx, pkg, mail, time.Duration(hours*float64(time.Hour)))
	if err != nil {
		return err
	}
	d, err := s.svc.Detail(ctx, pkg)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, struct {
		packageJSON
		InstallURL string `json:"install_url"`
	}{s.packageJSON(r, d), baseURL(r) + "/package/install?token=" + t.Token})
}

func (s *Server) apiPackageEdit(w http.ResponseWriter, r *http.Request) error {
	f := formOf(r)
	var e service.PackageEdit
	e.Title = f.optional("title")
	e.Description = f.optional("description")
	if f.has("protect") {
		p := parseBool(f.Get("protect"))
		e.Protect = &p
	}
	if f.has("tags") {
		tags := splitTags(f.Get("tags"))
		e.Tags = &tags
	}
	if e.Title != nil && *e.Title == "" {
		return badRequest("title must not empty")
	}
	if e.Title == nil && e.Description == nil && e.Protect == nil && e.Tags == nil {
		return badRequest("no parameter")
	}
	pkg, err := s.svc.EditPackage(r.Context(), apiPackage(r).ID, e)
	if err != nil {
		return err
	}
	d, err := s.svc.Detail(r.Context(), pkg)
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, s.packageJSON(r, d))
}

func (s *Server) apiPackageAttach(w http.ResponseWriter, r *http.Request) error {
	file := formOf(r).file("file")
	if file == nil {
		return badRequest("file required")
	}
	a, err := s.svc.AttachFile(r.Context(), apiPackage(r), service.Attachment{Path: file.Path, Name: file.Name, Size: file.Size})
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, s.attachedJSON(a))
}

// apiAttachedFile は attached_file_id の, パッケージの添付ファイル.
func (s *Server) apiAttachedFile(r *http.Request) (db.AttachedFile, error) {
	f := formOf(r)
	if !f.has("attached_file_id") {
		return db.AttachedFile{}, badRequest("attached_file_id required")
	}
	pkg := apiPackage(r)
	id := f.intParam("attached_file_id", 0)
	logAttr(r, "attached_id", id)
	a, err := s.svc.GetAttachedFile(r.Context(), pkg.ID, id)
	if errors.Is(err, service.ErrNotFound) {
		return a, errorStatus(http.StatusNotFound,
			fmt.Sprintf("the attached file (id=%d) not found in the package (id=%d)", id, pkg.ID))
	}
	return a, err
}

func (s *Server) apiPackageDetach(w http.ResponseWriter, r *http.Request) error {
	a, err := s.apiAttachedFile(r)
	if err != nil {
		return err
	}
	if err := s.svc.DeleteAttachedFile(r.Context(), a); err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, s.attachedJSON(a))
}

func (s *Server) apiPackageAttachDownload(w http.ResponseWriter, r *http.Request) error {
	a, err := s.apiAttachedFile(r)
	if err != nil {
		return err
	}
	expire := s.svc.Now().Add(service.DownloadExpiry)
	u, err := s.svc.AttachedDownloadURL(r.Context(), a)
	if err != nil {
		return err
	}
	if strings.HasPrefix(u, "/") {
		u = baseURL(r) + u
	}
	return writeJSON(w, http.StatusOK, map[string]any{
		"attached_file": s.attachedJSON(a),
		"download_url":  u,
		"expire":        s.svc.FormatTime(expire),
	})
}
