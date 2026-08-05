package constant

import (
	"fmt"
	"time"
)

const (
	redisBookThermalDaily        = "GSF:book:rank:thermal:%s:daily"
	redisBookThermalDay          = "GSF:book:rank:thermal:%s:history"
	RedisBookThermalDailyTotal   = "GSF:book:rank:thermal:total:daily"
	RedisBookThermalWeeklyTotal  = "GSF:book:rank:thermal:total:weekly"
	RedisBookThermalMonthlyTotal = "GSF:book:rank:thermal:total:monthly"
)

func RedisBookThermalDaily(time time.Time) string {
	return fmt.Sprintf(redisBookThermalDaily, time.Format("2006-01-02"))
}

func RedisBookThermalDay(time time.Time) string {
	return fmt.Sprintf(redisBookThermalDay, time.Format("2006-01-02"))
}
