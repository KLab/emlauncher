// Package config は EMLauncher (Go 版) の設定を読み込む.
//
// 1 環境 1 ファイルの YAML. キー名は PHP 版の設定 (config/emlauncher_config.php,
// config/mfw_serverenv_config.php) に揃えている. 例は config.sample.yml を参照.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	StorageS3        = "S3"
	StorageLocalFile = "LocalFile"
)

type Config struct {
	// Listen は HTTP サーバの待ち受けアドレス.
	Listen string `yaml:"listen"`

	// Timezone は DB の DATETIME を読み書きするときの TZ. 空なら実行環境の TZ (time.Local).
	// PHP 版の date.timezone と同じにすること.
	Timezone string `yaml:"timezone"`

	// CachePrefix は memcached のキー接頭辞. PHP 版の
	// "{application_identifier}_{MFW_ENV}_{PHP を置いたディレクトリ名}_" と同じにすること.
	CachePrefix string `yaml:"cache_prefix"`

	TitlePrefix string `yaml:"title_prefix"`
	EnableHTTPS bool   `yaml:"enable_https"`
	MailSender  string `yaml:"mail_sender"`
	MailBccTo   string `yaml:"mail_bcc_to"`

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

	location *time.Location
}

type Mail struct {
	// Sendmail はメール送信に使うコマンド. メッセージを標準入力に渡す (sendmail -t 形式).
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
}

type Package struct {
	FileSizeWarningIOS     int `yaml:"file_size_warning_ios"`
	FileSizeWarningAndroid int `yaml:"file_size_warning_android"`
}

type Session struct {
	// Secret は Cookie セッションの署名鍵. 複数台構成では全台で同じ値にする.
	Secret string `yaml:"secret"`
}

type Database struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Name     string `yaml:"name"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	// Authfile は "user:password" が書かれたファイル (PHP 版と同じ形式). 指定すると User / Password より優先する.
	Authfile string `yaml:"authfile"`
}

type Memcache struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type AWS struct {
	// Key / Secret が空なら IAM ロールなど既定の認証情報を使う.
	Key        string `yaml:"key"`
	Secret     string `yaml:"secret"`
	Region     string `yaml:"region"`
	BucketName string `yaml:"bucket_name"`
	// BaseURL は S3 互換ストレージ (LocalStack など) のエンドポイント. AWS の S3 では指定しない.
	BaseURL string `yaml:"base_url"`
	// ExternalURL はブラウザから見たエンドポイント (BaseURL と同じなら省略).
	ExternalURL string `yaml:"external_url"`
}

type LocalFile struct {
	Path      string `yaml:"path"`
	URLPrefix string `yaml:"url_prefix"`
	// Serve が true なら Go 版が URLPrefix 以下のファイルを配信する.
	// false なら従来どおり Apache の Alias などに任せる.
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

// Load は YAML ファイルを読み込む.
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(b)
}

func Parse(b []byte) (*Config, error) {
	c := &Config{
		Listen:       "127.0.0.1:8080",
		StorageClass: StorageS3,
		Mail:         Mail{Sendmail: "/usr/sbin/sendmail -t -i"},
		Database:     Database{Host: "localhost", Port: 3306, Name: "emlauncher"},
		Memcache:     Memcache{Host: "localhost", Port: 11211},
		APKFile:      APKFile{Java: "java"},
	}
	dec := yaml.NewDecoder(strings.NewReader(string(b)))
	dec.KnownFields(true)
	if err := dec.Decode(c); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	if err := c.init(); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	return c, nil
}

func (c *Config) init() error {
	c.location = time.Local
	if c.Timezone != "" {
		loc, err := time.LoadLocation(c.Timezone)
		if err != nil {
			return fmt.Errorf("timezone: %w", err)
		}
		c.location = loc
	}
	switch c.StorageClass {
	case StorageS3:
		if c.AWS.BucketName == "" || c.AWS.Region == "" {
			return fmt.Errorf("aws.bucket_name and aws.region are required for storage_class S3")
		}
	case StorageLocalFile:
		if c.LocalFile.Path == "" {
			return fmt.Errorf("local_file.path is required for storage_class LocalFile")
		}
	default:
		return fmt.Errorf("unknown storage_class %q", c.StorageClass)
	}
	if c.Database.Authfile != "" {
		b, err := os.ReadFile(c.Database.Authfile)
		if err != nil {
			return fmt.Errorf("database.authfile: %w", err)
		}
		user, pass, _ := strings.Cut(strings.TrimSpace(string(b)), ":")
		c.Database.User, c.Database.Password = user, pass
	}
	return nil
}

// Location は DB の日時に使う TZ.
func (c *Config) Location() *time.Location { return c.location }
