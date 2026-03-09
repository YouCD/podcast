package service

import (
	"context"

	"podcast/internal/database/models"
)

const templateGenre = 2

// TemplateService Template 业务服务（Genre=2 的 KeyInfo）
type TemplateService struct {
	keyInfo *KeyInfoService
}

// NewTemplateService 创建 Template 服务
func NewTemplateService(keyInfo *KeyInfoService) *TemplateService {
	return &TemplateService{keyInfo: keyInfo}
}

// Create 创建 Template（强制 Genre=2）
func (s *TemplateService) Create(ctx context.Context, template *models.KeyInfo) error {
	template.Genre = templateGenre
	return s.keyInfo.Create(ctx, template)
}

// FindAll 获取所有 Template
func (s *TemplateService) FindAll(ctx context.Context) ([]*models.KeyInfo, error) {
	return s.keyInfo.FindByGenre(ctx, templateGenre)
}

// FindByID 根据 ID 获取，并校验为 Template 类型
func (s *TemplateService) FindByID(ctx context.Context, id int) (*models.KeyInfo, error) {
	template, err := s.keyInfo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if template.Genre != templateGenre {
		return nil, errNotTemplate
	}
	return template, nil
}

// FindByKeyname 根据 keyname 获取
func (s *TemplateService) FindByKeyname(ctx context.Context, keyname string) (*models.KeyInfo, error) {
	return s.keyInfo.FindByKeynameAndGenre(ctx, keyname, templateGenre)
}

// Update 更新（强制 Genre=2）
func (s *TemplateService) Update(ctx context.Context, template *models.KeyInfo) error {
	template.Genre = templateGenre
	return s.keyInfo.Update(ctx, template)
}

// Delete 删除（需先校验为 Template 类型）
func (s *TemplateService) Delete(ctx context.Context, id int) error {
	template, err := s.FindByID(ctx, id)
	if err != nil {
		return err
	}
	return s.keyInfo.Delete(ctx, template)
}
