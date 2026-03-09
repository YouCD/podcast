package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 定义统一响应结构体
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"` // 使用omitempty避免空数据时返回null
}

// 定义常用状态码常量
const (
	SUCCESS        = 200
	ERROR          = 500
	INVALID_PARAMS = 400
	UNAUTHORIZED   = 401
	FORBIDDEN      = 403
	NOT_FOUND      = 404
)

// Success 成功响应 - 带数据
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    SUCCESS,
		Message: "success",
		Data:    data,
	})
}

// ErrorWithMessage 错误响应 - 自定义错误消息
func ErrorWithMessage(c *gin.Context, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    ERROR,
		Message: message,
		Data:    struct{}{},
	})
}

func SuccessWithMessageCode(c *gin.Context, Code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    Code,
		Message: message,
		Data:    struct{}{},
	})
}
