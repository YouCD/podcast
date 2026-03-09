package dao

import (
	"context"
	"crypto/sha256"
	"fmt"
	"podcast/internal/database/models"

	"gorm.io/gorm"
)

// UserDao 定义 User 数据访问接口
type UserDao interface {
	// Create 创建新的User，会自动对密码进行加盐处理
	Create(ctx context.Context, user *models.User, salt string) error

	// FindAll 获取所有User
	FindAll(ctx context.Context) ([]*models.User, error)

	// FindByID 根据ID查找User
	FindByID(ctx context.Context, id int) (*models.User, error)

	// FindByName 根据名称查找User
	FindByName(ctx context.Context, name string) (*models.User, error)

	// Update 更新User
	Update(ctx context.Context, user *models.User, slat string) error

	// Delete 删除User
	Delete(ctx context.Context, user *models.User) error

	// Authenticate 用户身份验证
	Authenticate(ctx context.Context, name, password, slat string) (*models.User, error)
}

type userDao struct {
	db *gorm.DB
}

// NewUserDao 创建 UserDao（依赖注入 *gorm.DB）
func NewUserDao(db *gorm.DB) UserDao {
	return &userDao{db: db}
}

// hashPassword 对密码进行加盐哈希
func (dao *userDao) hashPassword(password, salt string) string {
	data := []byte(password + salt)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// Create 创建新的User，会自动对密码进行加盐处理
func (dao *userDao) Create(ctx context.Context, user *models.User, salt string) error {
	// 对密码进行加盐哈希
	user.Password = dao.hashPassword(user.Password, salt)
	return dao.db.Model(&models.User{}).WithContext(ctx).Create(user).Error
}

// FindAll 获取所有User
func (dao *userDao) FindAll(ctx context.Context) ([]*models.User, error) {
	var users []*models.User
	err := dao.db.Model(&models.User{}).WithContext(ctx).Find(&users).Error
	return users, err
}

// FindByID 根据ID查找User
func (dao *userDao) FindByID(ctx context.Context, id int) (*models.User, error) {
	var user models.User
	err := dao.db.Model(&models.User{}).WithContext(ctx).First(&user, id).Error
	return &user, err
}

// FindByName 根据名称查找User
func (dao *userDao) FindByName(ctx context.Context, name string) (*models.User, error) {
	var user models.User
	err := dao.db.Model(&models.User{}).WithContext(ctx).Where("name = ?", name).First(&user).Error
	return &user, err
}

// Update 更新User
func (dao *userDao) Update(ctx context.Context, user *models.User, slat string) error {
	// 如果更新了密码，则对新密码进行加盐哈希
	if user.Password != "" {
		user.Password = dao.hashPassword(user.Password, slat)
	}
	return dao.db.Model(&models.User{}).WithContext(ctx).Where("name = ?", user.Name).Save(user).Error
}

// Delete 删除User
func (dao *userDao) Delete(ctx context.Context, user *models.User) error {
	return dao.db.Model(&models.User{}).WithContext(ctx).Delete(user).Error
}

// Authenticate 用户身份验证
func (dao *userDao) Authenticate(ctx context.Context, name, password, slat string) (*models.User, error) {
	hashedPassword := dao.hashPassword(password, slat)
	var user models.User
	err := dao.db.Model(&models.User{}).WithContext(ctx).Where("name = ? AND password = ?", name, hashedPassword).First(&user).Error
	return &user, err
}
