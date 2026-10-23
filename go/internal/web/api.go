package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"emlauncher/internal/core"
	"emlauncher/internal/model"
)

// API (/api/{action}). api_key で認証する. 仕様は /doc/api と PHP 版 (mainmodules/api) を参照.

type apiContext struct {
	w    http.ResponseWriter
	r    *http.Request
	form *form
	app  *model.Application
}

func (s *Server) api(w http.ResponseWriter, r *http.Request) {
	action := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api"), "/")
	f, err := parseForm(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	defer f.cleanup()

	app, err := model.GetApplicationByAPIKey(r.Context(), s.Core.DB, f.str("api_key"))
	if err != nil {
		s.apiError(w, r, err)
		return
	}
	if app == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid api_key"})
		return
	}
	logAttr(r, "app_id", app.ID)
	c := &apiContext{w: w, r: r, form: f, app: app}

	switch action {
	case "upload":
		s.apiUpload(c)
	case "package_list":
		s.apiPackageList(c)
	case "delete":
		s.apiDelete(c)
	case "create_token":
		s.apiCreateToken(c)
	case "package_edit", "package_attach", "package_detach":
		pkg, ok := s.apiTargetPackage(c)
		if !ok {
			return
		}
		switch action {
		case "package_edit":
			s.apiPackageEdit(c, pkg)
		case "package_attach":
			s.apiPackageAttach(c, pkg)
		case "package_detach":
			s.apiPackageDetach(c, pkg)
		}
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "404 Not Found"})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (s *Server) apiError(w http.ResponseWriter, r *http.Request, err error) {
	s.Log.Error("api error", "uri", r.URL.Path, "error", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error(), "exception": "Exception"})
}

// packageJSON は API が返すパッケージ. 数値フィールドは文字列にする
// (API ドキュメントの Caveats のとおり型は不定だが, PHP 版は DB から読んだ値を文字列で返していた).
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
	InstallCount     int            `json:"install_count"`
	AttachedFiles    []attachedJSON `json:"attached_files"`
}

type attachedJSON struct {
	ID               string `json:"id"`
	OriginalFileName string `json:"original_file_name"`
	FileSize         string `json:"file_size"`
	Created          string `json:"created"`
}

func (s *Server) attachedJSON(a *model.AttachedFile) attachedJSON {
	return attachedJSON{
		ID:               strconv.FormatInt(a.ID, 10),
		OriginalFileName: a.OriginalFileName,
		FileSize:         strconv.FormatInt(a.FileSize, 10),
		Created:          s.Core.Clock.Format(a.Created),
	}
}

func (s *Server) packageJSON(ctx context.Context, r *http.Request, p *model.Package) (*packageJSON, error) {
	j := &packageJSON{
		PackageURL:     absURL(r, fmt.Sprintf("/package?id=%d", p.ID), ""),
		ApplicationURL: absURL(r, fmt.Sprintf("/app?id=%d", p.AppID), ""),
		ID:             strconv.FormatInt(p.ID, 10),
		Platform:       p.Platform,
		Title:          p.Title,
		Protect:        p.Protect,
		Created:        s.Core.Clock.Format(p.Created),
		Tags:           []string{},
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
		v := strconv.FormatInt(p.FileSize.Int64, 10)
		j.FileSize = &v
	}
	tags, err := model.TagsByPackage(ctx, s.Core.DB, p.ID)
	if err != nil {
		return nil, err
	}
	for _, t := range tags {
		j.Tags = append(j.Tags, t.Name)
	}
	if j.InstallCount, err = model.InstallUserCount(ctx, s.Core.DB, p.ID); err != nil {
		return nil, err
	}
	afs, err := model.AttachedFilesByPackage(ctx, s.Core.DB, p.ID)
	if err != nil {
		return nil, err
	}
	for i := range afs {
		j.AttachedFiles = append(j.AttachedFiles, s.attachedJSON(&afs[i]))
	}
	return j, nil
}

func (s *Server) writePackage(c *apiContext, p *model.Package) {
	j, err := s.packageJSON(c.r.Context(), c.r, p)
	if err != nil {
		s.apiError(c.w, c.r, err)
		return
	}
	writeJSON(c.w, http.StatusOK, j)
}

func splitTags(s string) []string {
	return strings.Split(s, ",")
}

func (s *Server) apiUpload(c *apiContext) {
	if c.r.Method != http.MethodPost {
		writeJSON(c.w, http.StatusMethodNotAllowed, map[string]string{"error": "Method Not Allowed"})
		return
	}
	ctx := c.r.Context()
	file := c.form.file("file")
	title := c.form.str("title")
	if file == nil || title == "" {
		var fields []string
		if file == nil {
			fields = append(fields, "file")
		}
		if title == "" {
			fields = append(fields, "title")
		}
		writeJSON(c.w, http.StatusBadRequest, map[string]string{
			"error": "A required field (" + strings.Join(fields, ",") + ") is not present."})
		return
	}

	an, err := s.Core.Analyze(ctx, file.Name, file.Path)
	if err != nil {
		s.apiError(c.w, c.r, err)
		return
	}
	defer an.Cleanup()
	var extra []core.AttachUpload
	if d := c.form.file("dsym"); d != nil {
		extra = append(extra, core.AttachUpload{Upload: *d, Type: model.AttachedTypeDSYM})
	}
	info := core.PackageInfo{
		Title:       title,
		Description: c.form.ptr("description"),
		Tags:        splitTags(c.form.str("tags")),
		Protect:     parseBool(c.form.str("protect")),
	}
	pkg, err := s.Core.UploadPackage(ctx, c.app.ID, info, *file, an, extra)
	if err != nil {
		s.apiError(c.w, c.r, err)
		return
	}
	logAttr(c.r, "pkg_id", pkg.ID)

	if parseBool(c.form.str("notify")) {
		// 通知が送れなくてもアップロードは成功とする
		if err := s.Core.NotifyPackageUploaded(ctx, c.app, pkg, absURL(c.r, fmt.Sprintf("/package?id=%d", pkg.ID), "")); err != nil {
			s.Log.Error("notify failed", "pkg_id", pkg.ID, "error", err)
		}
	}
	s.writePackage(c, pkg)
}

func (s *Server) apiPackageList(c *apiContext) {
	ctx := c.r.Context()
	limit := int(c.form.int("limit", 20))
	if limit > 100 {
		limit = 100
	}
	limit = max(limit, 0)
	offset := max(int(c.form.int("offset", 0)), 0)
	platform := c.form.str("platform")

	var names []string
	seen := map[string]bool{}
	for _, t := range splitTags(c.form.str("tags")) {
		if t != "" && !seen[t] {
			seen[t] = true
			names = append(names, t)
		}
	}
	tags, err := model.TagsByNames(ctx, s.Core.DB, c.app.ID, names)
	if err != nil {
		s.apiError(c.w, c.r, err)
		return
	}
	// 存在しないタグが指定された場合は空
	if len(tags) != len(names) {
		writeJSON(c.w, http.StatusOK, []any{})
		return
	}
	var tagIDs []int64
	for _, t := range tags {
		tagIDs = append(tagIDs, t.ID)
	}
	pkgs, err := model.PackagesByApp(ctx, s.Core.DB, c.app.ID, platform, tagIDs, offset, limit)
	if err != nil {
		s.apiError(c.w, c.r, err)
		return
	}
	out := []*packageJSON{}
	for i := range pkgs {
		j, err := s.packageJSON(ctx, c.r, &pkgs[i])
		if err != nil {
			s.apiError(c.w, c.r, err)
			return
		}
		out = append(out, j)
	}
	writeJSON(c.w, http.StatusOK, out)
}

func (s *Server) apiDelete(c *apiContext) {
	ctx := c.r.Context()
	pkg, err := model.GetPackage(ctx, s.Core.DB, c.form.int("id", 0))
	if err != nil {
		s.apiError(c.w, c.r, err)
		return
	}
	if pkg == nil || pkg.AppID != c.app.ID {
		writeJSON(c.w, http.StatusBadRequest, map[string]string{"error": "Invalid package id"})
		return
	}
	// 削除前の内容を返す
	j, err := s.packageJSON(ctx, c.r, pkg)
	if err != nil {
		s.apiError(c.w, c.r, err)
		return
	}
	if _, _, err := s.Core.DeletePackage(ctx, c.app.ID, pkg.ID); err != nil {
		s.apiError(c.w, c.r, err)
		return
	}
	logAttr(c.r, "pkg_id", pkg.ID)
	writeJSON(c.w, http.StatusOK, j)
}

func (s *Server) apiCreateToken(c *apiContext) {
	ctx := c.r.Context()
	pkg, err := model.GetPackage(ctx, s.Core.DB, c.form.int("id", 0))
	if err != nil {
		s.apiError(c.w, c.r, err)
		return
	}
	if pkg == nil || pkg.AppID != c.app.ID {
		writeJSON(c.w, http.StatusBadRequest, map[string]string{"error": "Invalid package id"})
		return
	}
	mailAddr := c.form.str("mail")
	owner, err := model.IsOwner(ctx, s.Core.DB, c.app.ID, mailAddr)
	if err != nil {
		s.apiError(c.w, c.r, err)
		return
	}
	if !owner {
		writeJSON(c.w, http.StatusBadRequest, map[string]string{"error": "Invalid mail address"})
		return
	}
	hours, err := strconv.ParseFloat(c.form.str("expire_hour"), 64)
	if err != nil || hours <= 0 {
		hours = 1
	}
	token, _, err := s.Core.CreateInstallToken(mailAddr, pkg.ID, time.Duration(hours*float64(time.Hour)))
	if err != nil {
		s.apiError(c.w, c.r, err)
		return
	}
	j, err := s.packageJSON(ctx, c.r, pkg)
	if err != nil {
		s.apiError(c.w, c.r, err)
		return
	}
	writeJSON(c.w, http.StatusOK, struct {
		*packageJSON
		InstallURL string `json:"install_url"`
	}{j, absURL(c.r, "/package/install?token="+token, "")})
}

// apiTargetPackage は package_edit / attach / detach の対象パッケージ.
func (s *Server) apiTargetPackage(c *apiContext) (*model.Package, bool) {
	if !c.form.has("id") {
		writeJSON(c.w, http.StatusBadRequest, map[string]string{"error": "id required"})
		return nil, false
	}
	id := c.form.int("id", 0)
	pkg, err := model.GetPackage(c.r.Context(), s.Core.DB, id)
	if err != nil {
		s.apiError(c.w, c.r, err)
		return nil, false
	}
	if pkg == nil {
		writeJSON(c.w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("The package (id=%d) not found.", id)})
		return nil, false
	}
	if pkg.AppID != c.app.ID {
		writeJSON(c.w, http.StatusForbidden, map[string]string{"error": "Forbidden"})
		return nil, false
	}
	return pkg, true
}

func (s *Server) apiPackageEdit(c *apiContext, pkg *model.Package) {
	var e core.PackageEdit
	if c.form.has("title") {
		e.Title = c.form.ptr("title")
		if e.Title == nil || *e.Title == "" {
			writeJSON(c.w, http.StatusBadRequest, map[string]string{"error": "title must not empty"})
			return
		}
	}
	if c.form.has("description") {
		e.Description = c.form.ptr("description")
	}
	if c.form.has("protect") {
		p := parseBool(c.form.str("protect"))
		e.Protect = &p
	}
	if c.form.has("tags") {
		tags := splitTags(c.form.str("tags"))
		e.Tags = &tags
	}
	if e.Title == nil && e.Description == nil && e.Protect == nil && e.Tags == nil {
		writeJSON(c.w, http.StatusBadRequest, map[string]string{"error": "no parameter"})
		return
	}
	pkg, err := s.Core.EditPackage(c.r.Context(), c.app.ID, pkg.ID, e)
	if err != nil {
		s.apiError(c.w, c.r, err)
		return
	}
	s.writePackage(c, pkg)
}

func (s *Server) apiPackageAttach(c *apiContext, pkg *model.Package) {
	file := c.form.file("file")
	if file == nil {
		writeJSON(c.w, http.StatusBadRequest, map[string]string{"error": "file required"})
		return
	}
	af, err := s.Core.AttachFile(c.r.Context(), pkg, *file)
	if err != nil {
		s.apiError(c.w, c.r, err)
		return
	}
	writeJSON(c.w, http.StatusOK, s.attachedJSON(af))
}

func (s *Server) apiPackageDetach(c *apiContext, pkg *model.Package) {
	if !c.form.has("attached_file_id") {
		writeJSON(c.w, http.StatusBadRequest, map[string]string{"error": "attached_file_id required"})
		return
	}
	id := c.form.int("attached_file_id", 0)
	af, err := model.GetAttachedFile(c.r.Context(), s.Core.DB, id)
	if err != nil {
		s.apiError(c.w, c.r, err)
		return
	}
	if af == nil || af.PackageID != pkg.ID {
		writeJSON(c.w, http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("the attached file (id=%d) not found in the package (id=%d)", id, pkg.ID)})
		return
	}
	if err := s.Core.DetachFile(c.r.Context(), af); err != nil {
		s.apiError(c.w, c.r, err)
		return
	}
	writeJSON(c.w, http.StatusOK, s.attachedJSON(af))
}
