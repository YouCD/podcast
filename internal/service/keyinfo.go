package service

import (
	"context"

	"podcast/internal/database/dao"
	"podcast/internal/database/models"
)

// KeyInfoService KeyInfo 业务服务
type KeyInfoService struct {
	keyInfoDao dao.KeyInfoDao
}

// NewKeyInfoService 创建 KeyInfo 服务
func NewKeyInfoService(keyInfoDao dao.KeyInfoDao) *KeyInfoService {
	return &KeyInfoService{keyInfoDao: keyInfoDao}
}

// Create 创建
func (s *KeyInfoService) Create(ctx context.Context, keyInfo *models.KeyInfo) error {
	return s.keyInfoDao.Create(ctx, keyInfo)
}

// FindAll 获取全部
func (s *KeyInfoService) FindAll(ctx context.Context) ([]*models.KeyInfo, error) {
	return s.keyInfoDao.FindAll(ctx)
}

// FindByID 根据 ID 获取
func (s *KeyInfoService) FindByID(ctx context.Context, id int) (*models.KeyInfo, error) {
	return s.keyInfoDao.FindByID(ctx, id)
}

// FindByKeyName 根据 KeyName 获取
func (s *KeyInfoService) FindByKeyName(ctx context.Context, keyName string) (*models.KeyInfo, error) {
	return s.keyInfoDao.FindByKeyName(ctx, keyName)
}

// FindByGenre 根据 Genre 获取列表
func (s *KeyInfoService) FindByGenre(ctx context.Context, genre int) ([]*models.KeyInfo, error) {
	return s.keyInfoDao.FindByGenre(ctx, genre)
}

// FindByKeynameAndGenre 根据 Keyname 与 Genre 获取
func (s *KeyInfoService) FindByKeynameAndGenre(ctx context.Context, keyname string, genre int) (*models.KeyInfo, error) {
	return s.keyInfoDao.FindByKeynameAndGenre(ctx, keyname, genre)
}

// Update 更新
func (s *KeyInfoService) Update(ctx context.Context, keyInfo *models.KeyInfo) error {
	return s.keyInfoDao.Update(ctx, keyInfo)
}

// Delete 删除
func (s *KeyInfoService) Delete(ctx context.Context, keyInfo *models.KeyInfo) error {
	return s.keyInfoDao.Delete(ctx, keyInfo)
}
