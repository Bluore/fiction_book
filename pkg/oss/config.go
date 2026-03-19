package oss

// Config OSS 配置
type Config struct {
	Endpoint        string `mapstructure:"endpoint"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret"`
	BucketName      string `mapstructure:"bucket_name"`
	Region          string `mapstructure:"region"`
	RoleARN         string `mapstructure:"role_arn"`
	RoleSession     string `mapstructure:"role_session"`
	CDNURL          string `mapstructure:"cdn_url"`
}

// NewUploaderFromConfig 从配置创建上传器
func NewUploaderFromConfig(cfg *Config) (*AliOssUploader, error) {
	return NewAliOssUploader(
		cfg.Endpoint,
		cfg.AccessKeyID,
		cfg.AccessKeySecret,
		cfg.BucketName,
		cfg.CDNURL,
	)
}

// NewSTSClientFromConfig 从配置创建 STS 客户端
func NewSTSClientFromConfig(cfg *Config) *STSClient {
	roleSession := cfg.RoleSession
	if roleSession == "" {
		roleSession = "gin-template-session"
	}
	return NewSTSClient(
		cfg.Region,
		cfg.AccessKeyID,
		cfg.AccessKeySecret,
		cfg.RoleARN,
		roleSession,
	)
}
