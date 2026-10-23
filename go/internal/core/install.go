package core

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"text/template"
	"time"

	"emlauncher/internal/mail"
	"emlauncher/internal/model"
	"emlauncher/internal/random"
)

// memcached のキー. PHP 版と共有する (移行前に発行したトークンを移行後も使えるように).
const (
	installTokenPrefix = "pkg_install_token_"
	plistTokenPrefix   = "ios_plist_token_"
	plistTokenTTL      = 60 * time.Second
)

// InstallToken はログインせずにインストールできるトークンの中身 (memcached に JSON で保存).
type InstallToken struct {
	Mail      string     `json:"mail"`
	PackageID flexString `json:"package_id"`
	Expire    string     `json:"expire"`
}

// flexString は JSON の文字列と数値のどちらも受け付け, 文字列で書き出す.
type flexString string

func (f *flexString) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*f = flexString(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*f = flexString(n.String())
	return nil
}

// CreateInstallToken はインストールトークンを発行する.
func (c *Core) CreateInstallToken(mailAddr string, pkgID int64, ttl time.Duration) (token string, expire time.Time, err error) {
	expire = c.Clock.Now().Add(ttl)
	b, err := json.Marshal(InstallToken{
		Mail:      mailAddr,
		PackageID: flexString(strconv.FormatInt(pkgID, 10)),
		Expire:    c.Clock.Format(expire),
	})
	if err != nil {
		return "", time.Time{}, err
	}
	token = random.String(32)
	if err := c.Cache.Set(installTokenPrefix+token, b, ttl); err != nil {
		return "", time.Time{}, err
	}
	return token, expire, nil
}

// LookupInstallToken は有効なトークンの中身を返す. 無効なら nil.
func (c *Core) LookupInstallToken(token string) (*InstallToken, int64, error) {
	b, ok, err := c.Cache.Get(installTokenPrefix + token)
	if err != nil || !ok {
		return nil, 0, err
	}
	var t InstallToken
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, 0, nil
	}
	exp, err := c.Clock.Parse(t.Expire)
	if err != nil || exp.Before(c.Clock.Now()) {
		return nil, 0, nil
	}
	id, err := strconv.ParseInt(string(t.PackageID), 10, 64)
	if err != nil {
		return nil, 0, nil
	}
	return &t, id, nil
}

// CreatePlistToken は iOS のインストーラが plist を取りに来るときの一時トークン (60 秒) を発行する.
// itms-services からのリクエストはセッションを引き継げないため.
func (c *Core) CreatePlistToken(pkgID int64) (string, error) {
	token := random.String(32)
	err := c.Cache.Set(plistTokenPrefix+token, []byte(strconv.FormatInt(pkgID, 10)), plistTokenTTL)
	return token, err
}

func (c *Core) CheckPlistToken(token string, pkgID int64) bool {
	if token == "" {
		return false
	}
	b, ok, err := c.Cache.Get(plistTokenPrefix + token)
	return err == nil && ok && strings.TrimSpace(string(b)) == strconv.FormatInt(pkgID, 10)
}

// LogInstall はインストールを記録する.
func (c *Core) LogInstall(ctx context.Context, pkg *model.Package, mailAddr, userAgent string) error {
	return model.LogInstall(ctx, c.DB, pkg.AppID, pkg.ID, mailAddr, userAgent, c.Clock.Now())
}

var uploadedMailTmpl = template.Must(template.New("").Parse(`New package "{{.Package.Title}}" was uploaded to "{{.App.Title}}".
{{if .Package.Description.String}}
"{{.Package.Description.String}}"
{{end}}
Platform: {{.Package.Platform}}
Tags:{{range .Tags}} {{.Name}}{{end}}

Package URL: {{.URL}}
`))

// NotifyPackageUploaded はアプリをインストールしたことがあり, 通知を切っていないユーザに Bcc でメールを送る.
func (c *Core) NotifyPackageUploaded(ctx context.Context, app *model.Application, pkg *model.Package, packageURL string) error {
	users, err := model.InstallUsersByApp(ctx, c.DB, app.ID)
	if err != nil {
		return err
	}
	var bcc []string
	for _, u := range users {
		if u.Notify {
			bcc = append(bcc, u.Mail)
		}
	}
	if len(bcc) == 0 {
		return nil
	}
	tags, err := model.TagsByPackage(ctx, c.DB, pkg.ID)
	if err != nil {
		return err
	}
	var body bytes.Buffer
	if err := uploadedMailTmpl.Execute(&body, map[string]any{"Package": pkg, "App": app, "Tags": tags, "URL": packageURL}); err != nil {
		return err
	}
	to := c.Config.MailBccTo
	if to == "" {
		to = c.Config.MailSender
	}
	return c.Mail.Send(mail.Message{
		From:    c.Config.MailSender,
		To:      []string{to},
		Bcc:     bcc,
		Subject: "New Package Uploaded to " + app.Title,
		Body:    body.String(),
	})
}

var commentMailTmpl = template.Must(template.New("").Parse(`New Comment to "{{.App.Title}}".

"{{.Comment.Message}}"
{{if .Package}}
Target Package: {{.Package.Title}}
{{end}}
Comment URL: {{.URL}}
`))

// NotifyNewComment はアプリのオーナーにコメントを通知する.
func (c *Core) NotifyNewComment(ctx context.Context, app *model.Application, cm *model.Comment, pageURL string) error {
	owners, err := model.Owners(ctx, c.DB, app.ID)
	if err != nil || len(owners) == 0 {
		return err
	}
	var pkg *model.Package
	if cm.PackageID.Valid {
		if pkg, err = model.GetPackage(ctx, c.DB, cm.PackageID.Int64); err != nil {
			return err
		}
	}
	var body bytes.Buffer
	if err := commentMailTmpl.Execute(&body, map[string]any{"App": app, "Comment": cm, "Package": pkg, "URL": pageURL}); err != nil {
		return err
	}
	return c.Mail.Send(mail.Message{
		From:    c.Config.MailSender,
		To:      owners,
		Subject: "New Comment to " + app.Title,
		Body:    body.String(),
	})
}
