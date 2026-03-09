package dao

import (
	"context"
	"testing"

	"podcast/config"
	"podcast/internal/database/models"

	"github.com/stretchr/testify/assert"
)

func TestUserDao(t *testing.T) {
	cfg, err := config.LoadAppConfig("config.yaml")
	if err != nil {
		t.Skip("config not found, skip test")
		return
	}
	_, _ = models.Init(cfg)
	ctx := context.Background()
	userDao := NewUserDao(models.GetDb())

	// 测试创建用户
	user := &models.User{
		Name:     "admin",
		Password: "admin",
	}

	err = userDao.Create(ctx, user, "youcd")
	assert.NoError(t, err)
	assert.NotZero(t, user.ID)
	//
	//// 测试根据ID查找用户
	//foundUser, err := userDao.FindByID(ctx, user.ID)
	//assert.NoError(t, err)
	//assert.Equal(t, user.Name, foundUser.Name)
	//
	//// 测试根据名称查找用户
	//foundUserByName, err := userDao.FindByName(ctx, "test_user")
	//assert.NoError(t, err)
	//assert.Equal(t, user.ID, foundUserByName.ID)
	//
	//// 测试更新用户
	//user.Name = "updated_user"
	//err = userDao.Update(ctx, user)
	//assert.NoError(t, err)
	//
	//// 验证更新后的用户
	//updatedUser, err := userDao.FindByID(ctx, user.ID)
	//assert.NoError(t, err)
	//assert.Equal(t, "updated_user", updatedUser.Name)
	//
	//// 测试用户认证
	//authUser, err := userDao.Authenticate(ctx, "updated_user", "test_password")
	//assert.NoError(t, err)
	//assert.Equal(t, user.ID, authUser.ID)
	//
	//// 测试错误的密码认证
	//_, err = userDao.Authenticate(ctx, "updated_user", "wrong_password")
	//assert.Error(t, err)
	//
	//// 清理测试数据
	//err = userDao.Delete(ctx, user)
	//assert.NoError(t, err)
}
