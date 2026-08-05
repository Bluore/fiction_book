package response

import (
	"errors"
	"fiction-book/pkg/logger"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    data,
	})
}

// Error 错误响应
func Error(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
	})
}

// BadRequest 400 错误请求
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, message)
}

// Unauthorized 401 未授权
func Unauthorized(c *gin.Context) {
	Error(c, http.StatusUnauthorized, "身份验证失败，请重新登录")
}

// Forbidden 403 禁止访问
func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, message)
}

// NotFound 404 未找到
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, message)
}

// InternalError 500 内部错误
func InternalError(c *gin.Context, _ error) {
	// 生产环境不要暴露详细错误信息
	Error(c, http.StatusInternalServerError, "服务器内部出错")
}

type errorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// PushError 格式化推送Error
func PushError(c *gin.Context, err error) {
	errList := getErrorList()
	for key, resp := range *errList {
		if errors.Is(err, key) {
			Error(c, resp.Code, resp.Message)
			logger.Info("response Error:", zap.Any("error", resp))
			return
		}
	}
	logger.Error("response Error:", zap.Error(err))
	InternalError(c, err)
}
