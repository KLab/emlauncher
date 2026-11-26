package web

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	// maxFormValues はファイル以外のフィールドの合計の上限 (アイコンの data URL を含む).
	// urlencoded の本文の上限も同じ.
	maxFormValues = 32 << 20
	// maxFormFiles は 1 リクエストで受け付けるファイルの数 (PHP の max_file_uploads の既定値).
	maxFormFiles = 20
)

// uploadedFile はリクエストで送られたファイル. 中身は一時ファイル (Path) に書いてある.
type uploadedFile struct {
	Name        string // 送られたファイル名
	Path        string
	Size        int64
	ContentType string
}

// form はリクエストのパラメータ. クエリ文字列と本文 (urlencoded / multipart) を合わせたもの
// (同じ名前なら本文を優先する. PHP の $_REQUEST と同じ). post は本文のパラメータだけ ($_POST).
type form struct {
	url.Values
	post    url.Values
	files   map[string]*uploadedFile
	tempDir string
}

// file は name で送られたファイル. 無ければ nil.
func (f *form) file(name string) *uploadedFile {
	return f.files[name]
}

// has は name のパラメータがあるか (空文字列でも true).
func (f *form) has(name string) bool {
	_, ok := f.Values[name]
	return ok
}

// cleanup は一時ファイルを消す.
func (f *form) cleanup() {
	if f.tempDir != "" {
		os.RemoveAll(f.tempDir)
	}
}

// parseForm はリクエストのパラメータを読む. multipart のファイルはメモリに載せず,
// 一時ディレクトリ (TMPDIR) のファイルに書く (数 GB のパッケージを受けるため).
// 呼び出し側は使い終わったら cleanup を呼ぶ.
func parseForm(w http.ResponseWriter, r *http.Request) (*form, error) {
	f := &form{Values: r.URL.Query(), post: url.Values{}, files: map[string]*uploadedFile{}}
	ct, params, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if r.Method != http.MethodPost && r.Method != http.MethodPut || ct != "multipart/form-data" {
		r.Body = http.MaxBytesReader(w, r.Body, maxFormValues)
		if err := r.ParseForm(); err != nil {
			return nil, badRequest("invalid form: " + err.Error())
		}
		f.post = r.PostForm
		for k, v := range r.PostForm {
			f.Values[k] = v
		}
		return f, nil
	}
	if params["boundary"] == "" {
		return nil, badRequest("multipart boundary is missing")
	}
	if err := f.readMultipart(multipart.NewReader(r.Body, params["boundary"])); err != nil {
		f.cleanup()
		return nil, err
	}
	return f, nil
}

func (f *form) readMultipart(mr *multipart.Reader) error {
	body := url.Values{}
	valueBytes := 0
	for {
		p, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return badRequest("invalid multipart: " + err.Error())
		}
		name := p.FormName()
		if name == "" {
			p.Close()
			continue
		}
		if !hasFilenameParam(p) {
			b, err := io.ReadAll(io.LimitReader(p, int64(maxFormValues-valueBytes+1)))
			if err != nil {
				return fmt.Errorf("read form value %s: %w", name, err)
			}
			valueBytes += len(b)
			if valueBytes > maxFormValues {
				return badRequest("form values too large")
			}
			body.Add(name, string(b))
			continue
		}
		if p.FileName() == "" {
			// ファイルが選ばれていない <input type="file"> (filename="")
			p.Close()
			continue
		}
		if len(f.files) >= maxFormFiles {
			return badRequest("too many files")
		}
		uf, err := f.saveFile(p)
		if err != nil {
			return err
		}
		f.files[name] = uf
	}
	f.post = body
	for k, v := range body {
		f.Values[k] = v
	}
	return nil
}

func hasFilenameParam(p *multipart.Part) bool {
	_, params, _ := mime.ParseMediaType(p.Header.Get("Content-Disposition"))
	_, ok := params["filename"]
	return ok
}

func (f *form) saveFile(p *multipart.Part) (*uploadedFile, error) {
	if f.tempDir == "" {
		dir, err := os.MkdirTemp("", "emlauncher-upload-")
		if err != nil {
			return nil, err
		}
		f.tempDir = dir
	}
	out, err := os.CreateTemp(f.tempDir, "file-")
	if err != nil {
		return nil, err
	}
	n, err := io.Copy(out, p)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return nil, fmt.Errorf("receive file %s: %w", p.FileName(), err)
	}
	ct := p.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/octet-stream"
	}
	return &uploadedFile{
		Name:        filepath.Base(strings.ReplaceAll(p.FileName(), `\`, "/")),
		Path:        out.Name(),
		Size:        n,
		ContentType: ct,
	}, nil
}
