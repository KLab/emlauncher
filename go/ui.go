package emlauncher

import (
	"net/http"
	"strings"
)

// セッションのキー (PHP 版と同じ名前).
const (
	sesLoginUser      = "login_user"
	sesURLBeforeLogin = "url_before_login"
)

// uiContext は画面のリクエスト 1 回分の情報.
type uiContext struct {
	w      http.ResponseWriter
	r      *http.Request
	sess   *Session
	form   *form
	user   string // ログイン中のメールアドレス (未ログインなら空)
	module string
	action string
}

type authMode int

const (
	authLogin  authMode = iota // ログイン必須. 未ログインならログイン画面へ
	authNone                   // 認証しない (QR コード, plist など)
	authGuest                  // ログイン画面 (ログイン済みならトップへ)
	authAjax                   // ログイン必須. 未ログインなら 403 の JSON
	authCustom                 // ハンドラで判断する (トークン付きインストールなど)
)

type uiRoute struct {
	auth    authMode
	handler func(c *uiContext)
}

// routes は /{module}/{action} のハンドラを登録する. PHP 版と同じく, action 省略時は index.
func (s *Server) routes(mux *http.ServeMux) {
	st := staticHandler()
	for _, p := range []string{"/css/", "/favicon.ico", "/apple-touch-icon.png"} {
		mux.Handle(p, st)
	}
	mux.HandleFunc("/api/", s.api)
	mux.HandleFunc("/api", s.api)
	mux.HandleFunc("/", s.ui)
}

func (s *Server) uiRoutes() map[string]uiRoute {
	return map[string]uiRoute{
		"top/index": {authLogin, s.topIndex},

		"login/index":             {authGuest, s.loginIndex},
		"login/password":          {authGuest, s.loginPassword},
		"login/password_reminder": {authGuest, s.loginPasswordReminder},
		"login/password_confirm":  {authGuest, s.loginPasswordConfirm},
		"login/password_reset":    {authGuest, s.loginPasswordReset},
		"login/password_commit":   {authGuest, s.loginPasswordCommit},
		"login/google":            {authGuest, s.loginGoogle},
		"login/google_callback":   {authGuest, s.loginGoogleCallback},
		"login/google_error":      {authGuest, s.loginOAuthError("google")},
		"login/azuread":           {authGuest, s.loginAzureAD},
		"login/azuread_callback":  {authGuest, s.loginAzureADCallback},
		"login/azuread_error":     {authGuest, s.loginOAuthError("azuread")},
		"logout/index":            {authLogin, s.logout},

		"app/index":                      {authLogin, s.appPage(s.appIndex)},
		"app/new":                        {authLogin, s.appNew},
		"app/create":                     {authLogin, s.appCreate},
		"app/comment":                    {authLogin, s.appPage(s.appComment)},
		"app/comment_post":               {authLogin, s.appPage(s.appCommentPost)},
		"app/preferences":                {authLogin, s.appOwnerPage(s.appPreferences)},
		"app/preferences_refresh_apikey": {authLogin, s.appOwnerPage(s.appRefreshAPIKey)},
		"app/preferences_update":         {authLogin, s.appOwnerPage(s.appUpdateInfo)},
		"app/preferences_delete_tags":    {authLogin, s.appOwnerPage(s.appDeleteTags)},
		"app/preferences_update_owners":  {authLogin, s.appOwnerPage(s.appUpdateOwners)},
		"app/upload":                     {authLogin, s.appOwnerPage(s.appUpload)},
		"app/upload_post":                {authLogin, s.appOwnerPage(s.appUploadPost)},

		"package/index":                 {authLogin, s.pkgPage(s.pkgIndex)},
		"package/install":               {authCustom, s.pkgInstall},
		"package/install_plist":         {authNone, s.pkgInstallPlist},
		"package/guestpass_install":     {authCustom, s.pkgGuestPassInstall},
		"package/create_token":          {authLogin, s.pkgPage(s.pkgCreateToken)},
		"package/create_guestpass":      {authLogin, s.pkgPage(s.pkgCreateGuestPass)},
		"package/guestpass":             {authLogin, s.pkgPage(s.pkgGuestPass)},
		"package/expire_guestpass":      {authLogin, s.pkgPage(s.pkgExpireGuestPass)},
		"package/edit":                  {authLogin, s.pkgOwnerPage(s.pkgEdit)},
		"package/edit_commit":           {authLogin, s.pkgOwnerPage(s.pkgEditCommit)},
		"package/delete_confirm":        {authLogin, s.pkgOwnerPage(s.pkgDeleteConfirm)},
		"package/delete":                {authLogin, s.pkgOwnerPage(s.pkgDelete)},
		"package/attach":                {authLogin, s.pkgOwnerPage(s.pkgAttach)},
		"package/attach_download":       {authLogin, s.pkgPage(s.attachedPage(s.pkgAttachDownload))},
		"package/attach_delete_confirm": {authLogin, s.pkgOwnerPage(s.attachedPage(s.pkgAttachDeleteConfirm))},
		"package/attach_delete":         {authLogin, s.pkgOwnerPage(s.attachedPage(s.pkgAttachDelete))},

		"myapps/installed": {authLogin, s.myappsInstalled},
		"myapps/own":       {authLogin, s.myappsOwn},
		"myapps/delete":    {authLogin, s.myappsDelete},
		"guestpass/list":   {authLogin, s.guestpassList},

		"doc/keystore": {authLogin, s.docKeystore},
		"qr/code":      {authNone, s.qrCode},

		"ajax/notification_setting":     {authAjax, s.ajaxNotificationSetting},
		"ajax/upload_package_temporary": {authAjax, s.ajaxUploadPackageTemporary},
	}
}

// ui は画面のリクエストを module/action に振り分ける.
func (s *Server) ui(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	module, action := "top", "index"
	if len(parts) > 0 && parts[0] != "" {
		module = parts[0]
	}
	if len(parts) > 1 && parts[1] != "" {
		action = parts[1]
	}
	route, ok := s.routeTable[module+"/"+action]
	if _, isDoc := s.templates["doc/"+action]; !ok && module == "doc" && isDoc {
		route, ok = uiRoute{authLogin, s.docPage}, true
	}
	if !ok {
		if module == "ajax" {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "404 Not Found"})
			return
		}
		http.Error(w, "404 Not Found", http.StatusNotFound)
		return
	}

	sw := &sessionWriter{ResponseWriter: w, r: r, store: s.sessions}
	sw.sess = s.sessions.Load(r)
	c := &uiContext{w: sw, r: r, sess: sw.sess, user: sw.sess.Get(sesLoginUser), module: module, action: action}

	f, err := parseForm(r)
	if err != nil {
		http.Error(sw, "400 Bad Request", http.StatusBadRequest)
		return
	}
	defer f.cleanup()
	c.form = f

	switch route.auth {
	case authLogin:
		if c.user == "" {
			s.redirectToLogin(c)
			return
		}
		logAttr(r, "user", c.user)
	case authGuest:
		if c.user != "" {
			redirect(c.w, r, "/")
			return
		}
	case authAjax:
		if c.user == "" {
			writeJSON(c.w, http.StatusForbidden, map[string]string{"error": "login required"})
			return
		}
		logAttr(r, "user", c.user)
	}
	route.handler(c)
}

// redirectToLogin はログイン後に戻ってくる URL を保存して, ログイン画面へリダイレクトする.
func (s *Server) redirectToLogin(c *uiContext) {
	if c.r.Method == http.MethodGet {
		u := requestURL(c.r)
		if s.Config.EnableHTTPS {
			u = strings.Replace(u, "http:", "https:", 1)
		}
		c.sess.Set(sesURLBeforeLogin, u)
	} else {
		c.sess.Delete(sesURLBeforeLogin)
	}
	scheme := ""
	if s.Config.EnableHTTPS {
		scheme = "https"
	}
	redirect(c.w, c.r, absURL(c.r, "/login", scheme))
}

// sessionWriter はレスポンスヘッダを書く直前にセッションの Cookie を書く.
type sessionWriter struct {
	http.ResponseWriter
	r     *http.Request
	store interface {
		Save(http.ResponseWriter, *http.Request, *Session)
	}
	sess  *Session
	wrote bool
}

func (w *sessionWriter) WriteHeader(code int) {
	if !w.wrote {
		w.wrote = true
		w.store.Save(w.ResponseWriter, w.r, w.sess)
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *sessionWriter) Write(b []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

// badRequest は PHP 版の response(400) と同じく, 本文の無い 400 を返す.
func badRequest(c *uiContext) {
	c.w.WriteHeader(http.StatusBadRequest)
}
