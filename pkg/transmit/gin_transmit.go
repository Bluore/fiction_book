package transmit

import "github.com/gin-gonic/gin"

func GetStr(c *gin.Context, key string) string {
	val, ok := c.Get(key)
	if !ok {
		return ""
	}
	res, ok := val.(string)
	if !ok {
		return ""
	}
	return res
}
