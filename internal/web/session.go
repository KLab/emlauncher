package web

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// session は署名付き Cookie に入れるセッション. サーバ側に状態を持たないので,
// 複数台で動かしても (session.secret が同じなら) どのサーバでも読める.
type session struct {
	Mail string `json:"mail,omitempty"` // ログイン中のユーザ
	// URLBeforeLogin はログイン前に開こうとした URL. ログイン後にここへ戻る.
	URLBeforeLogin string `json:"url,omitempty"`
	// 削除確認画面で発行した CSRF トークン (使い捨て)
	PackageDeleteToken string `json:"pkg_del,omitempty"`
	AttachDeleteToken  string `json:"att_del,omitempty"`
	// OAuth の state と, ログインを拒否したアカウント (エラー画面に表示する)
	OAuthState string `json:"oauth_state,omitempty"`
	FailedMail string `json:"failed_mail,omitempty"`

	Issued int64 `json:"iat"`
}

const (
	sessionCookie = "emlauncher_session"
	// sessionLifetime を過ぎたセッションは無効 (Cookie を盗まれたときの影響を限る).
	sessionLifetime = 30 * 24 * time.Hour
	// clockSkew は複数台のサーバの時計のずれとして許す範囲 (発行時刻が未来でもこの範囲なら有効)
	clockSkew = 5 * time.Minute
)

type sessionCodec struct {
	key []byte
	now func() time.Time
}

func newSessionCodec(secret string) *sessionCodec {
	return &sessionCodec{key: []byte(secret), now: time.Now}
}

// load はリクエストの Cookie からセッションを読む. 無いか不正なら空のセッション.
func (c *sessionCodec) load(r *http.Request) *session {
	ck, err := r.Cookie(sessionCookie)
	if err != nil {
		return &session{}
	}
	s, ok := c.decode(ck.Value)
	if !ok {
		return &session{}
	}
	return s
}

// save はセッションを Cookie に書く. ログインし直すまで Issued は変えない.
func (c *sessionCodec) save(w http.ResponseWriter, r *http.Request, s *session) {
	if s.Issued == 0 {
		s.Issued = c.now().Unix()
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    c.encode(s),
		Path:     "/",
		HttpOnly: true,
		Secure:   requestScheme(r) == "https",
		SameSite: http.SameSiteLaxMode,
	})
}

func (c *sessionCodec) encode(s *session) string {
	payload, _ := json.Marshal(s)
	p := base64.RawURLEncoding.EncodeToString(payload)
	return p + "." + base64.RawURLEncoding.EncodeToString(c.sign(p))
}

func (c *sessionCodec) decode(v string) (*session, bool) {
	p, sig, ok := strings.Cut(v, ".")
	if !ok {
		return nil, false
	}
	got, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil || !hmac.Equal(got, c.sign(p)) {
		return nil, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(p)
	if err != nil {
		return nil, false
	}
	var s session
	if err := json.Unmarshal(payload, &s); err != nil {
		return nil, false
	}
	issued := time.Unix(s.Issued, 0)
	if now := c.now(); issued.After(now.Add(clockSkew)) || now.Sub(issued) > sessionLifetime {
		return nil, false
	}
	return &s, true
}

func (c *sessionCodec) sign(payload string) []byte {
	m := hmac.New(sha256.New, c.key)
	m.Write([]byte("emlauncher-session:" + payload))
	return m.Sum(nil)
}
