package dao

import (
	"context"
	"podcast/internal/database/models"

	"gorm.io/gorm"
)

// KeyInfoDao 定义 KeyInfo 数据访问接口
type KeyInfoDao interface {
	// Create 创建新的KeyInfo
	Create(ctx context.Context, keyInfo *models.KeyInfo) error

	// FindAll 获取所有KeyInfo
	FindAll(ctx context.Context) ([]*models.KeyInfo, error)

	// FindByID 根据ID查找KeyInfo
	FindByID(ctx context.Context, id int) (*models.KeyInfo, error)

	// FindByKeyName 根据KeyName查找KeyInfo
	FindByKeyName(ctx context.Context, keyName string) (*models.KeyInfo, error)

	// Update 更新KeyInfo
	Update(ctx context.Context, keyInfo *models.KeyInfo) error

	// Delete 删除KeyInfo
	Delete(ctx context.Context, keyInfo *models.KeyInfo) error

	// FindByGenre 根据Genre查找KeyInfo列表
	FindByGenre(ctx context.Context, genre int) ([]*models.KeyInfo, error)

	// FindByKeynameAndGenre 根据Keyname和Genre查找KeyInfo
	FindByKeynameAndGenre(ctx context.Context, keyname string, genre int) (*models.KeyInfo, error)
}

type keyInfoDao struct {
	db *gorm.DB
}

// NewKeyInfoDao 创建 KeyInfoDao（依赖注入 *gorm.DB）
func NewKeyInfoDao(db *gorm.DB) KeyInfoDao {
	return &keyInfoDao{db: db}
}

// Create 创建新的KeyInfo
// 参数: keyInfo KeyInfo结构体指针
// 返回: 错误信息
func (dao *keyInfoDao) Create(ctx context.Context, keyInfo *models.KeyInfo) error {
	return dao.db.Model(&models.KeyInfo{}).WithContext(ctx).Create(keyInfo).Error
}

// FindAll 获取所有KeyInfo
// 返回: KeyInfo切片和错误信息
func (dao *keyInfoDao) FindAll(ctx context.Context) ([]*models.KeyInfo, error) {
	var keyInfos []*models.KeyInfo
	err := dao.db.Model(&models.KeyInfo{}).WithContext(ctx).Find(&keyInfos).Error
	return keyInfos, err
}

// FindByID 根据ID查找KeyInfo
// 参数: id KeyInfo ID
// 返回: KeyInfo指针和错误信息
func (dao *keyInfoDao) FindByID(ctx context.Context, id int) (*models.KeyInfo, error) {
	var keyInfo models.KeyInfo
	err := dao.db.Model(&models.KeyInfo{}).WithContext(ctx).First(&keyInfo, id).Error
	return &keyInfo, err
}

// FindByKeyName 根据KeyName查找KeyInfo
// 参数: keyName 键名
// 返回: KeyInfo指针和错误信息
func (dao *keyInfoDao) FindByKeyName(ctx context.Context, keyName string) (*models.KeyInfo, error) {
	var keyInfo models.KeyInfo
	err := dao.db.Model(&models.KeyInfo{}).WithContext(ctx).Where("key_name = ?", keyName).First(&keyInfo).Error
	return &keyInfo, err
}

// Update 更新KeyInfo
// 参数: keyInfo KeyInfo结构体指针
// 返回: 错误信息
func (dao *keyInfoDao) Update(ctx context.Context, keyInfo *models.KeyInfo) error {
	return dao.db.Model(&models.KeyInfo{}).WithContext(ctx).Where("key_name = ?", keyInfo.KeyName).Update("data", keyInfo.Data).Error
}

// Delete 删除KeyInfo
// 参数: keyInfo KeyInfo结构体指针
// 返回: 错误信息
func (dao *keyInfoDao) Delete(ctx context.Context, keyInfo *models.KeyInfo) error {
	return dao.db.Model(&models.KeyInfo{}).WithContext(ctx).Delete(keyInfo).Error
}

// FindByGenre 根据Genre查找KeyInfo列表
// 参数: genre 类型
// 返回: KeyInfo切片和错误信息
// 特殊情况: 当没有找到匹配的记录时，返回空切片和nil错误
func (dao *keyInfoDao) FindByGenre(ctx context.Context, genre int) ([]*models.KeyInfo, error) {
	var keyInfos []*models.KeyInfo
	err := dao.db.Model(&models.KeyInfo{}).WithContext(ctx).Where("genre = ?", genre).Find(&keyInfos).Error
	return keyInfos, err
}

// FindByKeynameAndGenre 根据Keyname和Genre查找KeyInfo
// 参数: keyname 键名, genre 类型
// 返回: KeyInfo指针和错误信息
// 特殊情况: 当没有找到匹配的记录时，返回nil和错误信息
//
//nolint:all
func (dao *keyInfoDao) FindByKeynameAndGenre(ctx context.Context, keyname string, genre int) (keyInfo *models.KeyInfo, err error) {
	err = dao.db.Model(&models.KeyInfo{}).WithContext(ctx).Where("key_name = ? AND genre = ?", keyname, genre).Scan(&keyInfo).Error
	return keyInfo, err
}
