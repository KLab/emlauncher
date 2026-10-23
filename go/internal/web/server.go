// Package web は EMLauncher の HTTP ハンドラ.
//
// URL とリクエストパラメータ名は PHP 版と同じ (/{module}/{action}?...).
package web

import (
	"log/slog"
	"net/http"

	"github.com/jmoiron/sqlx"

	"github.com/KLabTools/emlauncher/go/internal/cache"
	"github.com/KLabTools/emlauncher/go/internal/clock"
	"github.com/KLabTools/emlauncher/go/internal/config"
	"github.com/KLabTools/emlauncher/go/internal/storage"
)

type Server struct {
	Config  *config.Config
	DB      *sqlx.DB
	Cache   *cache.Cache
	Storage storage.Storage
	Clock   *clock.Clock
	Log     *slog.Logger
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.health)

	if l, ok := s.Storage.(*storage.Local); ok && s.Config.LocalFile.Serve {
		mux.Handle(l.URLPrefix()+"/", l.Handler())
	}

	s.routes(mux)
	return s.accessLog(mux)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("OK"))
}
