package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

const (
	contentTypeIcon = "image/png"
	// パッケージなどは PHP 版の実際の保存形式に合わせてすべてこの ContentType にする.
	contentTypeFile = "application/octet-stream"

	// CopyObject で 1 回にコピーできる上限. これより大きいものはマルチパートでコピーする.
	maxCopyObjectSize = 5 << 30
	copyPartSize      = 512 << 20
	copyConcurrency   = 8
	uploadPartSize    = 64 << 20
	uploadConcurrency = 4
)

// S3Config は S3 の保存先の設定 (設定 aws).
type S3Config struct {
	Key, Secret string // 空なら既定の認証情報 (IAM ロールなど)
	Region      string
	Bucket      string
	// BaseURL は S3 互換ストレージのエンドポイント. 指定すると path-style になる.
	BaseURL string
	// ExternalURL は返す URL の BaseURL の部分を置き換えるもの (ブラウザから見たエンドポイント).
	ExternalURL string
	// Logger は失敗しても処理を続けるもの (移動元の削除) を記録する. nil なら記録しない.
	Logger *slog.Logger
}

type S3 struct {
	log      *slog.Logger
	client   *s3.Client
	presign  *s3.PresignClient
	uploader *manager.Uploader
	bucket   string
	region   string
	baseURL  string
	external string

	// マルチパートでコピーする大きさとパートの大きさ (テストで小さくする)
	maxCopySize  int64
	copyPartSize int64
}

func NewS3(ctx context.Context, c S3Config) (*S3, error) {
	opts := []func(*awsconfig.LoadOptions) error{}
	if c.Region != "" {
		opts = append(opts, awsconfig.WithRegion(c.Region))
	}
	if c.Key != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(c.Key, c.Secret, "")))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("storage: aws config: %w", err)
	}
	baseURL := strings.TrimRight(c.BaseURL, "/")
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if baseURL != "" {
			o.BaseEndpoint = aws.String(baseURL)
			o.UsePathStyle = true
		}
	})
	return &S3{
		log:     c.Logger,
		client:  client,
		presign: s3.NewPresignClient(client),
		uploader: manager.NewUploader(client, func(u *manager.Uploader) {
			u.PartSize = uploadPartSize
			u.Concurrency = uploadConcurrency
		}),
		bucket:   c.Bucket,
		region:   awsCfg.Region,
		baseURL:  baseURL,
		external: strings.TrimRight(c.ExternalURL, "/"),

		maxCopySize:  maxCopyObjectSize,
		copyPartSize: copyPartSize,
	}, nil
}

func (s *S3) PutIcon(ctx context.Context, key string, png []byte) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &s.bucket,
		Key:         &key,
		Body:        bytes.NewReader(png),
		ACL:         types.ObjectCannedACLPublicRead,
		ContentType: aws.String(contentTypeIcon),
	})
	if err != nil {
		return fmt.Errorf("storage: put %s: %w", key, err)
	}
	return nil
}

func (s *S3) PutFile(ctx context.Context, key, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:      &s.bucket,
		Key:         &key,
		Body:        f,
		ACL:         types.ObjectCannedACLPrivate,
		ContentType: aws.String(contentTypeFile),
	})
	if err != nil {
		return fmt.Errorf("storage: upload %s: %w", key, err)
	}
	return nil
}

func (s *S3) Move(ctx context.Context, src, dst string) error {
	size, err := s.Size(ctx, src)
	if err != nil {
		return err
	}
	if size <= s.maxCopySize {
		_, err = s.client.CopyObject(ctx, &s3.CopyObjectInput{
			Bucket:     &s.bucket,
			Key:        &dst,
			CopySource: aws.String(s.copySource(src)),
			ACL:        types.ObjectCannedACLPrivate,
		})
	} else {
		err = s.multipartCopy(ctx, src, dst, size)
	}
	if err != nil {
		return fmt.Errorf("storage: copy %s to %s: %w", src, dst, err)
	}
	// コピーできたら移動は成功とする. 移動元を消せなくても失敗にしない: 失敗にすると呼び出し側は
	// DB をロールバックするが, コピー先は既にあるので行の無いファイルが残る. 消せなかった移動元
	// (一時ファイル) は使われないだけで害は無い.
	if err := s.Delete(ctx, src); err != nil && s.log != nil {
		s.log.Warn("storage: moved, but could not delete the source", "src", src, "dst", dst, "err", err)
	}
	return nil
}

func (s *S3) copySource(key string) string {
	return url.PathEscape(s.bucket) + "/" + escapeKey(key)
}

// multipartCopy は 5GB を超えるオブジェクトを UploadPartCopy でコピーする.
func (s *S3) multipartCopy(ctx context.Context, src, dst string, size int64) error {
	created, err := s.client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket:      &s.bucket,
		Key:         &dst,
		ACL:         types.ObjectCannedACLPrivate,
		ContentType: aws.String(contentTypeFile),
	})
	if err != nil {
		return err
	}
	abort := func() {
		// 失敗したアップロードの断片を残さない. ctx が切れていても実行する
		actx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Minute)
		defer cancel()
		s.client.AbortMultipartUpload(actx, &s3.AbortMultipartUploadInput{Bucket: &s.bucket, Key: &dst, UploadId: created.UploadId})
	}

	nparts := int((size + s.copyPartSize - 1) / s.copyPartSize)
	parts := make([]types.CompletedPart, nparts)
	errs := make([]error, nparts)
	sem := make(chan struct{}, copyConcurrency)
	var wg sync.WaitGroup
	for i := range nparts {
		first := int64(i) * s.copyPartSize
		last := min(first+s.copyPartSize, size) - 1
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			out, err := s.client.UploadPartCopy(ctx, &s3.UploadPartCopyInput{
				Bucket:          &s.bucket,
				Key:             &dst,
				UploadId:        created.UploadId,
				PartNumber:      aws.Int32(int32(i + 1)),
				CopySource:      aws.String(s.copySource(src)),
				CopySourceRange: aws.String(fmt.Sprintf("bytes=%d-%d", first, last)),
			})
			if err != nil {
				errs[i] = err
				return
			}
			parts[i] = types.CompletedPart{ETag: out.CopyPartResult.ETag, PartNumber: aws.Int32(int32(i + 1))}
		}()
	}
	wg.Wait()
	if err := errors.Join(errs...); err != nil {
		abort()
		return err
	}
	_, err = s.client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:          &s.bucket,
		Key:             &dst,
		UploadId:        created.UploadId,
		MultipartUpload: &types.CompletedMultipartUpload{Parts: parts},
	})
	if err != nil {
		abort()
		return err
	}
	return nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &s.bucket, Key: &key})
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("storage: delete %s: %w", key, err)
	}
	return nil
}

func (s *S3) Size(ctx context.Context, key string) (int64, error) {
	out, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: &s.bucket, Key: &key})
	if isNotFound(err) {
		return 0, fmt.Errorf("%w: %s", ErrNotFound, key)
	}
	if err != nil {
		return 0, fmt.Errorf("storage: head %s: %w", key, err)
	}
	return aws.ToInt64(out.ContentLength), nil
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	var nf *types.NotFound
	var nk *types.NoSuchKey
	if errors.As(err, &nf) || errors.As(err, &nk) {
		return true
	}
	var ae smithy.APIError
	return errors.As(err, &ae) && (ae.ErrorCode() == "NotFound" || ae.ErrorCode() == "NoSuchKey")
}

// PublicURL は署名の無いオブジェクトの URL.
func (s *S3) PublicURL(key string) string {
	if s.baseURL != "" {
		return s.externalize(s.baseURL + "/" + url.PathEscape(s.bucket) + "/" + escapeKey(key))
	}
	if strings.Contains(s.bucket, ".") {
		// ドットを含むバケット名は https の仮想ホスト形式にできないので path-style (aws-sdk-php と同じ)
		return "https://s3." + s.region + ".amazonaws.com/" + url.PathEscape(s.bucket) + "/" + escapeKey(key)
	}
	return "https://" + s.bucket + ".s3." + s.region + ".amazonaws.com/" + escapeKey(key)
}

func (s *S3) DownloadURL(ctx context.Context, key string, expires time.Duration, fileName string) (string, error) {
	in := &s3.GetObjectInput{Bucket: &s.bucket, Key: &key}
	if fileName != "" {
		in.ResponseContentDisposition = aws.String(contentDisposition(fileName))
	}
	req, err := s.presign.PresignGetObject(ctx, in, s3.WithPresignExpires(expires))
	if err != nil {
		return "", fmt.Errorf("storage: presign %s: %w", key, err)
	}
	return s.externalize(req.URL), nil
}

// externalize は URL の baseURL の部分を externalURL に置き換える.
func (s *S3) externalize(u string) string {
	if s.baseURL == "" || s.external == "" {
		return u
	}
	if rest, ok := strings.CutPrefix(u, s.baseURL); ok {
		return s.external + rest
	}
	return u
}

// escapeKey はキーを URL のパスとしてエスケープする (/ はそのまま).
func escapeKey(key string) string {
	parts := strings.Split(key, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}
