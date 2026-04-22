package catch

import "github.com/redis/go-redis/v9"

type LimitCatch interface {
}
type limitCatch struct {
	redis *redis.Client
}

func NewLimitCatch(redis *redis.Client) LimitCatch {
	return &limitCatch{redis: redis}
}
