package web

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
)

//go:embed templates static
var assets embed.FS

// page はテンプレートに渡すデータ. 各ページ固有のデータは Data.
type page struct {
	r           *http.Request
	TitlePrefix string
	LoginUser   string
	Module      string
	Action      string
	Data        any
}

// URL は絶対 URL (PHP 版テンプレートの url()).
func (p *page) URL(path string) string { return absURL(p.r, path, "") }

// QR は path を指す QR コード画像の URL.
func (p *page) QR(path string, size int) string {
	return fmt.Sprintf("%s?s=%d&q=%s", p.URL("/qr/code"), size, url.QueryEscape(path))
}

var funcs = template.FuncMap{
	"mod": func(a, b int) int { return a % b },
	// nl2br は改行を <br> にする (HTML はエスケープする).
	"nl2br": func(s string) template.HTML {
		esc := template.HTMLEscapeString(s)
		esc = strings.ReplaceAll(esc, "\r\n", "\n")
		return template.HTML(strings.ReplaceAll(esc, "\n", "<br />\n"))
	},
}

type templates map[string]*template.Template

// loadTemplates は templates/ 以下の各ページを, レイアウトとブロックと一緒に読み込む.
func loadTemplates() (templates, error) {
	ts := templates{}
	err := fs.WalkDir(assets, "templates", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		name := strings.TrimSuffix(strings.TrimPrefix(p, "templates/"), ".html")
		if name == "layout" || name == "blocks" {
			return nil
		}
		t, err := template.New(name).Funcs(funcs).ParseFS(assets, "templates/layout.html", "templates/blocks.html", p)
		if err != nil {
			return err
		}
		ts[name] = t
		return nil
	})
	if err != nil {
		return nil, err
	}
	// エラーページ
	t, err := template.New("error").Funcs(funcs).ParseFS(assets, "templates/layout.html", "templates/blocks.html")
	if err != nil {
		return nil, err
	}
	if _, err := t.New("content").Parse(`{{template "error" .}}`); err != nil {
		return nil, err
	}
	ts["_error"] = t
	return ts, nil
}

// render はページを描画する. name は templates/ 以下のパス (拡張子無し).
func (s *Server) render(c *uiContext, status int, name string, data any) {
	t, ok := s.templates[name]
	if !ok {
		panic("template not found: " + name)
	}
	p := &page{
		r:           c.r,
		TitlePrefix: s.Config.TitlePrefix,
		LoginUser:   c.user,
		Module:      c.module,
		Action:      c.action,
		Data:        data,
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "layout", p); err != nil {
		s.Log.Error("render", "template", name, "error", err)
		http.Error(c.w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	c.w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	c.w.WriteHeader(status)
	c.w.Write(buf.Bytes())
}

// errorPage はエラーページ. PHP 版と同じメッセージ ("Not Found", "Permission Denied" など) を表示する.
func (s *Server) errorPage(c *uiContext, status int, message string) {
	s.render(c, status, "_error", message)
}

// staticHandler は埋め込んだ静的ファイル (CSS, favicon) を返す.
func staticHandler() http.Handler {
	sub, _ := fs.Sub(assets, "static")
	fsrv := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=86400")
		fsrv.ServeHTTP(w, r)
	})
}
