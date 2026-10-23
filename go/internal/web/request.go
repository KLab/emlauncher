package web

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/KLabTools/emlauncher/go/internal/core"
)

// maxValueBytes はファイル以外のフォームの値の上限 (アイコンの data URL を含む).
const maxValueBytes = 32 << 20

// form は PHP の $_REQUEST + $_FILES に相当する. POST の値が GET より優先する.
// multipart のファイルはメモリに載せず一時ファイルに書く (最大 7GB のパッケージがあるため).
type form struct {
	values url.Values
	files  map[string]*core.Upload
}

func parseForm(r *http.Request) (*form, error) {
	f := &form{values: url.Values{}, files: map[string]*core.Upload{}}
	post := url.Values{}
	ct, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	switch {
	case r.Method == http.MethodPost && ct == "multipart/form-data":
		mr, err := r.MultipartReader()
		if err != nil {
			return nil, err
		}
		for {
			part, err := mr.NextPart()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				f.cleanup()
				return nil, err
			}
			name := part.FormName()
			if name == "" {
				part.Close()
				continue
			}
			if part.FileName() == "" {
				b, err := io.ReadAll(io.LimitReader(part, maxValueBytes+1))
				part.Close()
				if err != nil {
					f.cleanup()
					return nil, err
				}
				if len(b) > maxValueBytes {
					f.cleanup()
					return nil, fmt.Errorf("form value %q is too large", name)
				}
				post.Add(name, string(b))
				continue
			}
			up, err := saveUpload(part.FileName(), part.Header.Get("Content-Type"), part)
			part.Close()
			if err != nil {
				f.cleanup()
				return nil, err
			}
			f.files[name] = up
		}
	case r.Method == http.MethodPost:
		r.Body = http.MaxBytesReader(nil, r.Body, maxValueBytes)
		if err := r.ParseForm(); err != nil {
			return nil, err
		}
		post = r.PostForm
	}
	for k, v := range post {
		f.values[k] = v
	}
	for k, v := range r.URL.Query() {
		if _, ok := f.values[k]; !ok {
			f.values[k] = v
		}
	}
	return f, nil
}

func saveUpload(name, contentType string, r io.Reader) (*core.Upload, error) {
	tmp, err := os.CreateTemp("", "emlauncher-upload-")
	if err != nil {
		return nil, err
	}
	n, err := io.Copy(tmp, r)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp.Name())
		return nil, err
	}
	return &core.Upload{Name: name, Path: tmp.Name(), ContentType: contentType, Size: n}, nil
}

// cleanup はアップロードされた一時ファイルを消す.
func (f *form) cleanup() {
	for _, u := range f.files {
		os.Remove(u.Path)
	}
}

// get は値を返す. 無ければ ok=false.
func (f *form) get(key string) (string, bool) {
	v, ok := f.values[key]
	if !ok || len(v) == 0 {
		return "", false
	}
	return v[0], true
}

func (f *form) str(key string) string {
	v, _ := f.get(key)
	return v
}

// ptr は値があればそのポインタ, 無ければ nil.
func (f *form) ptr(key string) *string {
	if v, ok := f.get(key); ok {
		return &v
	}
	return nil
}

func (f *form) int(key string, def int64) int64 {
	v, ok := f.get(key)
	if !ok {
		return def
	}
	n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	if err != nil {
		return 0 // PHP の (int) キャストと同じく, 数値でなければ 0
	}
	return n
}

// has は値かファイルがあるか (PHP の mfwRequest::has).
func (f *form) has(key string) bool {
	if _, ok := f.values[key]; ok {
		return true
	}
	_, ok := f.files[key]
	return ok
}

// list は "key[]" の値.
func (f *form) list(key string) []string {
	return f.values[key+"[]"]
}

func (f *form) file(key string) *core.Upload {
	return f.files[key]
}

// truthy は PHP の (bool) キャストと同じ (空文字と "0" が false).
func truthy(v string) bool {
	return v != "" && v != "0"
}

// parseBool は API の真偽値 (PHP 版の apiActions::parseBool).
func parseBool(v string) bool {
	if !truthy(v) {
		return false
	}
	switch v {
	case "False", "false", "F", "f", "No", "no", "Null", "null":
		return false
	}
	return true
}

var nestedKeyRe = regexp.MustCompile(`^([^\[\]]+)\[(\d+)\]\[([^\[\]]+)\]$`)

// nested は "key[i][field]" 形式の値を i の順に返す.
func (f *form) nested(key string) []map[string]string {
	byIdx := map[int]map[string]string{}
	for k, v := range f.values {
		m := nestedKeyRe.FindStringSubmatch(k)
		if m == nil || m[1] != key || len(v) == 0 {
			continue
		}
		i, _ := strconv.Atoi(m[2])
		if byIdx[i] == nil {
			byIdx[i] = map[string]string{}
		}
		byIdx[i][m[3]] = v[0]
	}
	var idx []int
	for i := range byIdx {
		idx = append(idx, i)
	}
	sort.Ints(idx)
	var out []map[string]string
	for _, i := range idx {
		out = append(out, byIdx[i])
	}
	return out
}
