package web

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// アクセスログに載せる追加情報 (PHP 版の apache_log() に相当).
type logAttrs struct {
	mu    sync.Mutex
	attrs []slog.Attr
}

type logAttrsKey struct{}

// logAttr はアクセスログに項目を追加する (user, app_id, pkg_id など).
func logAttr(r *http.Request, key string, value any) {
	if la, ok := r.Context().Value(logAttrsKey{}).(*logAttrs); ok {
		la.mu.Lock()
		la.attrs = append(la.attrs, slog.Any(key, value))
		la.mu.Unlock()
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
	size   int64
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.size += int64(n)
	return n, err
}

func (s *Server) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		la := &logAttrs{}
		r = r.WithContext(context.WithValue(r.Context(), logAttrsKey{}, la))
		sw := &statusWriter{ResponseWriter: w}
		defer func() {
			if v := recover(); v != nil {
				s.Log.Error("panic", "method", r.Method, "uri", r.URL.RequestURI(), "panic", v)
				if sw.status == 0 {
					http.Error(sw, "500 Internal Server Error", http.StatusInternalServerError)
				}
			}
			attrs := []slog.Attr{
				slog.String("method", r.Method),
				slog.String("uri", r.URL.RequestURI()),
				slog.Int("status", sw.status),
				slog.Int64("size", sw.size),
				slog.Duration("duration", time.Since(start)),
				slog.String("remote", remoteAddr(r)),
				slog.String("ua", r.UserAgent()),
			}
			la.mu.Lock()
			attrs = append(attrs, la.attrs...)
			la.mu.Unlock()
			s.Log.LogAttrs(r.Context(), slog.LevelInfo, "access", attrs...)
		}()
		next.ServeHTTP(sw, r)
	})
}

// remoteAddr はクライアントの IP. ロードバランサ配下では X-Forwarded-For の先頭.
func remoteAddr(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	host := r.RemoteAddr
	for i := len(host) - 1; i >= 0; i-- {
		if host[i] == ':' {
			return host[:i]
		}
	}
	return host
}
