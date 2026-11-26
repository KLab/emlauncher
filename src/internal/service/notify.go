package service

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"emlauncher/internal/db"
	"emlauncher/internal/mailer"
)

var uploadMail = template.Must(template.New("upload").Parse(
	`New package "{{.Package.Title}}" was uploaded to "{{.App.Title}}".

{{with .Package.Description.String}}"{{.}}"

{{end}}Platform: {{.Package.Platform}}
Tags:{{range .Tags}} {{.Name}}{{end}}

Package URL: {{.URL}}
`))

var commentMail = template.Must(template.New("comment").Parse(
	`New Comment to "{{.App.Title}}".

"{{.Comment.Message}}"

{{with .Package}}Target Package: {{.Title}}
{{end}}Comment URL: {{.URL}}
`))

// notifyUpload はアプリをインストールしたユーザ (通知を切っていない人) に, 新しいパッケージを知らせる.
// 受信者同士でアドレスが見えないよう Bcc で送る.
func (s *Service) notifyUpload(ctx context.Context, appID int64, d PackageDetail, baseURL string) error {
	app, err := s.q.GetApplication(ctx, appID)
	if err != nil {
		return err
	}
	users, err := s.q.ListAppInstallUsers(ctx, appID)
	if err != nil {
		return err
	}
	var bcc []string
	for _, u := range users {
		// スキーマのコメントと逆だが, PHP 版のコードどおり notify が真なら送る
		if u.Notify != 0 {
			bcc = append(bcc, u.Mail)
		}
	}
	if len(bcc) == 0 {
		return nil
	}
	var body bytes.Buffer
	err = uploadMail.Execute(&body, map[string]any{
		"App": app, "Package": d.Package, "Tags": d.Tags,
		"URL": fmt.Sprintf("%s/package?id=%d", baseURL, d.ID),
	})
	if err != nil {
		return err
	}
	return s.mailer.Send(ctx, mailer.Message{
		From:    s.cfg.MailSender,
		To:      []string{s.cfg.MailBccRecipient()},
		Bcc:     bcc,
		Subject: "New Package Uploaded to " + app.Title,
		Body:    body.String(),
	})
}

// notifyComment はアプリのオーナー全員に新しいコメントを知らせる.
func (s *Service) notifyComment(ctx context.Context, app db.Application, c db.Comment, baseURL string) error {
	owners, err := s.q.ListOwners(ctx, app.ID)
	if err != nil || len(owners) == 0 {
		return err
	}
	var pkg *db.Package
	if c.PackageID.Valid {
		if p, err := s.q.GetPackage(ctx, c.PackageID.Int64); err == nil {
			pkg = &p
		}
	}
	var body bytes.Buffer
	err = commentMail.Execute(&body, map[string]any{
		"App": app, "Comment": c, "Package": pkg,
		"URL": fmt.Sprintf("%s/app/comment?id=%d#comment-%d", baseURL, app.ID, c.Number),
	})
	if err != nil {
		return err
	}
	return s.mailer.Send(ctx, mailer.Message{
		From:    s.cfg.MailSender,
		To:      owners,
		Subject: "New Comment to " + app.Title,
		Body:    body.String(),
	})
}
