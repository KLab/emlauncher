// Package web は EMLauncher の HTTP のハンドラ (画面, API, ajax).
//
// URL は PHP 版と同じ "/{module}/{action}?..." で, 対応は Handler の中に列挙している.
// ハンドラは error を返し, page (画面) / api (JSON) がエラーを応答にする.
package web

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/KLab/emlauncher-go/internal/config"
	"github.com/KLab/emlauncher-go/internal/pkgfile"
	"github.com/KLab/emlauncher-go/internal/service"
	"github.com/KLab/emlauncher-go/internal/storage"
)

type Server struct {
	svc        *service.Service
	cfg        *config.Config
	sessions   *sessionCodec
	templates  templates
	static     fs.FS
	localFiles *storage.Local // LocalFile を配信する場合
	bundletool *pkgfile.Bundletool
	oauth      map[string]*oauthProvider
	log        *slog.Logger
}

type Options struct {
	Service *service.Service
	Config  *config.Config
	// Templates は templates/ の中身, Static は static/ の中身.
	Templates fs.FS
	Static    fs.FS
	// LocalFiles を指定すると, その URL の接頭辞以下を配信する.
	LocalFiles *storage.Local
	Bundletool *pkgfile.Bundletool
	Logger     *slog.Logger
}

func New(o Options) (*Server, error) {
	ts, err := loadTemplates(o.Templates)
	if err != nil {
		return nil, err
	}
	return &Server{
		svc:        o.Service,
		cfg:        o.Config,
		sessions:   newSessionCodec(o.Config.Session.Secret),
		templates:  ts,
		static:     o.Static,
		localFiles: o.LocalFiles,
		bundletool: o.Bundletool,
		oauth:      oauthProviders(o.Config.Login),
		log:        o.Logger,
	}, nil
}

// Handler は URL とハンドラの対応.
func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(s.accessLog)
	r.Use(s.canonicalPath)
	r.NotFound(notFoundText)
	r.MethodNotAllowed(notFoundText)

	// 画面・API・ajax は PHP 版と同じくメソッドを問わない (HEAD は GET と同じ処理で本文を返さない).
	// POST で送られることが前提の値 (削除のトークンなど) はハンドラで本文から読む.
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("OK")) })
	static := http.FileServer(http.FS(s.static))
	r.Handle("/css/*", static)
	r.Handle("/favicon.ico", static)
	r.Handle("/apple-touch-icon.png", static)
	if s.localFiles != nil {
		r.Handle(s.localFiles.URLPrefix()+"/*", s.localFiles.Handler())
	}

	r.Route("/api", func(r chi.Router) {
		r.Use(s.withForm(s.jsonError))
		r.Use(s.requireAPIKey)
		r.NotFound(s.api(func(http.ResponseWriter, *http.Request) error {
			return errorStatus(http.StatusNotFound, "404 Not Found")
		}))
		r.HandleFunc("/upload", s.api(s.apiUpload))
		r.HandleFunc("/package_list", s.api(s.apiPackageList))
		r.HandleFunc("/delete", s.api(s.apiDelete))
		r.HandleFunc("/create_token", s.api(s.apiCreateToken))
		r.Group(func(r chi.Router) {
			r.Use(s.requireAPIPackage)
			r.HandleFunc("/package_edit", s.api(s.apiPackageEdit))
			r.HandleFunc("/package_attach", s.api(s.apiPackageAttach))
			r.HandleFunc("/package_detach", s.api(s.apiPackageDetach))
			r.HandleFunc("/package_attach_download", s.api(s.apiPackageAttachDownload))
		})
	})
	r.Route("/ajax", func(r chi.Router) {
		r.Use(s.withForm(s.jsonError))
		r.Use(s.requireLoginJSON)
		r.NotFound(s.api(func(http.ResponseWriter, *http.Request) error {
			return errorStatus(http.StatusNotFound, "404 Not Found")
		}))
		r.HandleFunc("/notification_setting", s.api(s.ajaxNotificationSetting))
		r.HandleFunc("/upload_package_temporary", s.api(s.ajaxUploadTemporary))
	})

	r.Group(func(r chi.Router) {
		r.Use(s.withForm(s.pageError))

		// ログイン不要
		r.HandleFunc("/qr/code", s.qrCode)
		r.Route("/login", func(r chi.Router) {
			r.Use(s.redirectIfLoggedIn)
			r.HandleFunc("/", s.page(s.loginIndex))
			r.HandleFunc("/password", s.page(s.loginPassword))
			r.HandleFunc("/password_reminder", s.page(s.passwordReminder))
			r.HandleFunc("/password_confirm", s.page(s.passwordConfirm))
			r.HandleFunc("/password_reset", s.page(s.passwordReset))
			r.HandleFunc("/password_commit", s.page(s.passwordCommit))
			for name := range s.oauth {
				r.HandleFunc("/"+name, s.page(s.oauthStart(name)))
				r.HandleFunc("/"+name+"_callback", s.page(s.oauthCallback(name)))
				r.HandleFunc("/"+name+"_error", s.page(s.oauthError(name)))
			}
		})
		// トークンで認証する (インストーラやゲストはセッションを持たない)
		r.HandleFunc("/package/install", s.page(s.packageInstall))
		r.HandleFunc("/package/install_plist", s.page(s.installPlist))
		r.HandleFunc("/package/guestpass_install", s.page(s.guestpassInstall))

		// ログインが必要な画面
		r.Group(func(r chi.Router) {
			r.Use(s.requireLogin)
			r.HandleFunc("/", s.page(s.top))
			r.HandleFunc("/logout", s.page(s.logout))
			r.Route("/app", func(r chi.Router) {
				r.HandleFunc("/new", s.page(s.appNew))
				r.HandleFunc("/create", s.page(s.appCreate))
				r.HandleFunc("/", s.page(s.appIndex))
				r.HandleFunc("/comment", s.page(s.appComment))
				r.HandleFunc("/comment_post", s.page(s.appCommentPost))
				r.Group(func(r chi.Router) {
					r.Use(s.requireAppOwner)
					r.HandleFunc("/upload", s.page(s.appUpload))
					r.HandleFunc("/upload_post", s.page(s.appUploadPost))
					r.HandleFunc("/preferences", s.page(s.appPreferences))
					r.HandleFunc("/preferences_update", s.page(s.appPreferencesUpdate))
					r.HandleFunc("/preferences_refresh_apikey", s.page(s.appRefreshAPIKey))
					r.HandleFunc("/preferences_delete_tags", s.page(s.appDeleteTags))
					r.HandleFunc("/preferences_update_owners", s.page(s.appUpdateOwners))
				})
			})
			r.Route("/package", func(r chi.Router) {
				r.HandleFunc("/", s.page(s.packageIndex))
				r.HandleFunc("/create_token", s.page(s.packageCreateToken))
				r.HandleFunc("/attach_download", s.page(s.packageAttachDownload))
				r.HandleFunc("/create_guestpass", s.page(s.packageCreateGuestPass))
				r.HandleFunc("/guestpass", s.page(s.packageGuestPass))
				r.HandleFunc("/expire_guestpass", s.page(s.packageExpireGuestPass))
				r.Group(func(r chi.Router) {
					r.Use(s.requirePackageOwner)
					r.HandleFunc("/edit", s.page(s.packageEdit))
					r.HandleFunc("/edit_commit", s.page(s.packageEditCommit))
					r.HandleFunc("/delete_confirm", s.page(s.packageDeleteConfirm))
					r.HandleFunc("/delete", s.page(s.packageDelete))
					r.HandleFunc("/attach", s.page(s.packageAttach))
					r.HandleFunc("/attach_delete_confirm", s.page(s.packageAttachDeleteConfirm))
					r.HandleFunc("/attach_delete", s.page(s.packageAttachDelete))
				})
			})
			r.Route("/myapps", func(r chi.Router) {
				r.HandleFunc("/installed", s.page(s.myappsInstalled))
				r.HandleFunc("/own", s.page(s.myappsOwn))
				r.HandleFunc("/delete", s.page(s.myappsDelete))
			})
			r.HandleFunc("/guestpass/list", s.page(s.guestpassList))
			r.HandleFunc("/doc/{page}", s.page(s.doc))
		})
	})
	return r
}

// notFoundText は存在しない URL の応答 (PHP 版と同じプレーンテキスト).
func notFoundText(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("404 Not Found"))
}

// canonicalPath は URL のパスを "/{module}/{action}" の形に揃える (PHP 版の PATH_INFO での振り分けと同じ).
// "/app", "/app/", "/app/index" は同じで, action の後のパスと末尾の "/" は無視する.
// "/", "/top", "/top/index" はトップ. 静的ファイルと LocalFile の配信はそのまま.
func (s *Server) canonicalPath(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if s.passThrough(p) {
			next.ServeHTTP(w, r)
			return
		}
		module, action := moduleAction(p)
		switch {
		case module == "top" && action == "index":
			p = "/"
		case action == "index":
			p = "/" + module
		default:
			p = "/" + module + "/" + action
		}
		if p != r.URL.Path {
			u := *r.URL
			u.Path, u.RawPath = p, ""
			r2 := r.Clone(r.Context())
			r2.URL = &u
			r = r2
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) passThrough(p string) bool {
	if strings.HasPrefix(p, "/css/") || p == "/favicon.ico" || p == "/apple-touch-icon.png" {
		return true
	}
	return s.localFiles != nil && strings.HasPrefix(p, s.localFiles.URLPrefix()+"/")
}

// accessLog はリクエストを 1 行ずつログに出す. ハンドラが logAttr で足した項目
// (PHP 版の apache_log と同じ user, app_id, pkg_id など) も出す.
func (s *Server) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		uri := r.URL.RequestURI()
		la := &logAttrs{}
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r.WithContext(context.WithValue(r.Context(), ctxLog, la)))
		args := []any{"method", r.Method, "uri", uri, "status", ww.Status(),
			"bytes", ww.BytesWritten(), "duration", time.Since(start).Round(time.Millisecond),
			"remote", remoteIP(r), "user_agent", r.UserAgent()}
		user, extra := la.get()
		if user == "" {
			user = s.sessions.load(r).Mail
		}
		args = append(args, "user", user)
		s.log.Info("access", append(args, extra...)...)
	})
}

// logAttrs はアクセスログに足す項目.
type logAttrs struct {
	mu    sync.Mutex
	user  string
	attrs []any
}

func (la *logAttrs) get() (string, []any) {
	la.mu.Lock()
	defer la.mu.Unlock()
	return la.user, la.attrs
}

// logAttr はアクセスログに項目 (app_id, pkg_id, attached_id, guestpass_id, platform) を足す.
// 同じ項目は後の値で置き換える.
func logAttr(r *http.Request, key string, value any) {
	la, ok := r.Context().Value(ctxLog).(*logAttrs)
	if !ok {
		return
	}
	la.mu.Lock()
	defer la.mu.Unlock()
	for i := 0; i+1 < len(la.attrs); i += 2 {
		if la.attrs[i] == key {
			la.attrs[i+1] = value
			return
		}
	}
	la.attrs = append(la.attrs, key, value)
}

// logUser はアクセスログのユーザ (トークンでのインストールではトークンのユーザ).
func logUser(r *http.Request, mail string) {
	if la, ok := r.Context().Value(ctxLog).(*logAttrs); ok {
		la.mu.Lock()
		la.user = mail
		la.mu.Unlock()
	}
}
