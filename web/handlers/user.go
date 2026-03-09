package handlers

import (
	"time"

	"podcast/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// UserHandler 用户相关处理器
type UserHandler struct {
	userService *service.UserService
}

// NewUserHandler 创建用户处理器
func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// Login 用户登录接口
func (h *UserHandler) Login(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		ErrorWithMessage(c, err.Error())
		return
	}

	user, err := h.userService.Authenticate(c.Request.Context(), req.Name, req.Password)
	if err != nil {
		ErrorWithMessage(c, "用户名或密码错误")
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":   user.ID,
		"name": user.Name,
		"exp":  time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString([]byte(h.userService.Token()))
	if err != nil {
		ErrorWithMessage(c, "生成令牌失败")
		return
	}

	Success(c, gin.H{
		"success": true,
		"message": "登录成功",
		"token":   tokenString,
		"user":    gin.H{"id": user.ID, "name": user.Name},
	})
}
