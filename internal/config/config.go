// Package config は EMLauncher の設定ファイル (YAML) を読む.
//
// キー名は PHP 版の config/emlauncher_config.php, config/mfw_serverenv_config.php に揃えている.
// tools/export_php_config.php が書き出す JSON (YAML として読める) もそのまま読める.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Storage の種類 (storage_class).
const (
	StorageS3        = "S3"
	StorageLocalFile = "LocalFile"
)

type Config struct {
	Listen string `yaml:"listen"`
	// BaseURL はブラウザから見たサイトの URL ("https://emlauncher.example.com"). 指定すると, メールや
	// リダイレクト, API の応答などの絶対 URL をこれで作る (Host / X-Forwarded-Host ヘッダを使わない).
	BaseURL     string `yaml:"base_url"`
	Timezone    string `yaml:"timezone"`
	CachePrefix string `yaml:"cache_prefix"`
	TitlePrefix string `yaml:"title_prefix"`
	EnableHTTPS bool   `yaml:"enable_https"`
	MailSender  string `yaml:"mail_sender"`
	MailBccTo   string `yaml:"mail_bcc_to"`
	// UploadLockWaitTimeout はアップロード時の DB のロック待ちのタイムアウト (秒). 0 なら DB の設定のまま.
	UploadLockWaitTimeout int `yaml:"upload_lock_wait_timeout"`
	// APIKeyParamEnabled が false なら, API の api_key をリクエストパラメータでは受け付けない
	// (Authorization: Bearer ヘッダだけ). クエリの api_key はアクセスログに残るため.
	APIKeyParamEnabled bool `yaml:"api_key_param_enabled"`

	Mail     Mail     `yaml:"mail"`
	Login    Login    `yaml:"login"`
	Package  Package  `yaml:"package"`
	Session  Session  `yaml:"session"`
	Database Database `yaml:"database"`
	Memcache Memcache `yaml:"memcache"`

	StorageClass string    `yaml:"storage_class"`
	AWS          AWS       `yaml:"aws"`
	LocalFile    LocalFile `yaml:"local_file"`
	APKFile      APKFile   `yaml:"apkfile"`

	// Location は timezone を読み込んだもの. DB の DATETIME はこの TZ のローカル時刻.
	Location *time.Location `yaml:"-"`
}

type Mail struct {
	// Sendmail はメッセージを標準入力に渡すコマンド (sendmail -t 形式). 空白で区切って実行する.
	Sendmail string `yaml:"sendmail"`
}

type Login struct {
	EnablePassword         bool   `yaml:"enable_password"`
	EnableGoogleAuth       bool   `yaml:"enable_google_auth"`
	GoogleAppID            string `yaml:"google_app_id"`
	GoogleAppSecret        string `yaml:"google_app_secret"`
	AllowedMailaddrPattern string `yaml:"allowed_mailaddr_pattern"`
	EnableAzureADAuth      bool   `yaml:"enable_azuread_auth"`
	AzureADAppID           string `yaml:"azuread_app_id"`
	AzureADAppSecret       string `yaml:"azuread_app_secret"`

	// AllowedMail は AllowedMailaddrPattern をコンパイルしたもの. 未設定なら nil.
	AllowedMail *regexp.Regexp `yaml:"-"`
}

type Package struct {
	FileSizeWarningIOS     int `yaml:"file_size_warning_ios"`
	FileSizeWarningAndroid int `yaml:"file_size_warning_android"`
}

type Session struct {
	Secret string `yaml:"secret"`
}

type Database struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	Name string `yaml:"name"`
	// AuthFile は "user:password" が書かれたファイル. 指定すると User / Password より優先する.
	AuthFile string `yaml:"authfile"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type Memcache struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// Addr は memcached の "host:port".
func (m Memcache) Addr() string { return fmt.Sprintf("%s:%d", m.Host, m.Port) }

type AWS struct {
	Key        string `yaml:"key"`
	Secret     string `yaml:"secret"`
	Region     string `yaml:"region"`
	BucketName string `yaml:"bucket_name"`
	// BaseURL は S3 互換ストレージのエンドポイント. 指定すると path-style になる.
	BaseURL string `yaml:"base_url"`
	// ExternalURL は署名付き URL の BaseURL の部分を置き換える (ブラウザから見たエンドポイント).
	ExternalURL string `yaml:"external_url"`
}

type LocalFile struct {
	Path      string `yaml:"path"`
	URLPrefix string `yaml:"url_prefix"`
	// Serve が true なら URLPrefix 以下を EMLauncher が配信する.
	Serve bool `yaml:"serve"`
}

type APKFile struct {
	Java       string `yaml:"java"`
	Bundletool string `yaml:"bundletool"`
	Keystore   string `yaml:"keystore"`
	KSPass     string `yaml:"kspass"`
	KeyAlias   string `yaml:"keyalias"`
	KeyPass    string `yaml:"keypass"`
	AAPT2      string `yaml:"aapt2"`
}

// Load は設定ファイルを読み, 既定値を補って検証する.
// 知らないキーがあればエラーにする (設定の誤りに気付けるように).
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	c, err := Parse(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return c, nil
}

// Parse は設定ファイルの内容を読む. Load と同じく既定値を補って検証する.
func Parse(b []byte) (*Config, error) {
	c := &Config{
		Listen:             "127.0.0.1:8080",
		APIKeyParamEnabled: true,
		StorageClass:       StorageS3,
		Mail:               Mail{Sendmail: "/usr/sbin/sendmail -t -i"},
		Database:           Database{Host: "localhost", Port: 3306, Name: "emlauncher"},
		Memcache:           Memcache{Host: "localhost", Port: 11211},
		LocalFile:          LocalFile{URLPrefix: "/files", Serve: true},
		APKFile:            APKFile{Java: "java"},
	}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	// 空のファイルは io.EOF になる. 既定値だけで検証する.
	if err := dec.Decode(c); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if err := c.prepare(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) prepare() error {
	var errs []error

	if c.BaseURL != "" {
		if err := c.prepareBaseURL(); err != nil {
			errs = append(errs, fmt.Errorf("base_url: %w", err))
		}
	}

	if c.Timezone == "" {
		c.Location = time.Local
	} else {
		loc, err := time.LoadLocation(c.Timezone)
		if err != nil {
			errs = append(errs, fmt.Errorf("timezone: %w", err))
		}
		c.Location = loc
	}

	if c.Login.AllowedMailaddrPattern != "" {
		re, err := CompilePreg(c.Login.AllowedMailaddrPattern)
		if err != nil {
			errs = append(errs, fmt.Errorf("login.allowed_mailaddr_pattern: %w", err))
		}
		c.Login.AllowedMail = re
	}

	if len(c.Session.Secret) < 16 {
		errs = append(errs, errors.New("session.secret: must be at least 16 characters"))
	}

	switch c.StorageClass {
	case StorageS3:
		if c.AWS.BucketName == "" {
			errs = append(errs, errors.New("aws.bucket_name: required for storage_class S3"))
		}
		// 無いと起動はできてもアップロードなどがすべて失敗するので, 起動時に止める
		if c.AWS.Region == "" {
			errs = append(errs, errors.New("aws.region: required for storage_class S3"))
		}
	case StorageLocalFile:
		if c.LocalFile.Path == "" {
			errs = append(errs, errors.New("local_file.path: required for storage_class LocalFile"))
		}
		if c.LocalFile.URLPrefix == "" {
			errs = append(errs, errors.New("local_file.url_prefix: required for storage_class LocalFile"))
		}
	default:
		errs = append(errs, fmt.Errorf("storage_class: unknown value %q (S3 or LocalFile)", c.StorageClass))
	}

	if strings.TrimSpace(c.Mail.Sendmail) == "" {
		errs = append(errs, errors.New("mail.sendmail: required"))
	}
	if c.UploadLockWaitTimeout < 0 {
		errs = append(errs, errors.New("upload_lock_wait_timeout: must not be negative"))
	}

	if c.Database.AuthFile != "" {
		user, pass, err := readAuthFile(c.Database.AuthFile)
		if err != nil {
			errs = append(errs, fmt.Errorf("database.authfile: %w", err))
		}
		c.Database.User, c.Database.Password = user, pass
	}
	if c.Database.User == "" {
		errs = append(errs, errors.New("database.user: required (or database.authfile)"))
	}

	return errors.Join(errs...)
}

// readAuthFile は PHP 版 (mfwServerEnv) と同じく authfile を読む. 内容全体の前後の空白を除いて
// ":" で分け, 1 つ目をユーザ, 2 つ目をパスワード (無ければ空) にする.
func readAuthFile(path string) (user, password string, err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	parts := strings.Split(strings.TrimSpace(string(b)), ":")
	user = parts[0]
	if len(parts) > 1 {
		password = parts[1]
	}
	return user, password, nil
}

// MailBccRecipient は Bcc で送るメールの To. 未設定なら送信元.
func (c *Config) MailBccRecipient() string {
	if c.MailBccTo != "" {
		return c.MailBccTo
	}
	return c.MailSender
}

// FileSizeWarningMB はプラットフォームごとのストアの制限サイズ (MB). 0 なら警告しない.
func (c *Config) FileSizeWarningMB(platform string) int {
	switch platform {
	case "iOS":
		return c.Package.FileSizeWarningIOS
	case "Android":
		return c.Package.FileSizeWarningAndroid
	}
	return 0
}

// prepareBaseURL は base_url を検証し, 末尾の "/" を除く. スキームとホストだけを書く.
func (c *Config) prepareBaseURL() error {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" || u.Host == "" {
		return errors.New(`must be "http(s)://host[:port]"`)
	}
	if strings.Trim(u.Path, "/") != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return errors.New("must not have a path, query, fragment or user info")
	}
	if c.EnableHTTPS && u.Scheme != "https" {
		return errors.New("must be https when enable_https is true")
	}
	c.BaseURL = u.Scheme + "://" + u.Host
	return nil
}
