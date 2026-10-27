package storage

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3 struct {
	client      *s3.Client
	presign     *s3.PresignClient
	uploader    *manager.Uploader
	bucket      string
	region      string
	baseURL     string
	externalURL string
}

func NewS3(ctx context.Context, c S3Config) (*S3, error) {
	opts := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(c.Region)}
	if c.Key != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(c.Key, c.Secret, "")))
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("s3: %w", err)
	}
	baseURL := strings.TrimRight(c.BaseURL, "/")
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if baseURL != "" {
			o.BaseEndpoint = aws.String(baseURL)
			o.UsePathStyle = true
		}
	})
	externalURL := strings.TrimRight(c.ExternalURL, "/")
	if externalURL == "" {
		externalURL = baseURL
	}
	return &S3{
		client:      client,
		presign:     s3.NewPresignClient(client),
		uploader:    manager.NewUploader(client),
		bucket:      c.BucketName,
		region:      c.Region,
		baseURL:     baseURL,
		externalURL: externalURL,
	}, nil
}

func (s *S3) SaveIcon(ctx context.Context, key string, png []byte) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &s.bucket,
		Key:         &key,
		Body:        bytes.NewReader(png),
		ACL:         types.ObjectCannedACLPublicRead,
		ContentType: aws.String("image/png"),
	})
	return err
}

func (s *S3) SaveFile(ctx context.Context, key, srcPath string) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()
	// 大きなファイルは自動でマルチパートアップロードになる
	_, err = s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:      &s.bucket,
		Key:         &key,
		Body:        f,
		ACL:         types.ObjectCannedACLPrivate,
		ContentType: aws.String(FileContentType),
	})
	return err
}

func (s *S3) Rename(ctx context.Context, srcKey, dstKey string) error {
	_, err := s.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     &s.bucket,
		Key:        &dstKey,
		CopySource: aws.String(s.bucket + "/" + escapeKey(srcKey)),
		ACL:        types.ObjectCannedACLPrivate,
	})
	if err != nil {
		return err
	}
	return s.Delete(ctx, srcKey)
}

func (s *S3) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &s.bucket, Key: &key})
	return err
}

func (s *S3) PublicURL(key string) string {
	if s.externalURL != "" {
		return s.externalURL + "/" + s.bucket + "/" + escapeKey(key)
	}
	if strings.Contains(s.bucket, ".") {
		return fmt.Sprintf("https://s3.%s.amazonaws.com/%s/%s", s.region, s.bucket, escapeKey(key))
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, escapeKey(key))
}

func (s *S3) DownloadURL(ctx context.Context, key string, expires time.Duration, fileName string) (string, error) {
	in := &s3.GetObjectInput{Bucket: &s.bucket, Key: &key}
	if fileName != "" {
		in.ResponseContentDisposition = aws.String(ContentDisposition(fileName))
	}
	req, err := s.presign.PresignGetObject(ctx, in, s3.WithPresignExpires(expires))
	if err != nil {
		return "", err
	}
	u := req.URL
	// S3 互換ストレージで, サーバから見たエンドポイントとブラウザから見たエンドポイントが違う場合 (PHP 版と同じ)
	if s.baseURL != "" && s.externalURL != s.baseURL {
		u = strings.Replace(u, s.baseURL, s.externalURL, 1)
	}
	return u, nil
}
