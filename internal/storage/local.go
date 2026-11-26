package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/KLab/emlauncher-go/internal/randstr"
)

// Local はローカルディスクの保存先 (設定 storage_class: LocalFile).
//
// キー k のファイルは {dir}/{k}, URL は {urlPrefix}/{k}. URL に有効期限は無い.
type Local struct {
	dir       string
	urlPrefix string
}

func NewLocal(dir, urlPrefix string) *Local {
	return &Local{
		dir:       strings.TrimRight(dir, "/"),
		urlPrefix: strings.TrimRight(urlPrefix, "/"),
	}
}

// path はキーに対応するファイルのパス. ".." などでディレクトリの外を指すキーはエラー.
func (l *Local) path(key string) (string, error) {
	key = strings.Trim(key, "/")
	if key == "" || !fs.ValidPath(key) {
		return "", fmt.Errorf("storage: invalid key %q", key)
	}
	return filepath.Join(l.dir, filepath.FromSlash(key)), nil
}

func (l *Local) PutIcon(_ context.Context, key string, png []byte) error {
	p, err := l.prepare(key)
	if err != nil {
		return err
	}
	return writeFileAtomic(p, func(w io.Writer) error {
		_, err := w.Write(png)
		return err
	})
}

func (l *Local) PutFile(_ context.Context, key, src string) error {
	p, err := l.prepare(key)
	if err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	return writeFileAtomic(p, func(w io.Writer) error {
		_, err := io.Copy(w, in)
		return err
	})
}

func (l *Local) Move(_ context.Context, src, dst string) error {
	sp, err := l.path(src)
	if err != nil {
		return err
	}
	dp, err := l.prepare(dst)
	if err != nil {
		return err
	}
	if err := os.Rename(sp, dp); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrNotFound, src)
		}
		return err
	}
	return nil
}

func (l *Local) Delete(_ context.Context, key string) error {
	p, err := l.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func (l *Local) Size(_ context.Context, key string) (int64, error) {
	p, err := l.path(key)
	if err != nil {
		return 0, err
	}
	st, err := os.Stat(p)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, fmt.Errorf("%w: %s", ErrNotFound, key)
	}
	if err != nil {
		return 0, err
	}
	if !st.Mode().IsRegular() {
		return 0, fmt.Errorf("%w: %s", ErrNotFound, key)
	}
	return st.Size(), nil
}

func (l *Local) PublicURL(key string) string {
	return l.urlPrefix + "/" + strings.Trim(key, "/")
}

func (l *Local) DownloadURL(_ context.Context, key string, _ time.Duration, _ string) (string, error) {
	return l.PublicURL(key), nil
}

// prepare はキーのファイルを置くディレクトリを作る (0755. chmod はしない: 他のユーザが
// 作ったディレクトリに書くこともあるため).
func (l *Local) prepare(key string) (string, error) {
	p, err := l.path(key)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", err
	}
	return p, nil
}

// writeFileAtomic は同じディレクトリの一時ファイルに書いてから rename する.
// 書き込みの途中で失敗しても, 中途半端なファイルが正式なキーに残らない.
// パーミッションは作成時に 0644 (umask が効く) にする. chmod は使わない (他のユーザが所有する
// ディレクトリや, chmod できないファイルシステムでも動くように).
func writeFileAtomic(dst string, write func(io.Writer) error) (err error) {
	var f *os.File
	for range 10 {
		name := filepath.Join(filepath.Dir(dst), ".tmp-"+filepath.Base(dst)+"-"+randstr.New(8))
		f, err = os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if !errors.Is(err, fs.ErrExist) {
			break
		}
	}
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			f.Close()
			os.Remove(f.Name())
		}
	}()
	if err := write(f); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), dst)
}

// Handler は urlPrefix 以下のリクエストにファイルを返す http.Handler.
// ディレクトリの一覧は返さない. キーとして不正なパス (..) は 404.
func (l *Local) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, ok := strings.CutPrefix(r.URL.Path, l.urlPrefix+"/")
		if !ok || key != path.Clean(key) {
			http.NotFound(w, r)
			return
		}
		p, err := l.path(key)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		f, err := os.Open(p)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		st, err := f.Stat()
		if err != nil || !st.Mode().IsRegular() {
			http.NotFound(w, r)
			return
		}
		http.ServeContent(w, r, st.Name(), st.ModTime(), f)
	})
}

// URLPrefix は Handler で配信するパスの接頭辞.
func (l *Local) URLPrefix() string { return l.urlPrefix }
