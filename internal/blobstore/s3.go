package blobstore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	tmtypes "github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"

	"github.com/hostrepute/reputecast/internal/config"
)

// S3 keeps objects in an S3-compatible bucket (AWS S3, MinIO, Cloudflare
// R2, DigitalOcean Spaces, Backblaze B2, Wasabi…), under the configured
// prefix. Large objects and objects of unknown size go up in parts.
type S3 struct {
	client *s3.Client
	up     *transfermanager.Client
	bucket string
	prefix string
	sse    string
	kmsKey string
}

// NewS3Client builds the S3 client for a bucket's settings: the endpoint
// and path style of an S3-compatible service, the region, and static keys
// or else the default AWS credential chain (environment, shared files, an
// IAM role, IRSA).
func NewS3Client(ctx context.Context, c config.S3Config) (*s3.Client, error) {
	var opts []func(*awsconfig.LoadOptions) error
	if c.Region != "" {
		opts = append(opts, awsconfig.WithRegion(c.Region))
	}
	if c.AccessKeyID != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(c.AccessKeyID, c.SecretAccessKey, "")))
	}
	if c.Endpoint != "" {
		// Services other than AWS: checksums only where an operation needs
		// one, as several don't take the SDK's default CRC32 trailers.
		opts = append(opts, awsconfig.WithRequestChecksumCalculation(aws.RequestChecksumCalculationWhenRequired),
			awsconfig.WithResponseChecksumValidation(aws.ResponseChecksumValidationWhenRequired))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("blobstore: load the AWS configuration: %w", err)
	}
	if awsCfg.Region == "" {
		// S3-compatible services sign with any region; AWS's default.
		awsCfg.Region = "us-east-1"
	}
	return s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if c.Endpoint != "" {
			o.BaseEndpoint = aws.String(c.Endpoint)
		}
		o.UsePathStyle = c.PathStyle
	}), nil
}

// NewS3 opens the bucket storage.s3 names.
func NewS3(ctx context.Context, c config.S3Config) (*S3, error) {
	if strings.TrimSpace(c.Bucket) == "" {
		return nil, errors.New("blobstore: storage.s3.bucket isn't set")
	}
	client, err := NewS3Client(ctx, c)
	if err != nil {
		return nil, err
	}
	up := transfermanager.New(client, func(o *transfermanager.Options) {
		if c.Endpoint != "" {
			// The transfer manager's own setting wins over the client's.
			o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		}
	})
	return &S3{client: client, up: up, bucket: c.Bucket, prefix: normPrefix(c.Prefix), sse: c.SSE, kmsKey: c.KMSKeyID}, nil
}

// normPrefix is a prefix as keys are put under it: "" or ending in "/".
func normPrefix(p string) string {
	p = strings.Trim(p, "/")
	if p == "" {
		return ""
	}
	return p + "/"
}

// Bucket is the bucket's name.
func (s *S3) Bucket() string { return s.bucket }

// Prefix is what every key is put under ("" or ending in "/").
func (s *S3) Prefix() string { return s.prefix }

func (s *S3) key(key string) (*string, error) {
	if err := ValidKey(key); err != nil {
		return nil, err
	}
	return aws.String(s.prefix + key), nil
}

// Put uploads in one request up to 16 MB, in 8 MB parts beyond (and for a
// reader of unknown size past its first 16 MB); a failed multipart upload
// is aborted, so nothing is left behind.
func (s *S3) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	k, err := s.key(key)
	if err != nil {
		return err
	}
	in := &transfermanager.UploadObjectInput{Bucket: aws.String(s.bucket), Key: k, Body: exactly(r, size)}
	if size > 0 {
		// Sizes the parts of a large object (S3 takes 10,000 at most).
		in.ContentLength = aws.Int64(size)
	}
	if contentType != "" {
		in.ContentType = aws.String(contentType)
	}
	if s.sse != "" {
		in.ServerSideEncryption = tmtypes.ServerSideEncryption(s.sse)
		if s.kmsKey != "" {
			in.SSEKMSKeyID = aws.String(s.kmsKey)
		}
	}
	if _, err := s.up.UploadObject(ctx, in); err != nil {
		return fmt.Errorf("blobstore: put %s: %w", key, err)
	}
	return nil
}

func (s *S3) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	return s.OpenRange(ctx, key, 0, -1)
}

func (s *S3) OpenRange(ctx context.Context, key string, off, n int64) (io.ReadCloser, error) {
	k, err := s.key(key)
	if err != nil {
		return nil, err
	}
	if off < 0 {
		return nil, fmt.Errorf("blobstore: negative offset %d", off)
	}
	in := &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: k}
	switch {
	case n == 0:
		// HTTP has no empty range: whether the object exists, then nothing.
		if _, err := s.Stat(ctx, key); err != nil {
			return nil, err
		}
		return io.NopCloser(bytes.NewReader(nil)), nil
	case n > 0:
		in.Range = aws.String(fmt.Sprintf("bytes=%d-%d", off, off+n-1))
	case off > 0:
		in.Range = aws.String(fmt.Sprintf("bytes=%d-", off))
	}
	out, err := s.client.GetObject(ctx, in)
	if err != nil {
		if status(err) == http.StatusRequestedRangeNotSatisfiable {
			// From past the end (or an empty object): nothing to read.
			return io.NopCloser(bytes.NewReader(nil)), nil
		}
		return nil, s.fail("open", key, err)
	}
	return out.Body, nil
}

func (s *S3) Stat(ctx context.Context, key string) (Info, error) {
	k, err := s.key(key)
	if err != nil {
		return Info{}, err
	}
	out, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(s.bucket), Key: k})
	if err != nil {
		return Info{}, s.fail("stat", key, err)
	}
	return Info{Size: aws.ToInt64(out.ContentLength), ModTime: aws.ToTime(out.LastModified)}, nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	k, err := s.key(key)
	if err != nil {
		return err
	}
	if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(s.bucket), Key: k}); err != nil && status(err) != http.StatusNotFound {
		return s.fail("delete", key, err)
	}
	return nil
}

func (s *S3) List(ctx context.Context, prefix string, fn func(key string, info Info) error) error {
	pages := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{Bucket: aws.String(s.bucket), Prefix: aws.String(s.prefix + prefix)})
	for pages.HasMorePages() {
		page, err := pages.NextPage(ctx)
		if err != nil {
			return s.fail("list", prefix, err)
		}
		for _, o := range page.Contents {
			key := strings.TrimPrefix(aws.ToString(o.Key), s.prefix)
			if err := fn(key, Info{Size: aws.ToInt64(o.Size), ModTime: aws.ToTime(o.LastModified)}); err != nil {
				return err
			}
		}
	}
	return nil
}

// Check asks the bucket whether it exists and these credentials reach it.
func (s *S3) Check(ctx context.Context) error {
	if _, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s.bucket)}); err != nil {
		switch status(err) {
		case http.StatusNotFound:
			return fmt.Errorf("the bucket %s doesn't exist", s.bucket)
		case http.StatusForbidden:
			return fmt.Errorf("the credentials may not use the bucket %s", s.bucket)
		}
		return fmt.Errorf("the bucket %s doesn't answer: %w", s.bucket, err)
	}
	return nil
}

// fail names an operation's error; a key that holds nothing is
// ErrNotExist.
func (s *S3) fail(op, key string, err error) error {
	var api smithy.APIError
	if status(err) == http.StatusNotFound && (!errors.As(err, &api) || api.ErrorCode() != "NoSuchBucket") {
		return fmt.Errorf("blobstore: %s %s: %w", op, key, ErrNotExist)
	}
	return fmt.Errorf("blobstore: %s %s: %w", op, key, err)
}

// status is an S3 error's HTTP status (0 when it didn't get one).
func status(err error) int {
	var re interface{ HTTPStatusCode() int }
	if errors.As(err, &re) {
		return re.HTTPStatusCode()
	}
	return 0
}
