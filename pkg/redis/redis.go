package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"fiction-book/pkg/config"
)

var (
	client *redis.Client
)

// Init 初始化 Redis 客户端（全局）
func Init(cfg *config.RedisConfig) error {
	c, err := NewClient(cfg)
	if err != nil {
		return err
	}
	client = c
	return nil
}

// NewClient 创建 Redis 客户端
func NewClient(cfg *config.RedisConfig) (*redis.Client, error) {
	c := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return c, nil
}

// GetClient 获取全局客户端
func GetClient() (*redis.Client, error) {
	if client == nil {
		return nil, errors.New("redis client not initialized")
	}
	return client, nil
}

// Set 设置键值对，默认永不过期
func Set(ctx context.Context, key string, value interface{}) error {
	return SetWithExpiration(ctx, key, value, 0)
}

// SetWithExpiration 设置键值对，可指定过期时间
// expiration 为 0 时表示永不过期
func SetWithExpiration(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if client == nil {
		return errors.New("redis client not initialized")
	}
	return client.Set(ctx, key, value, expiration).Err()
}

// Get 获取值
func Get(ctx context.Context, key string) (string, error) {
	if client == nil {
		return "", errors.New("redis client not initialized")
	}
	return client.Get(ctx, key).Result()
}

// Delete 删除键
func Delete(ctx context.Context, keys ...string) error {
	if client == nil {
		return errors.New("redis client not initialized")
	}
	return client.Del(ctx, keys...).Err()
}

// Exists 检查键是否存在
func Exists(ctx context.Context, key string) (bool, error) {
	if client == nil {
		return false, errors.New("redis client not initialized")
	}
	n, err := client.Exists(ctx, key).Result()
	return n > 0, err
}

// Close 关闭连接
func Close() error {
	if client == nil {
		return nil
	}
	return client.Close()
}
