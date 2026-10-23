package harness

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Storage はストレージ (LocalFile / S3) をキーで直接確認する.
type Storage interface {
	Kind() string
	Exists(key string) bool
	Read(key string) []byte
	// Keys は prefix で始まるキーをソートして返す.
	Keys(prefix string) []string
	// Meta は S3 の ContentType と public-read かどうかを返す. LocalFile では ok=false.
	Meta(key string) (contentType string, publicRead bool, ok bool)
}

func newStorage(t testing.TB, cfg Config) Storage {
	switch cfg.Storage {
	case "local":
		return &localStorage{t: t, dir: cfg.LocalDir}
	case "s3":
		client := s3.New(s3.Options{
			Region:       "ap-northeast-1",
			BaseEndpoint: aws.String(cfg.S3Endpoint),
			UsePathStyle: true,
			Credentials:  credentials.NewStaticCredentialsProvider("test", "test", ""),
		})
		return &s3Storage{t: t, client: client, bucket: cfg.S3Bucket}
	}
	t.Fatalf("unknown E2E_STORAGE %q", cfg.Storage)
	return nil
}

type localStorage struct {
	t   testing.TB
	dir string
}

func (s *localStorage) Kind() string { return "local" }

func (s *localStorage) path(key string) string {
	return filepath.Join(s.dir, filepath.FromSlash(strings.Trim(key, "/")))
}

func (s *localStorage) Exists(key string) bool {
	_, err := os.Stat(s.path(key))
	return err == nil
}

func (s *localStorage) Read(key string) []byte {
	s.t.Helper()
	b, err := os.ReadFile(s.path(key))
	if err != nil {
		s.t.Fatalf("read %s: %v", key, err)
	}
	return b
}

func (s *localStorage) Keys(prefix string) []string {
	s.t.Helper()
	var keys []string
	err := filepath.WalkDir(s.dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(s.dir, p)
		key := filepath.ToSlash(rel)
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
		return nil
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		s.t.Fatalf("walk %s: %v", s.dir, err)
	}
	sort.Strings(keys)
	return keys
}

func (s *localStorage) Meta(string) (string, bool, bool) { return "", false, false }

type s3Storage struct {
	t      testing.TB
	client *s3.Client
	bucket string
}

func (s *s3Storage) Kind() string { return "s3" }

func (s *s3Storage) Exists(key string) bool {
	s.t.Helper()
	_, err := s.client.HeadObject(context.Background(), &s3.HeadObjectInput{Bucket: &s.bucket, Key: &key})
	if err == nil {
		return true
	}
	var nf *types.NotFound
	if errors.As(err, &nf) {
		return false
	}
	s.t.Fatalf("head %s: %v", key, err)
	return false
}

func (s *s3Storage) Read(key string) []byte {
	s.t.Helper()
	out, err := s.client.GetObject(context.Background(), &s3.GetObjectInput{Bucket: &s.bucket, Key: &key})
	if err != nil {
		s.t.Fatalf("get %s: %v", key, err)
	}
	defer out.Body.Close()
	b, err := io.ReadAll(out.Body)
	if err != nil {
		s.t.Fatalf("get %s: %v", key, err)
	}
	return b
}

func (s *s3Storage) Keys(prefix string) []string {
	s.t.Helper()
	var keys []string
	p := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{Bucket: &s.bucket, Prefix: &prefix})
	for p.HasMorePages() {
		page, err := p.NextPage(context.Background())
		if err != nil {
			s.t.Fatalf("list %s: %v", prefix, err)
		}
		for _, o := range page.Contents {
			keys = append(keys, *o.Key)
		}
	}
	sort.Strings(keys)
	return keys
}

func (s *s3Storage) Meta(key string) (string, bool, bool) {
	s.t.Helper()
	head, err := s.client.HeadObject(context.Background(), &s3.HeadObjectInput{Bucket: &s.bucket, Key: &key})
	if err != nil {
		s.t.Fatalf("head %s: %v", key, err)
	}
	acl, err := s.client.GetObjectAcl(context.Background(), &s3.GetObjectAclInput{Bucket: &s.bucket, Key: &key})
	if err != nil {
		s.t.Fatalf("get acl %s: %v", key, err)
	}
	public := false
	for _, g := range acl.Grants {
		if g.Grantee != nil && g.Grantee.URI != nil &&
			*g.Grantee.URI == "http://acs.amazonaws.com/groups/global/AllUsers" && g.Permission == types.PermissionRead {
			public = true
		}
	}
	return aws.ToString(head.ContentType), public, true
}
