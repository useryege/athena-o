package accountavatar

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

var ErrNotFound = errors.New("account avatar object not found")

type Config struct {
	Endpoint     string
	Region       string
	Bucket       string
	AccessKey    string
	SecretKey    string
	UsePathStyle bool
}

func (c Config) Validate() error {
	missing := make([]string, 0, 5)
	if strings.TrimSpace(c.Endpoint) == "" {
		missing = append(missing, "endpoint")
	}
	if strings.TrimSpace(c.Region) == "" {
		missing = append(missing, "region")
	}
	if strings.TrimSpace(c.Bucket) == "" {
		missing = append(missing, "bucket")
	}
	if strings.TrimSpace(c.AccessKey) == "" {
		missing = append(missing, "access key")
	}
	if strings.TrimSpace(c.SecretKey) == "" {
		missing = append(missing, "secret key")
	}
	if len(missing) != 0 {
		return fmt.Errorf("account avatar S3 configuration is missing %s", strings.Join(missing, ", "))
	}

	endpoint, err := url.Parse(c.Endpoint)
	if err != nil {
		return fmt.Errorf("parse account avatar S3 endpoint: %w", err)
	}
	if endpoint.Scheme != "http" && endpoint.Scheme != "https" {
		return fmt.Errorf("account avatar S3 endpoint scheme must be http or https")
	}
	if endpoint.Host == "" {
		return fmt.Errorf("account avatar S3 endpoint must include a host")
	}
	if endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return fmt.Errorf("account avatar S3 endpoint must not include credentials, a query, or a fragment")
	}
	if endpoint.Path != "" && endpoint.Path != "/" {
		return fmt.Errorf("account avatar S3 endpoint must not include a path")
	}
	if err := validateBucket(c.Bucket); err != nil {
		return err
	}
	return nil
}

type ObjectInfo struct {
	Key          string
	ETag         string
	ContentType  string
	Size         int64
	LastModified time.Time
}

type Object struct {
	ObjectInfo
	Body io.ReadCloser
}

type Store struct {
	client *s3.Client
	bucket string
}

func NewStore(ctx context.Context, cfg Config) (*Store, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion(strings.TrimSpace(cfg.Region)),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			strings.TrimSpace(cfg.AccessKey),
			cfg.SecretKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("load account avatar S3 configuration: %w", err)
	}

	endpoint := strings.TrimRight(strings.TrimSpace(cfg.Endpoint), "/")
	client := s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(endpoint)
		options.UsePathStyle = cfg.UsePathStyle
	})

	return &Store{client: client, bucket: strings.TrimSpace(cfg.Bucket)}, nil
}

func (s *Store) Put(
	ctx context.Context,
	key string,
	body io.Reader,
	size int64,
	contentType string,
) (ObjectInfo, error) {
	if err := validateObjectKey(key); err != nil {
		return ObjectInfo{}, err
	}
	if body == nil {
		return ObjectInfo{}, fmt.Errorf("account avatar object body is required")
	}
	if size < 0 {
		return ObjectInfo{}, fmt.Errorf("account avatar object size must not be negative")
	}
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		return ObjectInfo{}, fmt.Errorf("account avatar object content type is required")
	}

	output, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          body,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(contentType),
	})
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("put account avatar object %q: %w", key, err)
	}

	return ObjectInfo{
		Key:         key,
		ETag:        normalizeETag(aws.ToString(output.ETag)),
		ContentType: contentType,
		Size:        size,
	}, nil
}

func (s *Store) Get(ctx context.Context, key string) (*Object, error) {
	if err := validateObjectKey(key); err != nil {
		return nil, err
	}

	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, objectError("get", key, err)
	}

	return &Object{
		ObjectInfo: ObjectInfo{
			Key:          key,
			ETag:         normalizeETag(aws.ToString(output.ETag)),
			ContentType:  aws.ToString(output.ContentType),
			Size:         aws.ToInt64(output.ContentLength),
			LastModified: aws.ToTime(output.LastModified),
		},
		Body: output.Body,
	}, nil
}

func (s *Store) Stat(ctx context.Context, key string) (ObjectInfo, error) {
	if err := validateObjectKey(key); err != nil {
		return ObjectInfo{}, err
	}

	output, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return ObjectInfo{}, objectError("stat", key, err)
	}

	return ObjectInfo{
		Key:          key,
		ETag:         normalizeETag(aws.ToString(output.ETag)),
		ContentType:  aws.ToString(output.ContentType),
		Size:         aws.ToInt64(output.ContentLength),
		LastModified: aws.ToTime(output.LastModified),
	}, nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	if err := validateObjectKey(key); err != nil {
		return err
	}

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete account avatar object %q: %w", key, err)
	}
	return nil
}

func (s *Store) List(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	if err := validatePrefix(prefix); err != nil {
		return nil, err
	}

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})
	objects := make([]ObjectInfo, 0)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list account avatar objects with prefix %q: %w", prefix, err)
		}
		for _, item := range page.Contents {
			objects = append(objects, ObjectInfo{
				Key:          aws.ToString(item.Key),
				ETag:         normalizeETag(aws.ToString(item.ETag)),
				Size:         aws.ToInt64(item.Size),
				LastModified: aws.ToTime(item.LastModified),
			})
		}
	}
	return objects, nil
}

func objectError(operation, key string, err error) error {
	var apiError smithy.APIError
	if errors.As(err, &apiError) {
		switch apiError.ErrorCode() {
		case "NoSuchKey", "NotFound", "NoSuchObject":
			return fmt.Errorf("%w: %s", ErrNotFound, key)
		}
	}
	return fmt.Errorf("%s account avatar object %q: %w", operation, key, err)
}

func validateBucket(bucket string) error {
	bucket = strings.TrimSpace(bucket)
	if len(bucket) < 3 || len(bucket) > 63 {
		return fmt.Errorf("account avatar S3 bucket must contain 3 to 63 characters")
	}
	if bucket[0] == '.' || bucket[0] == '-' || bucket[len(bucket)-1] == '.' || bucket[len(bucket)-1] == '-' {
		return fmt.Errorf("account avatar S3 bucket must begin and end with a letter or digit")
	}
	for _, char := range bucket {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '.' || char == '-' {
			continue
		}
		return fmt.Errorf("account avatar S3 bucket may contain only lowercase letters, digits, dots, and hyphens")
	}
	if strings.Contains(bucket, "..") || strings.Contains(bucket, ".-") || strings.Contains(bucket, "-.") {
		return fmt.Errorf("account avatar S3 bucket contains an invalid dot or hyphen sequence")
	}
	return nil
}

func validateObjectKey(key string) error {
	if key == "" {
		return fmt.Errorf("account avatar object key is required")
	}
	if len(key) > 1024 {
		return fmt.Errorf("account avatar object key exceeds 1024 bytes")
	}
	if strings.HasPrefix(key, "/") || strings.HasSuffix(key, "/") || strings.Contains(key, "\\") {
		return fmt.Errorf("account avatar object key must be a relative slash-separated key")
	}
	for _, segment := range strings.Split(key, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return fmt.Errorf("account avatar object key contains an invalid path segment")
		}
	}
	for _, char := range key {
		if unicode.IsControl(char) {
			return fmt.Errorf("account avatar object key must not contain control characters")
		}
	}
	return nil
}

func validatePrefix(prefix string) error {
	if len(prefix) > 1024 {
		return fmt.Errorf("account avatar object prefix exceeds 1024 bytes")
	}
	if strings.HasPrefix(prefix, "/") || strings.Contains(prefix, "\\") {
		return fmt.Errorf("account avatar object prefix must be relative")
	}
	for _, char := range prefix {
		if unicode.IsControl(char) {
			return fmt.Errorf("account avatar object prefix must not contain control characters")
		}
	}
	return nil
}

func normalizeETag(etag string) string {
	return strings.Trim(etag, "\"")
}
