package web

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/KLab/emlauncher-go/internal/config"
	"github.com/KLab/emlauncher-go/internal/randstr"
)

// oauthProvider は Google / AzureAD の OAuth 2.0 (認可コード) でのログイン.
// コールバック URL は PHP 版と同じ /login/{name}_callback.
type oauthProvider struct {
	label        string // エラー画面の表示名
	authURL      string
	tokenURL     string
	userInfoURL  string
	scope        string
	clientID     string
	clientSecret string
	// mailField はユーザ情報の JSON のメールアドレスの項目.
	mailField string
	// authParams は認可のリクエストに加えるパラメータ.
	authParams url.Values
	// tokenScope ならトークンのリクエストにも scope を付ける.
	tokenScope bool
}

func oauthProviders(c config.Login) map[string]*oauthProvider {
	m := map[string]*oauthProvider{}
	if c.EnableGoogleAuth {
		m["google"] = &oauthProvider{
			label:        "Google",
			authURL:      "https://accounts.google.com/o/oauth2/auth",
			tokenURL:     "https://accounts.google.com/o/oauth2/token",
			userInfoURL:  "https://www.googleapis.com/oauth2/v1/userinfo",
			scope:        "https://www.googleapis.com/auth/userinfo.email",
			clientID:     c.GoogleAppID,
			clientSecret: c.GoogleAppSecret,
			mailField:    "email",
			authParams:   url.Values{"approval_prompt": {"auto"}},
		}
	}
	if c.EnableAzureADAuth {
		m["azuread"] = &oauthProvider{
			label:        "AzureAD",
			authURL:      "https://login.microsoftonline.com/organizations/oauth2/v2.0/authorize",
			tokenURL:     "https://login.microsoftonline.com/organizations/oauth2/v2.0/token",
			userInfoURL:  "https://graph.microsoft.com/v1.0/me",
			scope:        "https://graph.microsoft.com/User.Read",
			clientID:     c.AzureADAppID,
			clientSecret: c.AzureADAppSecret,
			mailField:    "userPrincipalName",
			tokenScope:   true,
		}
	}
	return m
}

func callbackURL(r *http.Request, name string) string {
	return baseURL(r) + "/login/" + name + "_callback"
}

// oauthStart は認可の画面にリダイレクトする. CSRF 対策の state をセッションに覚えておく.
func (s *Server) oauthStart(name string) handlerFunc {
	p := s.oauth[name]
	return func(w http.ResponseWriter, r *http.Request) error {
		sess := s.sessions.load(r)
		sess.OAuthState = randstr.New(32)
		s.sessions.save(w, r, sess)
		q := url.Values{
			"client_id":     {p.clientID},
			"redirect_uri":  {callbackURL(r, name)},
			"scope":         {p.scope},
			"response_type": {"code"},
			"state":         {sess.OAuthState},
		}
		for k, v := range p.authParams {
			q[k] = v
		}
		return redirect(w, r, p.authURL+"?"+q.Encode())
	}
}

func (s *Server) oauthCallback(name string) handlerFunc {
	p := s.oauth[name]
	return func(w http.ResponseWriter, r *http.Request) error {
		f := formOf(r)
		sess := s.sessions.load(r)
		state := sess.OAuthState
		sess.OAuthState = ""
		var mail string
		if state != "" && subtle.ConstantTimeCompare([]byte(state), []byte(f.Get("state"))) == 1 && f.Get("code") != "" {
			var err error
			mail, err = p.userMail(r.Context(), f.Get("code"), callbackURL(r, name))
			if err != nil {
				s.log.Warn("oauth login failed", "provider", name, "err", err)
			}
		}
		ok, err := s.svc.CanLogin(r.Context(), mail)
		if err != nil {
			return err
		}
		if !ok {
			sess.FailedMail = mail
			s.sessions.save(w, r, sess)
			return redirect(w, r, "/login/"+name+"_error")
		}
		return s.login(w, r, mail)
	}
}

func (s *Server) oauthError(name string) handlerFunc {
	p := s.oauth[name]
	return func(w http.ResponseWriter, r *http.Request) error {
		return s.renderOK(w, r, "login/oauth_error", struct{ Provider, Mail string }{p.label, s.sessions.load(r).FailedMail})
	}
}

var oauthClient = &http.Client{Timeout: 10 * time.Second}

// userMail は認可コードをアクセストークンに換え, ユーザ情報からメールアドレスを取り出す.
func (p *oauthProvider) userMail(ctx context.Context, code, redirectURI string) (string, error) {
	form := url.Values{
		"code":          {code},
		"client_id":     {p.clientID},
		"client_secret": {p.clientSecret},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	}
	if p.tokenScope {
		form.Set("scope", p.scope)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var token struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := doJSON(req, &token); err != nil {
		return "", fmt.Errorf("token: %w", err)
	}
	if token.AccessToken == "" {
		return "", fmt.Errorf("token: no access_token")
	}
	if token.TokenType == "" {
		token.TokenType = "Bearer"
	}
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, p.userInfoURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", token.TokenType+" "+token.AccessToken)
	var info map[string]any
	if err := doJSON(req, &info); err != nil {
		return "", fmt.Errorf("userinfo: %w", err)
	}
	mail, _ := info[p.mailField].(string)
	return mail, nil
}

func doJSON(req *http.Request, v any) error {
	res, err := oauthClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d: %s", res.StatusCode, b)
	}
	return json.Unmarshal(b, v)
}
