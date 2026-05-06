package constant

import (
	"fmt"
	"time"
)

const (
	redisBookThermalDaily        = "Rank:book:thermal:Daily:%s"
	redisBookThermalDay          = "Rank:book:thermal:%s"
	RedisBookThermalDailyTotal   = "Rank:book:thermal:Daily:total"
	RedisBookThermalWeeklyTotal  = "Rank:book:thermal:Weekly:total"
	RedisBookThermalMonthlyTotal = "Rank:book:thermal:Monthly:total"
)

func RedisBookThermalDaily(time time.Time) string {
	return fmt.Sprintf(redisBookThermalDaily, time.Format("2006-01-02"))
}

func RedisBookThermalDay(time time.Time) string {
	return fmt.Sprintf(redisBookThermalDay, time.Format("2006-01-02"))
}
