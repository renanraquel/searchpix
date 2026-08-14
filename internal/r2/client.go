package r2

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Config credenciais e endpoint do Cloudflare R2.
type Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	PublicBaseURL   string
	Endpoint        string
}

// Client cliente S3-compatible para R2.
type Client struct {
	s3     *s3.Client
	bucket string
	public string
}

// NewFromEnv monta o client a partir da config. Retorna nil se R2 não estiver configurado.
func New(cfg Config) (*Client, error) {
	if !cfg.Configured() {
		return nil, nil
	}
	endpoint := strings.TrimRight(strings.TrimSpace(cfg.Endpoint), "/")
	if endpoint == "" {
		endpoint = fmt.Sprintf("https://%s.r2.cloudflarestorage.com", strings.TrimSpace(cfg.AccountID))
	}
	public := strings.TrimRight(strings.TrimSpace(cfg.PublicBaseURL), "/")
	awsCfg := aws.Config{
		Region: "auto",
		Credentials: credentials.NewStaticCredentialsProvider(
			strings.TrimSpace(cfg.AccessKeyID),
			strings.TrimSpace(cfg.SecretAccessKey),
			"",
		),
	}
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})
	return &Client{
		s3:     client,
		bucket: strings.TrimSpace(cfg.Bucket),
		public: public,
	}, nil
}

func (c Config) Configured() bool {
	return strings.TrimSpace(c.AccessKeyID) != "" &&
		strings.TrimSpace(c.SecretAccessKey) != "" &&
		strings.TrimSpace(c.Bucket) != "" &&
		strings.TrimSpace(c.PublicBaseURL) != "" &&
		(strings.TrimSpace(c.Endpoint) != "" || strings.TrimSpace(c.AccountID) != "")
}

func (c *Client) PublicURL(key string) string {
	key = strings.TrimLeft(key, "/")
	return c.public + "/" + key
}

// PutObject envia bytes ao R2 e retorna a key e a URL pública.
func (c *Client) PutObject(ctx context.Context, key string, data []byte, contentType string) (string, string, error) {
	if c == nil {
		return "", "", fmt.Errorf("cliente R2 não configurado")
	}
	key = strings.TrimLeft(key, "/")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	_, err := c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(c.bucket),
		Key:          aws.String(key),
		Body:         bytes.NewReader(data),
		ContentType:  aws.String(contentType),
		CacheControl: aws.String("public, max-age=31536000, immutable"),
	})
	if err != nil {
		return "", "", fmt.Errorf("upload R2: %w", err)
	}
	return key, c.PublicURL(key), nil
}

// DeleteObject remove um objeto do R2 (best effort).
func (c *Client) DeleteObject(ctx context.Context, key string) error {
	if c == nil || strings.TrimSpace(key) == "" {
		return nil
	}
	key = strings.TrimLeft(key, "/")
	_, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	return err
}

// CarouselObjectKey monta a chave do objeto no bucket.
func CarouselObjectKey(tenantID, itemID, contentType string) string {
	ext := extensionForContentType(contentType)
	return path.Join("carousel", tenantID, itemID+ext)
}

func ProductObjectKey(tenantID, productID, contentType string) string {
	ext := extensionForContentType(contentType)
	return path.Join("products", tenantID, productID+ext)
}

func TenantBackgroundKey(tenantID, contentType string) string {
	ext := extensionForContentType(contentType)
	return path.Join("tenants", tenantID, "background"+ext)
}

func extensionForContentType(contentType string) string {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	switch {
	case strings.Contains(ct, "png"):
		return ".png"
	case strings.Contains(ct, "webp"):
		return ".webp"
	case strings.Contains(ct, "gif"):
		return ".gif"
	case strings.Contains(ct, "jpeg"), strings.Contains(ct, "jpg"):
		return ".jpg"
	case strings.Contains(ct, "mp4"):
		return ".mp4"
	case strings.Contains(ct, "webm"):
		return ".webm"
	case strings.HasPrefix(ct, "video/"):
		return ".mp4"
	case strings.HasPrefix(ct, "image/"):
		return ".jpg"
	default:
		return ".bin"
	}
}

// UploadWithTimeout helper com timeout padrão.
func (c *Client) UploadWithTimeout(key string, data []byte, contentType string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	return c.PutObject(ctx, key, data, contentType)
}
