package cache

import (
	"fiction-book/pkg/constant"
	"time"
)

// TimeShakeDuration 时间抖动附加数值，抖动 0~10%
func TimeShakeDuration(duration time.Duration) time.Duration {
	randInt := constant.RandQuick.IntN(100)
	var timeRange time.Duration = duration / 1000
	return time.Duration(randInt) * timeRange
}
