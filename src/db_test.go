package emlauncher

import (
	"context"
	"os"
	"testing"
	"time"
)

// testDatabase は E2E 環境の MySQL (e2e/env/up.sh) を使う. EML_TEST_DB_HOST が無ければスキップ.
func testDatabase(t *testing.T) DatabaseConfig {
	host := os.Getenv("EML_TEST_DB_HOST")
	if host == "" {
		t.Skip("EML_TEST_DB_HOST is not set")
	}
	return DatabaseConfig{Host: host, Port: 13306, Name: "emlauncher", User: "emlauncher", Password: "emlauncher"}
}

// DATETIME の読み書きが設定した TZ のローカル時刻になり, PHP 版が書いた文字列と一致すること.
func TestDatetimeLocation(t *testing.T) {
	tokyo, _ := time.LoadLocation("Asia/Tokyo")
	d, err := OpenDBDSN(DBDSN(testDatabase(t), tokyo))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	ctx := context.Background()
	conn, err := d.Connx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "CREATE TEMPORARY TABLE t (dt datetime not null, s varchar(32))"); err != nil {
		t.Fatal(err)
	}

	c := NewFixedClock(tokyo, time.Date(2026, 3, 4, 23, 30, 15, 700_000_000, time.UTC))
	now := c.Now() // 2026-03-05 08:30:15 JST
	if _, err := conn.ExecContext(ctx, "INSERT INTO t VALUES (?, 'go')", now); err != nil {
		t.Fatal(err)
	}
	// PHP 版は文字列で書く
	if _, err := conn.ExecContext(ctx, "INSERT INTO t VALUES ('2026-03-05 08:30:15', 'php')"); err != nil {
		t.Fatal(err)
	}

	var raw []string
	if err := conn.SelectContext(ctx, &raw, "SELECT CAST(dt AS CHAR) FROM t ORDER BY s"); err != nil {
		t.Fatal(err)
	}
	for _, r := range raw {
		if r != "2026-03-05 08:30:15" {
			t.Errorf("stored %q, want 2026-03-05 08:30:15 (秒未満の丸め・TZ ずれが無いこと)", r)
		}
	}
	var got []time.Time
	if err := conn.SelectContext(ctx, &got, "SELECT dt FROM t ORDER BY s"); err != nil {
		t.Fatal(err)
	}
	for _, g := range got {
		if !g.Equal(now) || c.Format(g) != "2026-03-05 08:30:15" {
			t.Errorf("read %v, want %v", g, now)
		}
	}
}
