// Package objstore is a thin S3-compatible object-storage client.
//
// It wraps aws-sdk-go-v2/service/s3 with an opinionated, narrow API
// (Put/Get/Delete/Head/List/Presign) that works against AWS S3 plus any
// S3-compatible service: MinIO, Alibaba OSS, Tencent COS, Backblaze B2,
// Cloudflare R2, Wasabi, and so on. The path-style addressing option exists
// for self-hosted MinIO clusters that don't speak virtual-hosted style.
package objstore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// Options configures Open.
type Options struct {
	// Endpoint overrides the default AWS endpoint. Required for non-AWS
	// services (MinIO, OSS, COS, R2, ...). For AWS itself, leave empty.
	// Format: "https://oss-cn-hangzhou.aliyuncs.com" or "http://minio:9000".
	Endpoint string

	// Region is the bucket region. Required (S3 sigV4 needs it even for
	// self-hosted services; "us-east-1" is a safe placeholder for MinIO).
	Region string

	// AccessKey / SecretKey override the default credential chain.
	// Leave both empty to rely on env / IAM role / config files.
	AccessKey string
	SecretKey string
	SessionToken string

	// UsePathStyle forces "http://host/bucket/key" instead of
	// "http://bucket.host/key". Required by most self-hosted MinIO setups.
	UsePathStyle bool

	// DefaultBucket, when set, lets callers omit the bucket argument
	// (use Put / Get / etc; the package-level helpers append the bucket).
	DefaultBucket string
}

// Client is a high-level wrapper around an *s3.Client.
type Client struct {
	raw    *s3.Client
	bucket string
}

// Open builds a Client according to opts.
func Open(ctx context.Context, opts Options) (*Client, error) {
	if opts.Region == "" {
		return nil, errors.New("objstore: Region is required")
	}

	loadOpts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(opts.Region),
	}
	if opts.AccessKey != "" || opts.SecretKey != "" {
		loadOpts = append(loadOpts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(opts.AccessKey, opts.SecretKey, opts.SessionToken),
		))
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("objstore: load aws config: %w", err)
	}

	s3Opts := []func(*s3.Options){
		func(o *s3.Options) {
			if opts.Endpoint != "" {
				o.BaseEndpoint = aws.String(opts.Endpoint)
			}
			o.UsePathStyle = opts.UsePathStyle
		},
	}
	return &Client{
		raw:    s3.NewFromConfig(cfg, s3Opts...),
		bucket: opts.DefaultBucket,
	}, nil
}

// Raw exposes the underlying *s3.Client for advanced calls (multipart upload,
// bucket lifecycle, ACL, etc).
func (c *Client) Raw() *s3.Client { return c.raw }

// ObjectInfo describes a single object's metadata.
type ObjectInfo struct {
	Key          string
	Size         int64
	ETag         string
	LastModified time.Time
	ContentType  string
}

// PutOptions tweaks a Put call. ContentType defaults to application/octet-stream.
type PutOptions struct {
	ContentType string
	Metadata    map[string]string
	CacheControl string
}

// Put uploads body to bucket/key. The reader is consumed once; if you need to
// retry, supply a re-seekable reader.
func (c *Client) Put(ctx context.Context, bucket, key string, body io.Reader, opts PutOptions) error {
	bkt, err := c.resolveBucket(bucket)
	if err != nil {
		return err
	}
	if opts.ContentType == "" {
		opts.ContentType = "application/octet-stream"
	}
	_, err = c.raw.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(bkt),
		Key:          aws.String(key),
		Body:         body,
		ContentType:  aws.String(opts.ContentType),
		Metadata:     opts.Metadata,
		CacheControl: aws.String(opts.CacheControl),
	})
	if err != nil {
		return fmt.Errorf("objstore: put %s/%s: %w", bkt, key, err)
	}
	return nil
}

// PutBytes is a small convenience wrapper around Put for in-memory blobs.
func (c *Client) PutBytes(ctx context.Context, bucket, key string, data []byte, opts PutOptions) error {
	return c.Put(ctx, bucket, key, bytes.NewReader(data), opts)
}

// Get returns the object body as an io.ReadCloser. The caller MUST Close it.
// Returns ErrNotFound when the key does not exist.
func (c *Client) Get(ctx context.Context, bucket, key string) (io.ReadCloser, *ObjectInfo, error) {
	bkt, err := c.resolveBucket(bucket)
	if err != nil {
		return nil, nil, err
	}
	out, err := c.raw.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bkt),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFoundErr(err) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, fmt.Errorf("objstore: get %s/%s: %w", bkt, key, err)
	}
	info := &ObjectInfo{
		Key:         key,
		Size:        derefInt64(out.ContentLength),
		ETag:        derefStr(out.ETag),
		ContentType: derefStr(out.ContentType),
	}
	if out.LastModified != nil {
		info.LastModified = *out.LastModified
	}
	return out.Body, info, nil
}

// GetBytes fully buffers the object — handy for small files only.
func (c *Client) GetBytes(ctx context.Context, bucket, key string) ([]byte, *ObjectInfo, error) {
	body, info, err := c.Get(ctx, bucket, key)
	if err != nil {
		return nil, nil, err
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, info, fmt.Errorf("objstore: read body: %w", err)
	}
	return data, info, nil
}

// Head returns metadata without fetching the body.
func (c *Client) Head(ctx context.Context, bucket, key string) (*ObjectInfo, error) {
	bkt, err := c.resolveBucket(bucket)
	if err != nil {
		return nil, err
	}
	out, err := c.raw.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bkt),
		Key:    aws.String(key),
	})
	if err != nil {
		if isNotFoundErr(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("objstore: head %s/%s: %w", bkt, key, err)
	}
	info := &ObjectInfo{
		Key:         key,
		Size:        derefInt64(out.ContentLength),
		ETag:        derefStr(out.ETag),
		ContentType: derefStr(out.ContentType),
	}
	if out.LastModified != nil {
		info.LastModified = *out.LastModified
	}
	return info, nil
}

// Exists is a thin wrapper over Head that swallows ErrNotFound.
func (c *Client) Exists(ctx context.Context, bucket, key string) (bool, error) {
	_, err := c.Head(ctx, bucket, key)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	return false, err
}

// Delete removes a single object. Deleting a missing key is a no-op (no error).
func (c *Client) Delete(ctx context.Context, bucket, key string) error {
	bkt, err := c.resolveBucket(bucket)
	if err != nil {
		return err
	}
	_, err = c.raw.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bkt),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("objstore: delete %s/%s: %w", bkt, key, err)
	}
	return nil
}

// ListOptions controls List behaviour.
type ListOptions struct {
	Prefix string
	// MaxKeys caps the number of returned objects. 0 = "all" (paginates internally).
	MaxKeys int32
}

// List walks objects under prefix, paginating as needed. The supplied fn is
// invoked per object; return false to stop iteration early.
func (c *Client) List(ctx context.Context, bucket string, opts ListOptions, fn func(ObjectInfo) bool) error {
	bkt, err := c.resolveBucket(bucket)
	if err != nil {
		return err
	}
	in := &s3.ListObjectsV2Input{
		Bucket: aws.String(bkt),
	}
	if opts.Prefix != "" {
		in.Prefix = aws.String(opts.Prefix)
	}
	if opts.MaxKeys > 0 {
		in.MaxKeys = aws.Int32(opts.MaxKeys)
	}
	p := s3.NewListObjectsV2Paginator(c.raw, in)
	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("objstore: list %s: %w", bkt, err)
		}
		for _, o := range page.Contents {
			info := ObjectInfo{
				Key:  derefStr(o.Key),
				Size: derefInt64(o.Size),
				ETag: derefStr(o.ETag),
			}
			if o.LastModified != nil {
				info.LastModified = *o.LastModified
			}
			if !fn(info) {
				return nil
			}
		}
	}
	return nil
}

// PresignGet returns a presigned URL good for `ttl` that lets anyone with the
// URL GET the object without further credentials.
func (c *Client) PresignGet(ctx context.Context, bucket, key string, ttl time.Duration) (string, error) {
	bkt, err := c.resolveBucket(bucket)
	if err != nil {
		return "", err
	}
	ps := s3.NewPresignClient(c.raw)
	req, err := ps.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bkt),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("objstore: presign get: %w", err)
	}
	return req.URL, nil
}

// PresignPut returns a presigned URL good for `ttl` that lets anyone with the
// URL PUT a new object.
func (c *Client) PresignPut(ctx context.Context, bucket, key string, ttl time.Duration) (string, error) {
	bkt, err := c.resolveBucket(bucket)
	if err != nil {
		return "", err
	}
	ps := s3.NewPresignClient(c.raw)
	req, err := ps.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bkt),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("objstore: presign put: %w", err)
	}
	return req.URL, nil
}

// ErrNotFound is returned when the requested key does not exist.
var ErrNotFound = errors.New("objstore: not found")

func (c *Client) resolveBucket(b string) (string, error) {
	if b != "" {
		return b, nil
	}
	if c.bucket == "" {
		return "", errors.New("objstore: bucket is required (no DefaultBucket configured)")
	}
	return c.bucket, nil
}

func isNotFoundErr(err error) bool {
	var nsk *types.NoSuchKey
	if errors.As(err, &nsk) {
		return true
	}
	var notFound *types.NotFound
	if errors.As(err, &notFound) {
		return true
	}
	// Some S3-compatible servers (and HeadObject on AWS) report errors without
	// the typed shape. Fall back to the smithy generic code check.
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NoSuchKey", "NotFound", "404":
			return true
		}
	}
	return false
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func derefInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}
