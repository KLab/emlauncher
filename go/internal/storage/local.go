package storage

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/KLabTools/emlauncher/go/internal/config"
)

// Local はローカルディスクに保存する (PHP 版の LocalFile と同じ配置).
//
//	ファイル: rtrim(path, '/') + '/' + trim(key, '/')
//	URL:     rtrim(url_prefix, '/') + '/' + trim(key, '/')
type Local struct {
	dir       string
	urlPrefix string
}

func NewLocal(c config.LocalFile) *Local {
	return &Local{
		dir:       strings.TrimRight(c.Path, "/"),
		urlPrefix: strings.TrimRight(c.URLPrefix, "/"),
	}
}

// Path はキーに対応するファイルのパス. キーに ".." などを含めてディレクトリの外を指すことはできない.
func (l *Local) Path(key string) string {
	clean := path.Clean("/" + strings.Trim(key, "/"))
	return filepath.Join(l.dir, filepath.FromSlash(clean))
}

func (l *Local) SaveIcon(_ context.Context, key string, png []byte) error {
	p := l.Path(key)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, png, 0o644)
}

func (l *Local) SaveFile(_ context.Context, key, srcPath string) error {
	p := l.Path(key)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()
	tmp, err := os.CreateTemp(filepath.Dir(p), ".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, src); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), p)
}

func (l *Local) Rename(_ context.Context, srcKey, dstKey string) error {
	dst := l.Path(dstKey)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.Rename(l.Path(srcKey), dst)
}

func (l *Local) Delete(_ context.Context, key string) error {
	err := os.Remove(l.Path(key))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

func (l *Local) PublicURL(key string) string {
	return l.urlPrefix + "/" + escapeKey(strings.Trim(key, "/"))
}

func (l *Local) DownloadURL(_ context.Context, key string, _ time.Duration, _ string) (string, error) {
	return l.PublicURL(key), nil
}

// URLPrefix は配信する URL のパス (末尾の "/" 無し).
func (l *Local) URLPrefix() string { return l.urlPrefix }

// Handler は URLPrefix 以下のファイルを配信する (local_file.serve: true のとき).
func (l *Local) Handler() http.Handler {
	fsrv := http.FileServer(noDirFS{http.Dir(l.dir)})
	return http.StripPrefix(l.urlPrefix, fsrv)
}

// noDirFS はディレクトリの一覧を返さない.
type noDirFS struct{ fs http.FileSystem }

func (n noDirFS) Open(name string) (http.File, error) {
	f, err := n.fs.Open(name)
	if err != nil {
		return nil, err
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if st.IsDir() {
		f.Close()
		return nil, fs.ErrNotExist
	}
	return f, nil
}
