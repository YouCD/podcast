package service

import (
	"context"

	"podcast/internal/database/dao"
	"podcast/internal/database/models"
)

// UserService 用户业务服务
type UserService struct {
	userDao dao.UserDao
	token   string
}

// NewUserService 创建用户服务
func NewUserService(userDao dao.UserDao, token string) *UserService {
	return &UserService{userDao: userDao, token: token}
}

// Authenticate 验证用户名密码，返回用户信息
func (s *UserService) Authenticate(ctx context.Context, name, password string) (*models.User, error) {
	return s.userDao.Authenticate(ctx, name, password, s.token)
}

// FindByID 根据用户ID查询
func (s *UserService) FindByID(ctx context.Context, id int) (*models.User, error) {
	return s.userDao.FindByID(ctx, id)
}

// Token 返回用于 JWT 签名的密钥（与 Authenticate 使用的 salt 一致）
func (s *UserService) Token() string { return s.token }
