package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseSample(t *testing.T) {
	c, err := Load("../../config.sample.yml")
	if err != nil {
		t.Fatal(err)
	}
	if c.StorageClass != StorageS3 || c.AWS.BucketName == "" {
		t.Errorf("unexpected sample: %+v", c)
	}
	if c.Location().String() != "Asia/Tokyo" {
		t.Errorf("location = %v", c.Location())
	}
}

func TestDefaults(t *testing.T) {
	c, err := Parse([]byte("storage_class: LocalFile\nlocal_file:\n  path: /tmp/x\n"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Location() != time.Local {
		t.Errorf("empty timezone should be time.Local, got %v", c.Location())
	}
	if c.Listen != "127.0.0.1:8080" || c.Database.Port != 3306 || c.Memcache.Port != 11211 || c.APKFile.Java != "java" {
		t.Errorf("defaults not applied: %+v", c)
	}
}

func TestAuthfile(t *testing.T) {
	f := filepath.Join(t.TempDir(), "dbauth")
	os.WriteFile(f, []byte("emluser:p@ss:word\n"), 0o600)
	c, err := Parse([]byte("storage_class: LocalFile\nlocal_file: {path: /tmp}\ndatabase: {authfile: " + f + ", user: ignored}\n"))
	if err != nil {
		t.Fatal(err)
	}
	if c.Database.User != "emluser" || c.Database.Password != "p@ss:word" {
		t.Errorf("user=%q pass=%q", c.Database.User, c.Database.Password)
	}
}

func TestErrors(t *testing.T) {
	for _, y := range []string{
		"storage_class: S3\n",                          // bucket 無し
		"storage_class: LocalFile\n",                   // path 無し
		"storage_class: FTP\n",                         // 不明
		"storage_class: LocalFile\nlocal_file: {path: /tmp}\ntimezone: Mars/Base\n",
		"storage_class: LocalFile\nlocal_file: {path: /tmp}\nunknown_key: 1\n", // typo を検出する
	} {
		if _, err := Parse([]byte(y)); err == nil {
			t.Errorf("no error for %q", y)
		}
	}
}
