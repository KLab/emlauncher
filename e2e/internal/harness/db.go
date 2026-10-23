package harness

import (
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// DB はテスト対象の MySQL を直接読み書きする. 値はすべて文字列 (NULL は nil) で扱う.
type DB struct {
	t  testing.TB
	db *sql.DB
}

type Row map[string]*string

func (r Row) Str(col string) string {
	v, ok := r[col]
	if !ok {
		panic("no column " + col)
	}
	if v == nil {
		return ""
	}
	return *v
}

func (r Row) Null(col string) bool { return r[col] == nil }

func (r Row) Int(col string) int64 {
	n, err := strconv.ParseInt(r.Str(col), 10, 64)
	if err != nil {
		panic(fmt.Sprintf("column %s=%q is not int", col, r.Str(col)))
	}
	return n
}

func (d *DB) Exec(query string, args ...any) sql.Result {
	d.t.Helper()
	res, err := d.db.Exec(query, args...)
	if err != nil {
		d.t.Fatalf("exec %q: %v", query, err)
	}
	return res
}

func (d *DB) Rows(query string, args ...any) []Row {
	d.t.Helper()
	rows, err := d.db.Query(query, args...)
	if err != nil {
		d.t.Fatalf("query %q: %v", query, err)
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	var out []Row
	for rows.Next() {
		vals := make([]sql.NullString, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			d.t.Fatalf("scan %q: %v", query, err)
		}
		r := Row{}
		for i, c := range cols {
			if vals[i].Valid {
				s := vals[i].String
				r[c] = &s
			} else {
				r[c] = nil
			}
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		d.t.Fatalf("query %q: %v", query, err)
	}
	return out
}

// Row はちょうど 1 行を返す. 0 行や複数行ならテスト失敗.
func (d *DB) Row(query string, args ...any) Row {
	d.t.Helper()
	rows := d.Rows(query, args...)
	if len(rows) != 1 {
		d.t.Fatalf("query %q %v: %d rows, want 1", query, args, len(rows))
	}
	return rows[0]
}

func (d *DB) Count(query string, args ...any) int {
	d.t.Helper()
	var n int
	if err := d.db.QueryRow(query, args...).Scan(&n); err != nil {
		d.t.Fatalf("count %q: %v", query, err)
	}
	return n
}

// PasswordHash は PHP 版 (model/UserPass.php) と同じ方式で user_pass.passhash を作る.
func PasswordHash(password, salt string, stretch int) string {
	h := sha1hex(password + salt)
	for range stretch {
		h = sha1hex(h + salt)
	}
	return fmt.Sprintf("%d:%s:%s", stretch, salt, h)
}

// CheckPasswordHash は passhash が password に一致するかを PHP 版と同じ方式で確認する.
func CheckPasswordHash(passhash, password string) bool {
	parts := strings.SplitN(passhash, ":", 3)
	if len(parts) != 3 {
		return false
	}
	stretch, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}
	return PasswordHash(password, parts[1], stretch) == passhash
}

func sha1hex(s string) string {
	h := sha1.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}
