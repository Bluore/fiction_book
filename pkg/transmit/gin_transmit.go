package transmit

import "github.com/gin-gonic/gin"

func GetStr(c *gin.Context, key string) string {
	val, err := c.Get(key)
	if err {
		return ""
	}
	res, ok := val.(string)
	if !ok {
		return ""
	}
	return res
}
