package constant

import (
	"fmt"
	"time"
)

const (
	redisBookThermalDaily      = "Rank:book:thermal:daily:%s"
	RedisBookThermalDailyTotal = "Rank:book:thermal:daily:total"
)

func RedisBookThermalDaily(time time.Time) string {
	return fmt.Sprintf(redisBookThermalDaily, time.Format("2006-01-02"))
}
