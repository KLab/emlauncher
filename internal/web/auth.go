package web

import (
	"errors"
	"net/http"
	"strings"

	"emlauncher/internal/service"
)

// requireLogin はログインしていなければログイン画面にリダイレクトする.
// GET なら開こうとした URL を覚えておき, ログイン後にそこへ戻る.
func (s *Server) requireLogin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := s.sessions.load(r)
		if sess.Mail == "" {
			s.redirectToLogin(w, r, sess)
			return
		}
		next.ServeHTTP(w, withUser(r, sess.Mail))
	})
}

func (s *Server) redirectToLogin(w http.ResponseWriter, r *http.Request, sess *session) {
	scheme := ""
	if s.cfg.EnableHTTPS {
		scheme = "https"
	}
	if r.Method == http.MethodGet {
		sess.URLBeforeLogin = absURL(r, scheme, r.URL.RequestURI())
	} else {
		sess.URLBeforeLogin = ""
	}
	s.sessions.save(w, r, sess)
	http.Redirect(w, r, absURL(r, scheme, "/login"), http.StatusFound)
}

// requireLoginJSON は ajax 用. ログインしていなければ 403 の JSON を返す.
func (s *Server) requireLoginJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mail := s.sessions.load(r).Mail
		if mail == "" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "login required"})
			return
		}
		next.ServeHTTP(w, withUser(r, mail))
	})
}

// redirectIfLoggedIn はログイン済みならトップにリダイレクトする (ログイン画面用).
func (s *Server) redirectIfLoggedIn(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.sessions.load(r).Mail != "" {
			redirect(w, r, "/")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// login はセッションをログイン状態にし, ログイン前に開こうとした URL (無ければトップ) にリダイレクトする.
func (s *Server) login(w http.ResponseWriter, r *http.Request, mail string) error {
	sess := s.sessions.load(r)
	back := sess.URLBeforeLogin
	if back == "" || !s.sameSite(r, back) {
		back = "/"
	}
	// ログインのたびにセッションを作り直す (発行時刻も新しくする)
	s.sessions.save(w, r, &session{Mail: mail})
	return redirect(w, r, back)
}

// sameSite は URL がこのサイトのものか (外部へのリダイレクトに使わせない).
func (s *Server) sameSite(r *http.Request, u string) bool {
	host := requestHost(r)
	return strings.HasPrefix(u, "http://"+host+"/") || strings.HasPrefix(u, "https://"+host+"/")
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) error {
	s.sessions.save(w, r, &session{})
	return redirect(w, r, "/login")
}

type loginData struct {
	EnablePassword    bool
	EnableGoogleAuth  bool
	EnableAzureADAuth bool
}

func (s *Server) loginIndex(w http.ResponseWriter, r *http.Request) error {
	if s.cfg.EnableHTTPS && requestScheme(r) != "https" {
		return redirect(w, r, absURL(r, "https", "/login"))
	}
	return s.renderOK(w, r, "login/index", loginData{
		EnablePassword:    s.cfg.Login.EnablePassword,
		EnableGoogleAuth:  s.cfg.Login.EnableGoogleAuth,
		EnableAzureADAuth: s.cfg.Login.EnableAzureADAuth,
	})
}

func (s *Server) loginPassword(w http.ResponseWriter, r *http.Request) error {
	if !s.cfg.Login.EnablePassword {
		return errNotFound
	}
	f := formOf(r)
	mail := f.Get("email")
	ok, err := s.svc.Authenticate(r.Context(), mail, f.Get("password"))
	if err != nil {
		return err
	}
	if !ok {
		return errorStatus(http.StatusUnauthorized, "invalid email or password")
	}
	return s.login(w, r, mail)
}

func (s *Server) passwordReminder(w http.ResponseWriter, r *http.Request) error {
	return s.renderOK(w, r, "login/password_reminder", nil)
}

func (s *Server) passwordConfirm(w http.ResponseWriter, r *http.Request) error {
	err := s.svc.SendPasswordReset(r.Context(), formOf(r).Get("email"), func(key string) string {
		return baseURL(r) + "/login/password_reset?key=" + key
	})
	if err != nil && !errors.Is(err, service.ErrNotFound) {
		return err
	}
	return s.renderOK(w, r, "login/password_confirm", struct{ Error bool }{Error: err != nil})
}

func (s *Server) passwordReset(w http.ResponseWriter, r *http.Request) error {
	key := formOf(r).Get("key")
	if _, err := s.svc.PasswordResetMail(r.Context(), key); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return badRequest("invalid key")
		}
		return err
	}
	return s.renderOK(w, r, "login/password_reset", struct{ Key string }{Key: key})
}

func (s *Server) passwordCommit(w http.ResponseWriter, r *http.Request) error {
	f := formOf(r)
	if err := s.svc.ResetPassword(r.Context(), f.Get("key"), f.Get("password")); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return badRequest("invalid key")
		}
		return err
	}
	return s.renderOK(w, r, "login/password_commit", nil)
}
