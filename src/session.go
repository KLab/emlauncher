// 署名付き Cookie のセッション.
// サーバ側に状態を持たないので, ロードバランサ配下で複数台になってもスティッキーセッションが要らない.
// 中身は改ざんできないが, 暗号化はしていない (秘密の値は入れない).

package emlauncher

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

const cookieName = "emlauncher_session"

// maxAge はセッションの有効期間 (署名に含める). Cookie 自体はブラウザを閉じるまで.
const maxAge = 30 * 24 * time.Hour

type SessionStore struct {
	key    []byte
	secure func(r *http.Request) bool
	now    func() time.Time
}

// NewStore は署名鍵からストアを作る. secure はリクエストが HTTPS かを返す (Cookie の Secure 属性).
func NewSessionStore(secret string, secure func(r *http.Request) bool) (*SessionStore, error) {
	if len(secret) < 16 {
		return nil, errors.New("session.secret must be at least 16 characters")
	}
	return &SessionStore{key: []byte(secret), secure: secure, now: time.Now}, nil
}

type Session struct {
	values  map[string]string
	changed bool
}

type payload struct {
	Values map[string]string `json:"v"`
	Issued int64             `json:"t"`
}

// Load はリクエストの Cookie からセッションを読む. 無い・不正・期限切れなら空のセッション.
func (s *SessionStore) Load(r *http.Request) *Session {
	sess := &Session{values: map[string]string{}}
	c, err := r.Cookie(cookieName)
	if err != nil {
		return sess
	}
	data, sig, ok := strings.Cut(c.Value, ".")
	if !ok || !hmac.Equal([]byte(sig), []byte(s.sign(data))) {
		return sess
	}
	b, err := base64.RawURLEncoding.DecodeString(data)
	if err != nil {
		return sess
	}
	var p payload
	if err := json.Unmarshal(b, &p); err != nil {
		return sess
	}
	if s.now().Sub(time.Unix(p.Issued, 0)) > maxAge {
		return sess
	}
	if p.Values != nil {
		sess.values = p.Values
	}
	return sess
}

// Save は変更があれば Cookie を書く. レスポンスヘッダを書く前に呼ぶこと.
func (s *SessionStore) Save(w http.ResponseWriter, r *http.Request, sess *Session) {
	if !sess.changed {
		return
	}
	c := &http.Cookie{
		Name:     cookieName,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secure(r),
		SameSite: http.SameSiteLaxMode,
	}
	if len(sess.values) == 0 {
		c.MaxAge = -1
	} else {
		var buf bytes.Buffer
		json.NewEncoder(&buf).Encode(payload{Values: sess.values, Issued: s.now().Unix()})
		data := base64.RawURLEncoding.EncodeToString(bytes.TrimSpace(buf.Bytes()))
		c.Value = data + "." + s.sign(data)
	}
	http.SetCookie(w, c)
	sess.changed = false
}

func (s *SessionStore) sign(data string) string {
	m := hmac.New(sha256.New, s.key)
	m.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func (s *Session) Get(key string) string { return s.values[key] }

func (s *Session) Set(key, value string) {
	if s.values[key] != value {
		s.values[key] = value
		s.changed = true
	}
}

func (s *Session) Delete(key string) {
	if _, ok := s.values[key]; ok {
		delete(s.values, key)
		s.changed = true
	}
}

func (s *Session) Clear() {
	if len(s.values) > 0 {
		s.values = map[string]string{}
		s.changed = true
	}
}
