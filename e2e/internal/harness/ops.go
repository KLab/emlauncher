package harness

import (
	"encoding/base64"
	"fmt"
	"image/color"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"strings"

	"emlauncher-e2e/internal/fixture"
)

// User はログイン済みのクライアントを持つ利用者.
type User struct {
	Mail     string
	Password string
	*Client
}

// CreateUserRecord は user_pass にパスワード付きのユーザを登録する (ログインはしない).
func (e *Env) CreateUserRecord(prefix string) (mail, password string) {
	e.t.Helper()
	mail = strings.ToLower(Unique(prefix)) + "@example.com"
	password = "pw-" + Unique("p")
	e.DB.Exec("INSERT INTO user_pass (mail, passhash) VALUES (?, ?)",
		mail, PasswordHash(password, "E2Esalt012345678", 12))
	return mail, password
}

// NewUser はユーザを登録してログインする.
func (e *Env) NewUser(prefix string) *User {
	e.t.Helper()
	mail, password := e.CreateUserRecord(prefix)
	c := e.NewClient()
	c.Login(mail, password)
	return &User{Mail: mail, Password: password, Client: c}
}

type App struct {
	ID     int64
	Title  string
	APIKey string
}

// IconDataURL は画面からアイコンを送るときの data URL を作る.
func IconDataURL(png []byte) string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
}

// CreateApp は画面 (/app/create) からアプリを作る.
func (e *Env) CreateApp(owner *User, title string) App {
	e.t.Helper()
	res := owner.PostForm("/app/create", url.Values{
		"title":       {title},
		"icon-data":   {IconDataURL(fixture.PNG(200, 100, color.RGBA{0x33, 0x99, 0xcc, 0xff}))},
		"description": {"description of " + title},
		"repository":  {"https://example.com/" + title + ".git"},
	})
	owner.ExpectRedirect(res, "/app?id=")
	id := QueryInt(e.t, res.Location(), "id")
	row := e.DB.Row("SELECT api_key FROM application WHERE id = ?", id)
	return App{ID: id, Title: title, APIKey: row.Str("api_key")}
}

// APIUpload は /api/upload を呼ぶ. form には title などを入れる (api_key は自動で付ける).
func (e *Env) APIUpload(apiKey string, form url.Values, files ...File) *Response {
	e.t.Helper()
	f := url.Values{"api_key": {apiKey}}
	for k, v := range form {
		f[k] = v
	}
	return e.NewClient().PostMultipart("/api/upload", f, files...)
}

// MustAPIUpload は /api/upload でアップロードし, 作られた package の id を返す.
func (e *Env) MustAPIUpload(apiKey, title, fileName string, data []byte, form url.Values) int64 {
	e.t.Helper()
	if form == nil {
		form = url.Values{}
	}
	form.Set("title", title)
	res := e.APIUpload(apiKey, form, File{Field: "file", Name: fileName, Data: data})
	ExpectStatus(e.t, res, http.StatusOK)
	id, err := strconv.ParseInt(fmt.Sprint(res.JSONObject(e.t)["id"]), 10, 64)
	if err != nil {
		e.t.Fatalf("upload response has no id: %s", res.Body)
	}
	return id
}

// APICall は /api/{action} を api_key 付きで呼ぶ.
func (e *Env) APICall(action, apiKey string, form url.Values) *Response {
	e.t.Helper()
	f := url.Values{"api_key": {apiKey}}
	for k, v := range form {
		f[k] = v
	}
	return e.NewClient().PostForm("/api/"+action, f)
}

// UIUploadTemp は画面のアップロードの前半 (/ajax/upload_package_temporary) を行う.
func UIUploadTemp(u *User, fileName string, data []byte) *Response {
	return u.PostMultipart("/ajax/upload_package_temporary?name="+url.QueryEscape(fileName), nil,
		File{Field: "file", Name: fileName, Data: data})
}

// UIUploadPostForm は一時アップロードの結果から /app/upload_post のフォームを作る (画面の JS と同じ).
func UIUploadPostForm(temp map[string]any, fileSize int) url.Values {
	f := url.Values{
		"platform":   {fmt.Sprint(temp["platform"])},
		"temp_name":  {fmt.Sprint(temp["temp_name"])},
		"file_name":  {fmt.Sprint(temp["file_name"])},
		"identifier": {jsonString(temp["identifier"])},
		"file_size":  {strconv.Itoa(fileSize)},
	}
	afs, _ := temp["attached_files"].([]any)
	for i, a := range afs {
		m := a.(map[string]any)
		for _, k := range []string{"file_name", "temp_name", "size", "type"} {
			f.Set(fmt.Sprintf("attached_files[%d][%s]", i, k), fmt.Sprint(m[k]))
		}
	}
	return f
}

func jsonString(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

// PackageKey は package 行からストレージのキーを作る.
func PackageKey(pkg Row) string {
	return fmt.Sprintf("package/%s/%s_%s", pkg.Str("app_id"), pkg.Str("id"), pkg.Str("file_name"))
}

// AttachedKey は attached_file 行からストレージのキーを作る.
func AttachedKey(af Row) string {
	return fmt.Sprintf("package/%s/%s_%s_%s", af.Str("app_id"), af.Str("package_id"), af.Str("id"), af.Str("file_name"))
}

func (e *Env) Package(id int64) Row {
	e.t.Helper()
	return e.DB.Row("SELECT * FROM package WHERE id = ?", id)
}

func (e *Env) Application(id int64) Row {
	e.t.Helper()
	return e.DB.Row("SELECT * FROM application WHERE id = ?", id)
}

// Download はインストール等で返された URL (LocalFile の相対 URL / S3 の署名付き URL) から中身を取得する.
func (e *Env) Download(rawURL string) *Response {
	e.t.Helper()
	return e.NewClient().Get(rawURL)
}

// RunBatch は古いパッケージ削除バッチを実行し, 出力を返す.
func (e *Env) RunBatch(keep int) string {
	e.t.Helper()
	cmd := strings.ReplaceAll(e.Config.BatchCmd, "{keep}", strconv.Itoa(keep))
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		e.t.Fatalf("batch %q: %v\n%s", cmd, err, out)
	}
	return string(out)
}
