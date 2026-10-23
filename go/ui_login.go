package emlauncher

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const passwordResetTTL = 30 * time.Minute

type loginView struct {
	EnablePassword    bool
	EnableGoogleAuth  bool
	EnableAzureADAuth bool
}

func (s *Server) loginIndex(c *uiContext) {
	if s.Config.EnableHTTPS && requestScheme(c.r) == "http" {
		redirect(c.w, c.r, absURL(c.r, "/login", "https"))
		return
	}
	l := s.Config.Login
	s.render(c, http.StatusOK, "login/index", loginView{l.EnablePassword, l.EnableGoogleAuth, l.EnableAzureADAuth})
}

// login はログイン状態にして, ログイン前に開こうとしていた URL へリダイレクトする.
func (s *Server) login(c *uiContext, mailAddr string) {
	c.sess.Set(sesLoginUser, mailAddr)
	u := c.sess.Get(sesURLBeforeLogin)
	if u == "" {
		u = "/"
	}
	redirect(c.w, c.r, u)
}

func (s *Server) logout(c *uiContext) {
	c.sess.Delete(sesLoginUser)
	c.sess.Delete(sesURLBeforeLogin)
	redirect(c.w, c.r, "/login")
}

func (s *Server) loginPassword(c *uiContext) {
	if !s.Config.Login.EnablePassword {
		s.errorPage(c, http.StatusForbidden, "password login is disabled")
		return
	}
	mailAddr := c.form.str("email")
	up, err := GetUserPass(c.r.Context(), s.Core.DB, mailAddr)
	if err != nil {
		s.serverError(c, err)
		return
	}
	if up == nil || !CheckPassword(up.Passhash.String, c.form.str("password")) {
		s.errorPage(c, http.StatusUnauthorized, "invalid email or password")
		return
	}
	s.login(c, mailAddr)
}

func (s *Server) loginPasswordReminder(c *uiContext) {
	s.render(c, http.StatusOK, "login/password_reminder", nil)
}

type resetData struct {
	Mail string `json:"mail"`
}

func (s *Server) loginPasswordConfirm(c *uiContext) {
	ctx := c.r.Context()
	up, err := GetUserPass(ctx, s.Core.DB, c.form.str("email"))
	if err != nil {
		s.serverError(c, err)
		return
	}
	if up == nil {
		s.render(c, http.StatusOK, "login/password_confirm", map[string]bool{"Error": true})
		return
	}
	key := randomHex(20)
	b, _ := json.Marshal(resetData{Mail: up.Mail})
	if err := s.Core.Cache.Set(key, b, passwordResetTTL); err != nil {
		s.serverError(c, err)
		return
	}
	err = s.Core.Mail.Send(MailMessage{
		From:    s.Config.MailSender,
		To:      []string{up.Mail},
		Subject: "Reset password",
		Body:    "EMLauncher password reset URL:\n" + absURL(c.r, "/login/password_reset?key="+key, "") + "\n",
	})
	if err != nil {
		s.serverError(c, err)
		return
	}
	s.render(c, http.StatusOK, "login/password_confirm", map[string]bool{"Error": false})
}

// resetUser はパスワードリセットのキーに対応するユーザ. 無効なキーなら nil.
func (s *Server) resetUser(ctx context.Context, key string) (*UserPass, error) {
	if key == "" {
		return nil, nil
	}
	b, ok, err := s.Core.Cache.Get(key)
	if err != nil || !ok {
		return nil, err
	}
	var d resetData
	if json.Unmarshal(b, &d) != nil || d.Mail == "" {
		return nil, nil // PHP 版が作ったキー (PHP のシリアライズ形式) は使えない
	}
	return GetUserPass(ctx, s.Core.DB, d.Mail)
}

func (s *Server) loginPasswordReset(c *uiContext) {
	key := c.form.str("key")
	up, err := s.resetUser(c.r.Context(), key)
	if err != nil {
		s.serverError(c, err)
		return
	}
	if up == nil {
		s.errorPage(c, http.StatusNotFound, "invalid key")
		return
	}
	s.render(c, http.StatusOK, "login/password_reset", map[string]string{"Key": key})
}

func (s *Server) loginPasswordCommit(c *uiContext) {
	ctx := c.r.Context()
	key := c.form.str("key")
	up, err := s.resetUser(ctx, key)
	if err != nil {
		s.serverError(c, err)
		return
	}
	password := c.form.str("password")
	if up == nil || password == "" {
		s.errorPage(c, http.StatusNotFound, "invalid key")
		return
	}
	if err := up.UpdatePasshash(ctx, s.Core.DB, HashPassword(password)); err != nil {
		s.serverError(c, err)
		return
	}
	s.Core.Cache.Delete(key)
	s.render(c, http.StatusOK, "login/password_commit", nil)
}

// OAuth (Google / AzureAD). PHP 版と同じエンドポイント・コールバック URL.

type oauthProvider struct {
	name      string
	authURL   string
	tokenURL  string
	userURL   string
	scope     string
	mailField string
	authExtra url.Values
}

var (
	googleOAuth = oauthProvider{
		name:      "google",
		authURL:   "https://accounts.google.com/o/oauth2/auth",
		tokenURL:  "https://accounts.google.com/o/oauth2/token",
		userURL:   "https://www.googleapis.com/oauth2/v1/userinfo",
		scope:     "https://www.googleapis.com/auth/userinfo.email",
		mailField: "email",
		authExtra: url.Values{"approval_prompt": {"auto"}},
	}
	azureADOAuth = oauthProvider{
		name:      "azuread",
		authURL:   "https://login.microsoftonline.com/organizations/oauth2/v2.0/authorize",
		tokenURL:  "https://login.microsoftonline.com/organizations/oauth2/v2.0/token",
		userURL:   "https://graph.microsoft.com/v1.0/me",
		scope:     "https://graph.microsoft.com/User.Read",
		mailField: "userPrincipalName",
	}
)

func (s *Server) oauthCredentials(p oauthProvider) (enabled bool, id, secret string) {
	l := s.Config.Login
	if p.name == "google" {
		return l.EnableGoogleAuth, l.GoogleAppID, l.GoogleAppSecret
	}
	return l.EnableAzureADAuth, l.AzureADAppID, l.AzureADAppSecret
}

func (s *Server) loginGoogle(c *uiContext)          { s.oauthStart(c, googleOAuth) }
func (s *Server) loginGoogleCallback(c *uiContext)  { s.oauthCallback(c, googleOAuth) }
func (s *Server) loginAzureAD(c *uiContext)         { s.oauthStart(c, azureADOAuth) }
func (s *Server) loginAzureADCallback(c *uiContext) { s.oauthCallback(c, azureADOAuth) }

func (s *Server) oauthStart(c *uiContext, p oauthProvider) {
	enabled, id, _ := s.oauthCredentials(p)
	if !enabled {
		s.errorPage(c, http.StatusForbidden, p.name+" login is disabled")
		return
	}
	state := randomString(32)
	c.sess.Set("oauth_state", state)
	q := url.Values{
		"client_id":     {id},
		"redirect_uri":  {absURL(c.r, "/login/"+p.name+"_callback", "")},
		"scope":         {p.scope},
		"response_type": {"code"},
		"state":         {state},
	}
	for k, v := range p.authExtra {
		q[k] = v
	}
	redirect(c.w, c.r, p.authURL+"?"+q.Encode())
}

func (s *Server) oauthCallback(c *uiContext, p oauthProvider) {
	enabled, id, secret := s.oauthCredentials(p)
	if !enabled {
		s.errorPage(c, http.StatusForbidden, p.name+" login is disabled")
		return
	}
	failedKey := p.name + "_failed_mail"
	state := c.sess.Get("oauth_state")
	c.sess.Delete("oauth_state")
	mailAddr := ""
	if state != "" && c.form.str("state") == state {
		var err error
		mailAddr, err = s.oauthUserMail(c.r.Context(), p, id, secret, c.form.str("code"), absURL(c.r, "/login/"+p.name+"_callback", ""))
		if err != nil {
			s.Log.Error("oauth", "provider", p.name, "error", err)
		}
	}
	ok, err := s.checkAccount(c.r.Context(), mailAddr)
	if err != nil {
		s.serverError(c, err)
		return
	}
	if !ok {
		c.sess.Set(failedKey, mailAddr)
		redirect(c.w, c.r, "/login/"+p.name+"_error")
		return
	}
	c.sess.Delete(failedKey)
	s.login(c, mailAddr)
}

func (s *Server) loginOAuthError(provider string) func(c *uiContext) {
	label := map[string]string{"google": "Google", "azuread": "AzureAD"}[provider]
	return func(c *uiContext) {
		s.render(c, http.StatusOK, "login/oauth_error", map[string]string{
			"Provider": label, "Mail": c.sess.Get(provider + "_failed_mail")})
	}
}

// oauthUserMail は認可コードをアクセストークンに換え, ユーザのメールアドレスを取得する.
func (s *Server) oauthUserMail(ctx context.Context, p oauthProvider, id, secret, code, callback string) (string, error) {
	if code == "" {
		return "", fmt.Errorf("no code")
	}
	form := url.Values{
		"code":          {code},
		"client_id":     {id},
		"client_secret": {secret},
		"redirect_uri":  {callback},
		"grant_type":    {"authorization_code"},
	}
	if p.name == "azuread" {
		form.Set("scope", p.scope)
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, p.tokenURL, strings.NewReader(form.Encode()))
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
	req, _ = http.NewRequestWithContext(ctx, http.MethodGet, p.userURL, nil)
	req.Header.Set("Authorization", token.TokenType+" "+token.AccessToken)
	var user map[string]any
	if err := doJSON(req, &user); err != nil {
		return "", fmt.Errorf("userinfo: %w", err)
	}
	m, _ := user[p.mailField].(string)
	return m, nil
}

var oauthClient = &http.Client{Timeout: 30 * time.Second}

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

// checkAccount はメールアドレスが allowed_mailaddr_pattern に一致するか, user_pass に登録されていればログインを認める.
func (s *Server) checkAccount(ctx context.Context, mailAddr string) (bool, error) {
	if mailAddr == "" {
		return false, nil
	}
	if p := s.Config.Login.AllowedMailaddrPattern; p != "" {
		re, err := CompilePHPRegexp(p)
		if err != nil {
			return false, fmt.Errorf("login.allowed_mailaddr_pattern: %w", err)
		}
		if re.MatchString(mailAddr) {
			return true, nil
		}
	}
	up, err := GetUserPass(ctx, s.Core.DB, mailAddr)
	return up != nil, err
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// serverError は 500 のエラーページ.
func (s *Server) serverError(c *uiContext, err error) {
	s.Log.Error("error", "uri", c.r.URL.RequestURI(), "error", err)
	s.errorPage(c, http.StatusInternalServerError, "Internal Server Error")
}
