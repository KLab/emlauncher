package web

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// templates は画面ごとのテンプレート. キーは "app/index" のような templates/ からの相対パス.
// 各画面は layout.html と blocks.html に, 画面のファイル (content を定義する) を加えたもの.
type templates map[string]*template.Template

const errorTemplate = "_error"

var templateFuncs = template.FuncMap{
	"mod": func(a, b int) int { return a % b },
	// nl2br は改行の前に <br /> を入れる (PHP の nl2br(htmlspecialchars(...)) と同じ).
	"nl2br": func(s string) template.HTML {
		return template.HTML(strings.ReplaceAll(template.HTMLEscapeString(s), "\n", "<br />\n"))
	},
}

func loadTemplates(fsys fs.FS) (templates, error) {
	base, err := template.New("").Funcs(templateFuncs).ParseFS(fsys, "layout.html", "blocks.html")
	if err != nil {
		return nil, err
	}
	ts := templates{}
	err = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.Contains(p, "/") || path.Ext(p) != ".html" {
			return err
		}
		b, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		t, err := template.Must(base.Clone()).Parse(string(b))
		if err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		ts[strings.TrimSuffix(p, ".html")] = t
		return nil
	})
	if err != nil {
		return nil, err
	}
	ts[errorTemplate] = template.Must(template.Must(base.Clone()).Parse(`{{define "content"}}{{template "error" .}}{{end}}`))
	return ts, nil
}

// page はテンプレートに渡すデータ. Data は画面ごとのデータ.
type page struct {
	TitlePrefix string
	LoginUser   string
	Module      string
	Action      string
	Data        any

	base string
}

// URL は path をブラウザから見た絶対 URL にする.
func (p *page) URL(path string) string { return p.base + path }

// QR は path (サイトの中のパス) を指す QR コードの画像の URL.
func (p *page) QR(path string, size int) string {
	return fmt.Sprintf("%s/qr/code?s=%d&q=%s", p.base, size, url.QueryEscape(path))
}

// moduleAction は URL のパス "/{module}/{action}" を分ける. 省略されたら top / index.
func moduleAction(urlPath string) (module, action string) {
	parts := strings.Split(strings.Trim(urlPath, "/"), "/")
	module, action = parts[0], "index"
	if module == "" {
		module = "top"
	}
	if len(parts) >= 2 && parts[1] != "" {
		action = parts[1]
	}
	return module, action
}

// render は画面 name ("app/index" など) を status で返す.
func (s *Server) render(w http.ResponseWriter, r *http.Request, status int, name string, data any) error {
	t, ok := s.templates[name]
	if !ok {
		return fmt.Errorf("template %s not found", name)
	}
	module, action := moduleAction(r.URL.Path)
	user := loginUser(r)
	if user == "" {
		// ログインが不要な画面やエラーページでも, ログインしていればヘッダに表示する
		user = s.sessions.load(r).Mail
	}
	p := &page{
		TitlePrefix: s.cfg.TitlePrefix,
		LoginUser:   user,
		Module:      module,
		Action:      action,
		Data:        data,
		base:        baseURL(r),
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "layout", p); err != nil {
		return fmt.Errorf("render %s: %w", name, err)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, err := buf.WriteTo(w)
	return err
}

func (s *Server) renderOK(w http.ResponseWriter, r *http.Request, name string, data any) error {
	return s.render(w, r, http.StatusOK, name, data)
}

// renderError はエラーページを返す.
func (s *Server) renderError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	if err := s.render(w, r, status, errorTemplate, msg); err != nil {
		s.log.Error("render error page", "err", err)
		http.Error(w, msg, status)
	}
}
