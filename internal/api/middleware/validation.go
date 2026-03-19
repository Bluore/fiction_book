package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/d60-Lab/gin-template/pkg/response"
)

const ValidatedRequestKey = "validatedRequest"

// Validation 通用请求参数绑定与验证中间件
// req: 请求参数结构体的指针，例如 &dto.CreateUserRequest{}
// 该中间件会自动根据请求方法（GET/POST等）选择绑定 Query 或 JSON
// 并自动进行参数校验
func Validation(req interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 创建请求结构体的新实例
		val := reflect.ValueOf(req)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		newReq := reflect.New(val.Type()).Interface()

		// 2. 绑定 URI 参数（如 :id），忽略错误因为字段可能不存在
		//nolint:errcheck
		c.ShouldBindUri(newReq)

		// 3. 绑定 Query 参数，忽略错误因为字段可能不存在
		//nolint:errcheck
		c.ShouldBindQuery(newReq)

		// 4. 对于有 body 的请求，绑定 JSON
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			if c.Request.ContentLength > 0 || c.Request.Method == "POST" {
				if err := c.ShouldBindJSON(newReq); err != nil {
					response.BadRequest(c, processError(err).Error())
					c.Abort()
					return
				}
			}
		}

		// 5. 手动触发验证
		if err := validate(newReq); err != nil {
			response.BadRequest(c, processError(err).Error())
			c.Abort()
			return
		}

		// 6. 将验证后的对象存入上下文
		c.Set(ValidatedRequestKey, newReq)
		c.Next()
	}
}

var validatorInstance = validator.New()

// validate 手动验证结构体
func validate(obj interface{}) error {
	return validatorInstance.Struct(obj)
}

// processError 处理验证错误，返回友好的错误信息
func processError(err error) error {
	var errs validator.ValidationErrors
	if errors.As(err, &errs) {
		for _, e := range errs {
			return fmt.Errorf("参数校验失败: 字段 %s %s", e.Field(), e.Tag())
		}
	}

	var unmarshalTypeError *json.UnmarshalTypeError
	if errors.As(err, &unmarshalTypeError) {
		return fmt.Errorf("类型错误: 字段 %s 期望类型 %s", unmarshalTypeError.Field, unmarshalTypeError.Type.String())
	}

	return err
}

// GetRequest 从上下文中获取已验证的请求对象
func GetRequest[T any](c *gin.Context) (*T, error) {
	val, exists := c.Get(ValidatedRequestKey)
	if !exists {
		return nil, errors.New("request not found in context")
	}
	req, ok := val.(*T)
	if !ok {
		return nil, errors.New("request type mismatch")
	}
	return req, nil
}

// MustGetRequest 从上下文中获取已验证的请求对象，如果不存在或类型不匹配则 panic
func MustGetRequest[T any](c *gin.Context) *T {
	req, err := GetRequest[T](c)
	if err != nil {
		panic(err)
	}
	return req
}
