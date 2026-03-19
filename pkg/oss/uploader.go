package oss

import (
	"fmt"
	"io"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// Uploader 文件上传接口
type Uploader interface {
	PutObject(objectName string, reader io.Reader, size int64, contentType string) (string, error)
}

// AliOssUploader 阿里云 OSS 上传实现
type AliOssUploader struct {
	client     *oss.Client
	bucket     *oss.Bucket
	bucketName string
	cdnURL     string
}

// NewAliOssUploader 创建阿里云 OSS 上传器
func NewAliOssUploader(endpoint, accessKeyID, accessKeySecret, bucketName, cdnURL string) (*AliOssUploader, error) {
	client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize oss client: %w", err)
	}

	bucket, err := client.Bucket(bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket %s: %w", bucketName, err)
	}

	return &AliOssUploader{
		client:     client,
		bucket:     bucket,
		bucketName: bucketName,
		cdnURL:     cdnURL,
	}, nil
}

// PutObject 上传对象
func (u *AliOssUploader) PutObject(objectName string, reader io.Reader, size int64, contentType string) (string, error) {
	var opts []oss.Option
	if size >= 0 {
		opts = append(opts, oss.ContentLength(size))
	}
	if contentType != "" {
		opts = append(opts, oss.ContentType(contentType))
	}

	if err := u.bucket.PutObject(objectName, reader, opts...); err != nil {
		return "", fmt.Errorf("failed to put object: %w", err)
	}

	if u.cdnURL != "" {
		return fmt.Sprintf("%s/%s", u.cdnURL, objectName), nil
	}

	return fmt.Sprintf("oss://%s/%s", u.bucketName, objectName), nil
}

// GetBucket 获取 Bucket（用于高级操作）
func (u *AliOssUploader) GetBucket() *oss.Bucket {
	return u.bucket
}
