package oss

import (
	"fmt"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/sts"
)

// STSClient STS 临时凭证客户端
type STSClient struct {
	region          string
	accessKeyID     string
	accessKeySecret string
	roleArn         string
	roleSession     string
}

// STSCredentials STS 临时凭证
type STSCredentials struct {
	AccessKeyID     string    `json:"access_key_id"`
	AccessKeySecret string    `json:"access_key_secret"`
	SecurityToken   string    `json:"security_token"`
	Expiration      time.Time `json:"expiration"`
}

// NewSTSClient 创建 STS 客户端
func NewSTSClient(region, accessKeyID, accessKeySecret, roleArn, roleSession string) *STSClient {
	return &STSClient{
		region:          region,
		accessKeyID:     accessKeyID,
		accessKeySecret: accessKeySecret,
		roleArn:         roleArn,
		roleSession:     roleSession,
	}
}

// AssumeRole 获取临时访问凭证
// durationSeconds: 临时凭证有效期（秒），范围：900-3600，默认 3600
func (c *STSClient) AssumeRole(durationSeconds int64) (*STSCredentials, error) {
	client, err := sts.NewClientWithAccessKey(
		c.region,
		c.accessKeyID,
		c.accessKeySecret,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create sts client: %w", err)
	}

	req := sts.CreateAssumeRoleRequest()
	req.Scheme = "https"
	req.RoleArn = c.roleArn
	req.RoleSessionName = c.roleSession

	if durationSeconds > 0 {
		if durationSeconds < 900 || durationSeconds > 3600 {
			return nil, fmt.Errorf("durationSeconds must be between 900 and 3600")
		}
		req.DurationSeconds = requests.NewInteger(int(durationSeconds))
	}

	response, err := client.AssumeRole(req)
	if err != nil {
		return nil, fmt.Errorf("failed to assume role: %w", err)
	}

	expiration, err := time.Parse(time.RFC3339, response.Credentials.Expiration)
	if err != nil {
		expiration = time.Now().Add(time.Hour)
	}

	return &STSCredentials{
		AccessKeyID:     response.Credentials.AccessKeyId,
		AccessKeySecret: response.Credentials.AccessKeySecret,
		SecurityToken:   response.Credentials.SecurityToken,
		Expiration:      expiration,
	}, nil
}

// IsExpired 检查凭证是否已过期
func (c *STSCredentials) IsExpired() bool {
	return time.Now().After(c.Expiration)
}

// ExpiresIn 返回凭证还有多久过期
func (c *STSCredentials) ExpiresIn() time.Duration {
	return time.Until(c.Expiration)
}
