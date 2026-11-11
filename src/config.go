// 設定の読み込み.
//
// 1 環境 1 ファイルの YAML. キー名は PHP 版の設定 (config/emlauncher_config.php,
// config/mfw_serverenv_config.php) に揃えている. 例は config.sample.yml を参照.

package emlauncher

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"emlauncher/pkgfile"
	"emlauncher/storage"
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

	// UploadLockWaitTimeout はアップロード時の DB のロック待ちのタイムアウト (秒).
	// 大きなファイルを並列にアップロードしてロック待ちがタイムアウトする場合に設定する.
	// 0 なら DB の設定 (innodb_lock_wait_timeout) のまま.
	UploadLockWaitTimeout int `yaml:"upload_lock_wait_timeout"`

	Mail     MailConfig     `yaml:"mail"`
	Login    LoginConfig    `yaml:"login"`
	Package  PackageConfig  `yaml:"package"`
	Session  SessionConfig  `yaml:"session"`
	Database DatabaseConfig `yaml:"database"`
	Memcache MemcacheConfig `yaml:"memcache"`

	StorageClass string                   `yaml:"storage_class"`
	AWS          storage.S3Config         `yaml:"aws"`
	LocalFile    storage.LocalConfig      `yaml:"local_file"`
	APKFile      pkgfile.BundletoolConfig `yaml:"apkfile"`

	location *time.Location
}

type MailConfig struct {
	// Sendmail はメール送信に使うコマンド. メッセージを標準入力に渡す (sendmail -t 形式).
	Sendmail string `yaml:"sendmail"`
}

type LoginConfig struct {
	EnablePassword         bool   `yaml:"enable_password"`
	EnableGoogleAuth       bool   `yaml:"enable_google_auth"`
	GoogleAppID            string `yaml:"google_app_id"`
	GoogleAppSecret        string `yaml:"google_app_secret"`
	AllowedMailaddrPattern string `yaml:"allowed_mailaddr_pattern"`
	EnableAzureADAuth      bool   `yaml:"enable_azuread_auth"`
	AzureADAppID           string `yaml:"azuread_app_id"`
	AzureADAppSecret       string `yaml:"azuread_app_secret"`
}

type PackageConfig struct {
	FileSizeWarningIOS     int `yaml:"file_size_warning_ios"`
	FileSizeWarningAndroid int `yaml:"file_size_warning_android"`
}

type SessionConfig struct {
	// Secret は Cookie セッションの署名鍵. 複数台構成では全台で同じ値にする.
	Secret string `yaml:"secret"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Name     string `yaml:"name"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	// Authfile は "user:password" が書かれたファイル (PHP 版と同じ形式). 指定すると User / Password より優先する.
	Authfile string `yaml:"authfile"`
}

type MemcacheConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// LoadConfig は YAML ファイルを読み込む.
func LoadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseConfig(b)
}

func ParseConfig(b []byte) (*Config, error) {
	c := &Config{
		Listen:       "127.0.0.1:8080",
		StorageClass: storage.ClassS3,
		Mail:         MailConfig{Sendmail: "/usr/sbin/sendmail -t -i"},
		Database:     DatabaseConfig{Host: "localhost", Port: 3306, Name: "emlauncher"},
		Memcache:     MemcacheConfig{Host: "localhost", Port: 11211},
		APKFile:      pkgfile.BundletoolConfig{Java: "java"},
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
	case storage.ClassS3:
		if c.AWS.BucketName == "" || c.AWS.Region == "" {
			return fmt.Errorf("aws.bucket_name and aws.region are required for storage_class S3")
		}
	case storage.ClassLocalFile:
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
