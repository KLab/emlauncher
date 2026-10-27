// EMLauncher の HTTP ハンドラ.
//
// URL とリクエストパラメータ名は PHP 版と同じ (/{module}/{action}?...).

package emlauncher

import (
	"log/slog"
	"net/http"

	"emlauncher/storage"
)

type Server struct {
	Config *Config
	Core   *Core
	Log    *slog.Logger

	templates  templates
	routeTable map[string]uiRoute
	sessions   *SessionStore
}

// Init はテンプレートの読み込みなどを行う. Handler より前に呼ぶ.
func (s *Server) Init() error {
	var err error
	if s.templates, err = loadTemplates(); err != nil {
		return err
	}
	s.sessions, err = NewSessionStore(s.Config.Session.Secret, func(r *http.Request) bool {
		return requestScheme(r) == "https"
	})
	if err != nil {
		return err
	}
	s.routeTable = s.uiRoutes()
	return nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.health)

	if l, ok := s.Core.Storage.(*storage.Local); ok && s.Config.LocalFile.Serve {
		mux.Handle(l.URLPrefix()+"/", l.Handler())
	}

	s.routes(mux)
	return s.accessLog(mux)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("OK"))
}
