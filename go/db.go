// MySQL への接続を作る.

package emlauncher

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// DSN は設定から go-sql-driver/mysql の DSN を作る.
// DATETIME は time.Time として読み書きし, TZ は設定の Location を使う (書き込み時もこの TZ に変換される).
func DBDSN(c DatabaseConfig, loc *time.Location) string {
	mc := mysql.NewConfig()
	mc.User = c.User
	mc.Passwd = c.Password
	mc.Net = "tcp"
	mc.Addr = net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
	mc.DBName = c.Name
	mc.ParseTime = true
	mc.Loc = loc
	mc.Collation = "utf8mb4_general_ci"
	return mc.FormatDSN()
}

func OpenDB(c *Config) (*sqlx.DB, error) {
	return OpenDBDSN(DBDSN(c.Database, c.Location()))
}

func OpenDBDSN(dsn string) (*sqlx.DB, error) {
	d, err := sqlx.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	d.SetConnMaxLifetime(5 * time.Minute)
	d.SetMaxIdleConns(10)
	if err := d.Ping(); err != nil {
		d.Close()
		return nil, fmt.Errorf("db: %w", err)
	}
	return d, nil
}
